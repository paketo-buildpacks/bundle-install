package bundleinstall

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

type BuildPlanMetadata struct {
	Version string `toml:"version"`
	Build   bool   `toml:"build"`
	Launch  bool   `toml:"launch"`
}

func Detect(gemfileParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		mriVersion, err := gemfileParser.ParseVersion(filepath.Join(context.WorkingDir, "Gemfile"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail.WithMessage("Gemfile is not present")
			}
			return packit.DetectResult{}, err
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
							Version: mriVersion,
							Build:   true,
						},
					},
				},
			},
		}, nil
	}
}
