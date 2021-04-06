package bundleinstall

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go

// VersionParser defines the interface for parsing the version of Ruby used by
// the application.
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

// BuildPlanMetadata declares the set of metadata included in buildplan
// requirements.
type BuildPlanMetadata struct {
	Version       string `toml:"version"`
	VersionSource string `toml:"version-source,omitempty"`
	Build         bool   `toml:"build"`
	Launch        bool   `toml:"launch"`
}

// Detect will return a packit.DetectFunc that will be invoked during the
// detect phase of the buildpack lifecycle.
//
// Detect will return a positive result if the application source code contains
// a Gemfile.
//
// The buildplan entries for a positive detection include providing the "gems"
// dependency, and requiring the "bundler" and "mri" dependencies. If the
// Gemfile contains a specified Ruby version, the "mri" build plan entry will
// include a specific Ruby version contraint.
func Detect(gemfileParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		mriVersion, err := gemfileParser.ParseVersion(filepath.Join(context.WorkingDir, "Gemfile"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail.WithMessage("Gemfile is not present")
			}
			return packit.DetectResult{}, err
		}
		var versionSource string
		if mriVersion != "" {
			versionSource = "Gemfile"
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: GemsDependency},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: BundlerDependency,
						Metadata: BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: MRIDependency,
						Metadata: BuildPlanMetadata{
							Version:       mriVersion,
							VersionSource: versionSource,
							Build:         true,
						},
					},
				},
			},
		}, nil
	}
}
