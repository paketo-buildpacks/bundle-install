package bundler

import (
	"path/filepath"

	"github.com/cloudfoundry/packit"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

type BuildPlanMetadata struct {
	VersionSource string `toml:"version-source"`
	Launch        bool   `toml:"launch"`
	Build         bool   `toml:"build"`
}

func Detect(buildpackYMLParser, gemfileLockParser, gemfileParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		var requirements []packit.BuildPlanRequirement

		version, err := buildpackYMLParser.ParseVersion(filepath.Join(context.WorkingDir, BuildpackYMLSource))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name:    Bundler,
				Version: version,
				Metadata: BuildPlanMetadata{
					VersionSource: BuildpackYMLSource,
					Launch:        true,
					Build:         true,
				},
			})
		}

		version, err = gemfileLockParser.ParseVersion(filepath.Join(context.WorkingDir, GemfileLockSource))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name:    Bundler,
				Version: version,
				Metadata: BuildPlanMetadata{
					VersionSource: GemfileLockSource,
					Launch:        true,
					Build:         true,
				},
			})
		}

		version, err = gemfileParser.ParseVersion(filepath.Join(context.WorkingDir, GemfileSource))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name:    Ruby,
				Version: version,
				Metadata: BuildPlanMetadata{
					VersionSource: GemfileSource,
					Launch:        true,
					Build:         true,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: Bundler},
				},
				Requires: requirements,
			},
		}, nil
	}
}
