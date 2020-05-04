package bundle

import (
	"fmt"

	"github.com/cloudfoundry/packit"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
type InstallProcess interface {
	Execute(workingDir, layerPath string) error
}

func Build(installProcess InstallProcess) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		//Output statement until LogEmitter is implemented
		fmt.Println("\nBundle Install CNB")

		gemsLayer, err := context.Layers.Get(LayerNameGems, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if err = gemsLayer.Reset(); err != nil {
			return packit.BuildResult{}, err
		}

		gemsLayer.SharedEnv.Default("BUNDLE_PATH", gemsLayer.Path)

		err = installProcess.Execute(context.WorkingDir, gemsLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

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
