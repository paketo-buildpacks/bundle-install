package bundle

import (
	"bytes"
	"fmt"

	"github.com/cloudfoundry/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) error
}

type BundleInstallProcess struct {
	executable Executable
}

func NewBundleInstallProcess(executable Executable) BundleInstallProcess {
	return BundleInstallProcess{
		executable: executable,
	}
}

func (ip BundleInstallProcess) Execute(workingDir, gemLayersDir string) error {
	buffer := bytes.NewBuffer(nil)
	err := ip.executable.Execute(pexec.Execution{
		Args:   []string{"config", "set", "path", gemLayersDir},
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		return fmt.Errorf("failed to execute bundle config output:\n%s\nerror: %s", buffer.String(), err)
	}

	buffer = bytes.NewBuffer(nil)
	err = ip.executable.Execute(pexec.Execution{
		Args:   []string{"install"},
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		return fmt.Errorf("failed to execute bundle install output:\n%s\nerror: %s", buffer.String(), err)
	}

	return nil
}
