package bundleinstall

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	pfs "github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go
//go:generate faux --interface VersionResolver --output fakes/version_resolver.go
//go:generate faux --interface Calculator --output fakes/calculator.go

// Executable defines the interface for executing an external process.
type Executable interface {
	Execute(pexec.Execution) error
}

// VersionResolver defines the interface for looking up and comparing the
// versions of Ruby installed in the environment.
type VersionResolver interface {
	Lookup() (version string, err error)
	CompareMajorMinor(left string, right string) (bool, error)
}

// Calculator defines the interface for calculating a checksum of the given set
// of file paths.
type Calculator interface {
	Sum(paths ...string) (string, error)
}

// BundleInstallProcess performs the "bundle install" build process.
type BundleInstallProcess struct {
	executable      Executable
	logger          scribe.Emitter
	versionResolver VersionResolver
	calculator      Calculator
}

// NewBundleInstallProcess initializes an instance of BundleInstallProcess.
func NewBundleInstallProcess(executable Executable, logger scribe.Emitter, versionResolver VersionResolver, calculator Calculator) BundleInstallProcess {
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
func (ip BundleInstallProcess) ShouldRun(metadata map[string]interface{}, workingDir string) (bool, string, string, error) {
	rubyVersion, err := ip.versionResolver.Lookup()
	if err != nil {
		return false, "", "", err
	}

	cachedRubyVersion, ok := metadata["ruby_version"].(string)
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

	cachedSHA, ok := metadata["cache_sha"].(string)
	cacheMatch := ok && cachedSHA == sum
	shouldRun := !cacheMatch || !rubyVersionMatch

	return shouldRun, sum, rubyVersion, nil
}

// Execute will configure and install a set of gems into a layer location using
// the Bundler CLI.
//
// First, to configure the Bundler environment, Execute will copy the local
// Bundler configuration, if any, into the target layer path. The configuration
// file created in the layer will become the defacto configuration file by
// setting `BUNDLE_USER_CONFIG` in the local environment while executing the
// subsequent Bundle CLI commands. The configuration will then be modifed with
// any settings specific to the invocation of Execute.  These configurations
// will override any settings previously applied in the local Bundle
// configuration.
//
// Once fully configured, Execute will run "bundle install" as a child process.
// During the execution of the "bundle install" process, Execute will have
// configured the command to use any locally vendored cache, enabling offline
// execution.
func (ip BundleInstallProcess) Execute(workingDir, layerPath string, config map[string]string, keepBuildFiles bool) error {
	ip.logger.Debug.Subprocess("Setting up bundle install config paths:")

	localConfigPath := filepath.Join(workingDir, ".bundle", "config")
	backupConfigPath := filepath.Join(workingDir, ".bundle", "config.bak")
	globalConfigPath := filepath.Join(layerPath, "config")

	ip.logger.Debug.Subprocess("  Local config path: %s", localConfigPath)
	ip.logger.Debug.Subprocess("  Backup config path: %s", backupConfigPath)
	ip.logger.Debug.Subprocess("  Global config path: %s", globalConfigPath)

	err := os.RemoveAll(globalConfigPath)
	if err != nil {
		return err
	}

	err = os.MkdirAll(layerPath, os.ModePerm)
	if err != nil {
		return err
	}

	if _, err := os.Stat(localConfigPath); err == nil {
		if _, err := os.Stat(backupConfigPath); err == nil {
			err = pfs.Copy(backupConfigPath, localConfigPath)
			if err != nil {
				return err
			}
		}

		err = pfs.Copy(localConfigPath, globalConfigPath)
		if err != nil {
			return err
		}

		err = pfs.Copy(localConfigPath, backupConfigPath)
		if err != nil {
			return err
		}
	}

	ip.logger.Debug.Subprocess("Adding global config path to $BUNDLE_USER_CONFIG")
	ip.logger.Debug.Break()
	env := append(os.Environ(), fmt.Sprintf("BUNDLE_USER_CONFIG=%s", globalConfigPath))

	var keys []string
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		args := []string{"config", "set", "--global", key, config[key]}

		ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))

		err := ip.executable.Execute(pexec.Execution{
			Args:   args,
			Stdout: ip.logger.ActionWriter,
			Stderr: ip.logger.ActionWriter,
			Env:    env,
		})
		if err != nil {
			return fmt.Errorf("failed to execute bundle config output:\nerror: %s", err)
		}
	}

	buffer := bytes.NewBuffer(nil)
	errorBuffer := bytes.NewBuffer(nil)
	args := []string{"config", "get", "cache_path"}

	ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))

	err = ip.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: buffer,
		Stderr: errorBuffer,
		Env:    env,
	})

	cachePath := filepath.Join("vendor", "cache")

	notConfigured := strings.Contains(buffer.String(), "have not configured a value") && strings.Contains(buffer.String(), "cache_path")
	if err != nil && !notConfigured {
		return fmt.Errorf("failed to execute bundle config output:\n%s\nerror: %s", errorBuffer.String(), err)
	}

	if !notConfigured && buffer.String() != "" {
		output := strings.TrimSpace(buffer.String())
		if strings.Contains(output, "Set for") {
			// bundler shows verbose output when configured in multiple places
			// extract the top value from the first "Set for" line: Set for ...: "value"
			for _, line := range strings.Split(output, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Set for") {
					if idx := strings.LastIndex(line, ": "); idx >= 0 {
						cachePath = strings.Trim(line[idx+2:], `"`)
						break
					}
				}
			}
		} else {
			// Simple value output (configured in exactly one place)
			cachePath = output
		}
	}

	ip.logger.Action(buffer.String())
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
		Stdout: ip.logger.ActionWriter,
		Stderr: ip.logger.ActionWriter,
		Env:    env,
	})
	if err != nil {
		return fmt.Errorf("failed to execute bundle install output:\n%s\nerror: %s", buffer.String(), err)
	}

	// Regenerate the lockfile to ensure it includes the current Ruby version.
	// This prevents bundler 4.x from trying to update it at runtime when
	// the working directory is read-only.
	ip.logger.Subprocess("Running 'bundle lock'")
	err = ip.executable.Execute(pexec.Execution{
		Args:   []string{"lock"},
		Stdout: ip.logger.ActionWriter,
		Stderr: ip.logger.ActionWriter,
		Env:    env,
	})
	if err != nil {
		return fmt.Errorf("failed to regenerate lockfile: %w", err)
	}

	if !keepBuildFiles {
		err = filepath.Walk(layerPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			switch filepath.Base(path) {
			case "Makefile", "mkmf.log", "gem_make.out":
				err = os.Remove(path)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to cleanup gem extension build files: %w", err)
		}
	}

	return nil
}
