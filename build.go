package bundleinstall

import (
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface InstallProcess --output fakes/install_process.go
//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go

// InstallProcess defines the interface for executing the "bundle install"
// build process.
type InstallProcess interface {
	ShouldRun(metadata map[string]interface{}, workingDir string) (should bool, checksum string, rubyVersion string, err error)
	Execute(workingDir, layerPath string, config map[string]string, keepBuildFiles bool) error
}

// EntryResolver defines the interface for determining what phases of the
// lifecycle will require gems.
type EntryResolver interface {
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

type SBOMGenerator interface {
	Generate(dir string) (sbom.SBOM, error)
}

// Build will return a packit.BuildFunc that will be invoked during the build
// phase of the buildpack lifecycle.
//
// Build will execute the installation process to install gems that can be used
// in the build or launch phases of the buildpack lifecycle. Specifically,
// Build will provide different sets of gems as discrete layers depending upon
// their requirement in either the build or launch phase.
//
// If gems are required during the build phase, Build will ensure that all
// gems, including those in the "development" and "test" groups are installed
// into a layer that is made available during the remainder of the build phase.
//
// If gems are required during the launch phase, Build will ensure that only
// those gems that are not in the "development" or "test" groups are installed
// into a layer that is made available during the launch phase.
//
// If gems are required during both the build and launch phases, Build will
// provide both of the above layers with their sets of gems. These layers
// operate mutually exclusively as only one is available in each of the build
// or launch phase.
//
// To improve performance when installing gems for use in both the build and
// launch phases, Build will copy the contents of the build layer into the
// launch layer before executing the launch layer installation process. This
// will result in the launch layer installation process performing an effective
// "no-op" as all of the gems that it requires should already be copied into
// the layer. The launch layer installation process will however perform a
// "bundle clean" to remove any extra gems, including those from the
// "development" and "test" groups that may have been copied from the build
// layer.
//
// Finally, upon completing the installation process, Build will remove any
// local bundler configuration files such that the Bundler CLI will only follow
// configuration from the global location, which will be configured to point to
// a file that is maintained in each of the build and launch layers
// respectively.
func Build(
	entries EntryResolver,
	installProcess InstallProcess,
	sbomGenerator SBOMGenerator,
	logger scribe.Emitter,
	clock chronos.Clock,
	environment Environment,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		launch, build := entries.MergeLayerTypes("gems", context.Plan.Entries)

		var layers []packit.Layer

		if build {
			logger.Debug.Process("Getting the layer associated with %s", LayerNameBuildGems)
			layer, err := context.Layers.Get(LayerNameBuildGems)
			if err != nil {
				return packit.BuildResult{}, err
			}
			logger.Debug.Subprocess(layer.Path)
			logger.Debug.Break()

			layer.Build = true
			layer.Cache = true

			logger.Debug.Process("Checking if the build environment install process should run")
			logger.Debug.Break()
			should, checksum, rubyVersion, err := installProcess.ShouldRun(layer.Metadata, context.WorkingDir)
			if err != nil {
				return packit.BuildResult{}, err
			}

			stack, ok := layer.Metadata["stack"]
			if ok && stack.(string) != context.Stack {
				logger.Process("Stack upgraded from %s to %s, clearing cached gems", stack.(string), context.Stack)
				layer, err = layer.Reset()
				if err != nil {
					return packit.BuildResult{}, err
				}

				should = true
				layer.Build = true
				layer.Cache = true
			}

			if should {
				logger.Process("Executing build environment install process")

				duration, err := clock.Measure(func() error {
					return installProcess.Execute(context.WorkingDir, layer.Path, map[string]string{
						"path":  layer.Path,
						"clean": "true",
					}, environment.KeepGemExtensionBuildFiles)
				})
				if err != nil {
					return packit.BuildResult{}, err
				}

				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				layer.BuildEnv.Default("BUNDLE_USER_CONFIG", filepath.Join(layer.Path, "config"))
				layer.Metadata = map[string]interface{}{
					"stack":        context.Stack,
					"cache_sha":    checksum,
					"ruby_version": rubyVersion,
				}

				logger.GeneratingSBOM(layer.Path)

				var sbomContent sbom.SBOM
				duration, err = clock.Measure(func() error {
					sbomContent, err = sbomGenerator.Generate(context.WorkingDir)
					return err
				})
				if err != nil {
					return packit.BuildResult{}, err
				}
				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)

				layer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
				if err != nil {
					return packit.BuildResult{}, err
				}
			} else {
				logger.Process("Reusing cached layer %s", layer.Path)
				logger.Break()
			}

			layers = append(layers, layer)
		}

		if launch {
			logger.Debug.Process("Getting the layer associated with %s", LayerNameLaunchGems)
			layer, err := context.Layers.Get(LayerNameLaunchGems)
			if err != nil {
				return packit.BuildResult{}, err
			}
			logger.Debug.Subprocess(layer.Path)
			logger.Debug.Break()

			layer.Launch = true

			logger.Debug.Process("Checking if the launch environment install process should run")
			logger.Debug.Break()
			should, checksum, rubyVersion, err := installProcess.ShouldRun(layer.Metadata, context.WorkingDir)
			if err != nil {
				return packit.BuildResult{}, err
			}

			stack, ok := layer.Metadata["stack"]
			if ok && stack.(string) != context.Stack {
				logger.Process("Stack upgraded from %s to %s, clearing cached gems", stack.(string), context.Stack)
				layer, err = layer.Reset()
				if err != nil {
					return packit.BuildResult{}, err
				}
				should = true
				layer.Launch = true
			}

			if should {
				logger.Process("Executing launch environment install process")

				duration, err := clock.Measure(func() error {
					if build {
						buildLayer, err := context.Layers.Get(LayerNameBuildGems)
						if err != nil {
							return err
						}

						err = fs.Copy(filepath.Join(buildLayer.Path), filepath.Join(layer.Path))
						if err != nil {
							return err
						}
					}

					return installProcess.Execute(context.WorkingDir, layer.Path, map[string]string{
						"path":    layer.Path,
						"without": "development:test",
						"clean":   "true",
					}, environment.KeepGemExtensionBuildFiles)
				})
				if err != nil {
					return packit.BuildResult{}, err
				}

				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				layer.LaunchEnv.Default("BUNDLE_USER_CONFIG", filepath.Join(layer.Path, "config"))
				layer.Metadata = map[string]interface{}{
					"stack":        context.Stack,
					"cache_sha":    checksum,
					"ruby_version": rubyVersion,
				}

				logger.GeneratingSBOM(layer.Path)

				var sbomContent sbom.SBOM
				duration, err = clock.Measure(func() error {
					sbomContent, err = sbomGenerator.Generate(context.WorkingDir)
					return err
				})
				if err != nil {
					return packit.BuildResult{}, err
				}
				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)

				layer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
				if err != nil {
					return packit.BuildResult{}, err
				}
			} else {
				logger.Process("Reusing cached layer %s", layer.Path)
				logger.Break()
			}
			layers = append(layers, layer)
		}

		for _, layer := range layers {
			logger.EnvironmentVariables(layer)
		}

		logger.Debug.Process("Cleaning up %s/.bundle/config", context.WorkingDir)
		err := os.RemoveAll(filepath.Join(context.WorkingDir, ".bundle", "config"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Debug.Process("Cleaning up %s/.bundle/config.bak", context.WorkingDir)
		logger.Debug.Break()
		err = os.RemoveAll(filepath.Join(context.WorkingDir, ".bundle", "config.bak"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		return packit.BuildResult{Layers: layers}, nil
	}
}
