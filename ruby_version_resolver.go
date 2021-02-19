package bundleinstall

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/paketo-buildpacks/packit/pexec"
)

type RubyVersionResolver struct {
	executable Executable
}

func NewRubyVersionResolver(executable Executable) RubyVersionResolver {
	return RubyVersionResolver{
		executable: executable,
	}
}

func (r RubyVersionResolver) Lookup() (string, error) {
	buffer := bytes.NewBuffer(nil)
	err := r.executable.Execute(pexec.Execution{
		Args:   []string{"--version"},
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		return "", fmt.Errorf("failed to obtain ruby version: %w: %s", err, buffer.String())
	}

	versions := regexp.MustCompile(`ruby (\d+\.\d+\.\d+)`).FindStringSubmatch(buffer.String())
	if versions == nil {
		return "", fmt.Errorf("no string matching 'ruby (\\d+\\.\\d+\\.\\d+)' found")
	}

	// return just the numeric version part of the `ruby --version` output
	return versions[1], nil
}
