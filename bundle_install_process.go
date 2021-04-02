package bundleinstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) error
}

//go:generate faux --interface VersionResolver --output fakes/version_resolver.go
type VersionResolver interface {
	Lookup() (version string, err error)
	CompareMajorMinor(left string, right string) (bool, error)
}

//go:generate faux --interface Calculator --output fakes/calculator.go
type Calculator interface {
	Sum(paths ...string) (string, error)
}

type BundleInstallProcess struct {
	executable      Executable
	logger          LogEmitter
	versionResolver VersionResolver
	calculator      Calculator
}

func NewBundleInstallProcess(executable Executable, logger LogEmitter, versionResolver VersionResolver, calculator Calculator) BundleInstallProcess {
	return BundleInstallProcess{
		executable:      executable,
		logger:          logger,
		versionResolver: versionResolver,
		calculator:      calculator,
	}
}

// ShouldRun will return true if it is determined that the BundleInstallProcess
// be executed during the build phase.
//
// The criteria for determining that the install process should be executed is
// if the major or minor version of Ruby has changed, or if the contents of the
// Gemfile or Gemfile.lock have changed.
//
// In addition to reporting if the install process should execute, this method
// will return the current version of Ruby and the checksum of the Gemfile and
// Gemfile.lock contents.
func (ip BundleInstallProcess) ShouldRun(layer packit.Layer, workingDir string) (bool, string, string, error) {
	rubyVersion, err := ip.versionResolver.Lookup()
	if err != nil {
		return false, "", "", err
	}

	cachedRubyVersion, ok := layer.Metadata["ruby_version"].(string)
	rubyVersionMatch := true

	if ok {
		rubyVersionMatch, err = ip.versionResolver.CompareMajorMinor(cachedRubyVersion, rubyVersion)
		if err != nil {
			return false, "", "", err
		}
	}

	var sum string
	_, err = os.Stat(filepath.Join(workingDir, "Gemfile.lock"))
	if err != nil {
		if !os.IsNotExist(err) {
			return false, "", "", err
		}
	} else {
		sum, err = ip.calculator.Sum(filepath.Join(workingDir, "Gemfile"), filepath.Join(workingDir, "Gemfile.lock"))
		if err != nil {
			return false, "", "", err
		}
	}

	cachedSHA, ok := layer.Metadata["cache_sha"].(string)
	cacheMatch := ok && cachedSHA == sum
	shouldRun := !cacheMatch || !rubyVersionMatch

	return shouldRun, sum, rubyVersion, nil
}

func (ip BundleInstallProcess) Execute(workingDir, layerPath string, config map[string]string) error {
	localConfigPath := filepath.Join(workingDir, ".bundle", "config")
	globalConfigPath := filepath.Join(layerPath, "config")
	if _, err := os.Stat(localConfigPath); err == nil {
		err := os.MkdirAll(layerPath, os.ModePerm)
		if err != nil {
			return err
		}

		err = fs.Copy(localConfigPath, globalConfigPath)
		if err != nil {
			return err
		}
	}

	env := append(os.Environ(), fmt.Sprintf("BUNDLE_USER_CONFIG=%s", globalConfigPath))

	var keys []string
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		buffer := bytes.NewBuffer(nil)
		args := []string{"config", "--global", key, config[key]}

		ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))

		err := ip.executable.Execute(pexec.Execution{
			Args:   args,
			Stdout: buffer,
			Stderr: buffer,
			Env:    env,
		})
		if err != nil {
			return fmt.Errorf("failed to execute bundle config output:\n%s\nerror: %s", buffer.String(), err)
		}
	}

	buffer := bytes.NewBuffer(nil)
	args := []string{"config", "--global", "cache_path", "--parseable"}

	ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))

	err := ip.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: buffer,
		Stderr: buffer,
		Env:    env,
	})
	if err != nil {
		return fmt.Errorf("failed to execute bundle config output:\n%s\nerror: %s", buffer.String(), err)
	}

	cachePath := filepath.Join("vendor", "cache")

	if buffer.String() != "" {
		// output is in the form: cache_path=path/to/custom/cache
		cachePathRaw := (strings.SplitN(buffer.String(), "=", 2))[1]
		cachePath = strings.Trim(cachePathRaw, "\n")
	}

	buffer = bytes.NewBuffer(nil)
	args = []string{"install"}

	_, err = os.Stat(filepath.Join(workingDir, cachePath))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		args = append(args, "--local")
	}

	ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))
	err = ip.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: buffer,
		Stderr: buffer,
		Env:    env,
	})
	if err != nil {
		return fmt.Errorf("failed to execute bundle install output:\n%s\nerror: %s", buffer.String(), err)
	}

	return nil
}
