package bundle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

type BuildPlanMetadata struct {
	Build  bool `toml:"build"`
	Launch bool `toml:"launch"`
}

func Detect(gemfileParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		_, err := os.Stat(filepath.Join(context.WorkingDir, "Gemfile"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail
			}

			panic(err)
		}

		mriVersion, err := gemfileParser.ParseVersion(filepath.Join(context.WorkingDir, "Gemfile"))
		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("failed to parse Gemfile: %w", err)
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: GemsDependency},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: GemsDependency,
						Metadata: BuildPlanMetadata{
							Build:  false,
							Launch: true,
						},
					},
					{
						Name: BundlerDependency,
						Metadata: BuildPlanMetadata{
							Build:  true,
							Launch: true,
						},
					},
					{
						Name:    MRIDependency,
						Version: mriVersion,
						Metadata: BuildPlanMetadata{
							Build:  true,
							Launch: true,
						},
					},
				},
			},
		}, nil
	}
}
