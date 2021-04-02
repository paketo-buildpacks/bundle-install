package bundleinstall

import (
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
type InstallProcess interface {
	ShouldRun(layer packit.Layer, workingDir string) (should bool, checksum string, rubyVersion string, err error)
	Execute(workingDir, layerPath string, config map[string]string) error
}

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

func Build(installProcess InstallProcess, logger LogEmitter, clock chronos.Clock, entries EntryResolver) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		gemsLayer, err := context.Layers.Get(LayerNameGems)
		if err != nil {
			return packit.BuildResult{}, err
		}

		should, sum, rubyVersion, err := installProcess.ShouldRun(gemsLayer, context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if !should {
			logger.Process("Reusing cached layer %s", gemsLayer.Path)
			logger.Break()

			err := os.RemoveAll(filepath.Join(context.WorkingDir, ".bundle", "config"))
			if err != nil {
				return packit.BuildResult{}, err
			}

			return packit.BuildResult{Layers: []packit.Layer{gemsLayer}}, nil
		}

		logger.Process("Executing build process")

		duration, err := clock.Measure(func() error {
			err := installProcess.Execute(context.WorkingDir, gemsLayer.Path, map[string]string{
				"path":    gemsLayer.Path,
				"without": "development:test",
				"clean":   "true",
			})
			if err != nil {
				return err
			}

			return os.RemoveAll(filepath.Join(context.WorkingDir, ".bundle", "config"))
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		gemsLayer.Launch, gemsLayer.Build = entries.MergeLayerTypes("gems", context.Plan.Entries)
		gemsLayer.Cache = gemsLayer.Build

		gemsLayer.Metadata = map[string]interface{}{
			"built_at":     clock.Now().Format(time.RFC3339Nano),
			"cache_sha":    sum,
			"ruby_version": rubyVersion,
		}

		gemsLayer.SharedEnv.Default("BUNDLE_USER_CONFIG", filepath.Join(gemsLayer.Path, "config"))
		logger.Environment(gemsLayer.SharedEnv)

		return packit.BuildResult{Layers: []packit.Layer{gemsLayer}}, nil
	}
}
