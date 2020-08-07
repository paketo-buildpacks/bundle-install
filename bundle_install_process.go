package bundleinstall

import (
	"bytes"
	"fmt"
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
	args = []string{"install"}

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
