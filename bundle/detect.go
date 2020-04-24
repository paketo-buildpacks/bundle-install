package bundle

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/bundler-cnb/bundler"
	"github.com/cloudfoundry/packit"
)

const (
	GemDependency = "gems"
)

type BuildPlanMetadata struct {
	Build  bool `toml:"build"`
	Launch bool `toml:"launch"`
}

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		_, err := os.Stat(filepath.Join(context.WorkingDir, "Gemfile"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return packit.DetectResult{}, packit.Fail
			}

			return packit.DetectResult{}, err
		}

		// TODO: Read version info and process
		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: GemDependency},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: GemDependency,
						Metadata: BuildPlanMetadata{
							Build:  false,
							Launch: true,
						},
					},
					{
						Name: bundler.Bundler,
						Metadata: BuildPlanMetadata{
							Build:  true,
							Launch: false,
						},
					},
				},
			},
		}, nil
	}
}
