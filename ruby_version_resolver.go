package bundleinstall

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/packit/v2/pexec"
)

// RubyVersionResolver identifies and compares versions of Ruby used in the
// build environment.
type RubyVersionResolver struct {
	executable Executable
}

// NewRubyVersionResolver initializes an instance of RubyVersionResolver.
func NewRubyVersionResolver(executable Executable) RubyVersionResolver {
	return RubyVersionResolver{
		executable: executable,
	}
}

// Lookup returns the version of Ruby installed in the build environment.
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

// CompareMajorMinor returns true if the current major or minor version of Ruby
// has changed from the cached version.
func (r RubyVersionResolver) CompareMajorMinor(cachedVersion, newVersion string) (bool, error) {
	cachedSemverVersion, err := semver.NewVersion(cachedVersion)
	if err != nil {
		return false, err
	}

	majorMinorConstraint, err := semver.NewConstraint(fmt.Sprintf("%d.%d.*", cachedSemverVersion.Major(), cachedSemverVersion.Minor()))
	if err != nil {
		return false, err
	}

	newSemverVersion, err := semver.NewVersion(newVersion)
	if err != nil {
		return false, err
	}

	return majorMinorConstraint.Check(newSemverVersion), nil
}
