package bundle

import (
	"time"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
type InstallProcess interface {
	Execute(workingDir, layerPath string) error
}

func Build(
	installProcess InstallProcess,
	logger LogEmitter,
	clock Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		gemsLayer, err := context.Layers.Get(LayerNameGems, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if err = gemsLayer.Reset(); err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Executing build process")
		logger.Subprocess("Running 'bundle install'")
		then := clock.Now()
		err = installProcess.Execute(context.WorkingDir, gemsLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Action("Completed in %s", time.Since(then).Round(time.Millisecond))
		logger.Break()

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
