package bundleinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
type InstallProcess interface {
	Execute(workingDir, layerPath string) error
}

//go:generate faux --interface Calculator --output fakes/calculator.go
type Calculator interface {
	Sum(path string) (string, error)
}

func Build(
	installProcess InstallProcess,
	calculator Calculator,
	logger LogEmitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		gemsLayer, err := context.Layers.Get(LayerNameGems, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		sum := ""

		_, err = os.Stat(filepath.Join(context.WorkingDir, "Gemfile.lock"))
		if err != nil && !os.IsNotExist(err) {
			return packit.BuildResult{}, fmt.Errorf("failed to stat Gemfile.lock: %w", err)
		}

		if err == nil {
			sum, err = calculator.Sum(filepath.Join(context.WorkingDir, "Gemfile.lock"))
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		cachedSHA, ok := gemsLayer.Metadata["cache_sha"].(string)
		if ok && cachedSHA != "" && cachedSHA == sum {
			logger.Process("Reusing cached layer %s", gemsLayer.Path)
			logger.Break()

			return packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "gems"},
					},
				},
				Layers: []packit.Layer{gemsLayer},
			}, nil
		}

		if err = gemsLayer.Reset(); err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Executing build process")
		logger.Subprocess("Running 'bundle install'")

		duration, err := clock.Measure(func() error {
			return installProcess.Execute(context.WorkingDir, gemsLayer.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		gemsLayer.Metadata = map[string]interface{}{
			"built_at":  clock.Now().Format(time.RFC3339Nano),
			"cache_sha": sum,
		}

		gemsLayer.SharedEnv.Default("BUNDLE_PATH", gemsLayer.Path)
		logger.Environment(gemsLayer.SharedEnv)

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "gems"},
				},
			},
			Layers: []packit.Layer{gemsLayer},
		}, nil
	}
}
