package bundleinstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) error
}

type BundleInstallProcess struct {
	executable Executable
	logger     LogEmitter
}

func NewBundleInstallProcess(executable Executable, logger LogEmitter) BundleInstallProcess {
	return BundleInstallProcess{
		executable: executable,
		logger:     logger,
	}
}

func (ip BundleInstallProcess) Execute(workingDir, gemLayersDir string) error {
	buffer := bytes.NewBuffer(nil)
	args := []string{"config", "path", gemLayersDir}

	ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))
	err := ip.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		return fmt.Errorf("failed to execute bundle config output:\n%s\nerror: %s", buffer.String(), err)
	}

	buffer = bytes.NewBuffer(nil)
	args = []string{"config", "clean", "true"}

	ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))
	err = ip.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		return fmt.Errorf("failed to execute bundle config output:\n%s\nerror: %s", buffer.String(), err)
	}

	buffer = bytes.NewBuffer(nil)
	args = []string{"config", "cache_path", "--parseable"}

	ip.logger.Subprocess("Running 'bundle %s'", strings.Join(args, " "))
	err = ip.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: buffer,
		Stderr: buffer,
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
	})

	if err != nil {
		return fmt.Errorf("failed to execute bundle install output:\n%s\nerror: %s", buffer.String(), err)
	}

	return nil
}
