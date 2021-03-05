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
	Sum(paths ...string) (string, error)
}

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(string, []packit.BuildpackPlanEntry, []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface VersionResolver --output fakes/version_resolver.go
type VersionResolver interface {
	Lookup() (version string, err error)
	CompareMajorMinor(string, string) (bool, error)
}

func Build(
	installProcess InstallProcess,
	calculator Calculator,
	logger LogEmitter,
	clock chronos.Clock,
	entries EntryResolver,
	versionResolver VersionResolver,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		entry, _ := entries.Resolve("gems", context.Plan.Entries, []interface{}{})

		gemsLayer, err := context.Layers.Get(LayerNameGems)
		if err != nil {
			return packit.BuildResult{}, err
		}

		rubyVersion, err := versionResolver.Lookup()
		if err != nil {
			return packit.BuildResult{}, err
		}

		cachedRubyVersion, cachedVersionExists := gemsLayer.Metadata["ruby_version"].(string)
		// If there is no cached ruby version, then the match is true
		rubyVersionMatch := true

		if cachedVersionExists {
			// rubyVersionMatch will be true if at least the major and minor versions match
			rubyVersionMatch, err = versionResolver.CompareMajorMinor(cachedRubyVersion, rubyVersion)
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		var sum string
		_, err = os.Stat(filepath.Join(context.WorkingDir, "Gemfile.lock"))
		if err != nil {
			if !os.IsNotExist(err) {
				return packit.BuildResult{}, fmt.Errorf("failed to stat Gemfile.lock: %w", err)
			}
		} else {
			sum, err = calculator.Sum(filepath.Join(context.WorkingDir, "Gemfile"), filepath.Join(context.WorkingDir, "Gemfile.lock"))
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		cachedSHA, ok := gemsLayer.Metadata["cache_sha"].(string)
		cacheMatch := ok && cachedSHA == sum

		if cacheMatch && rubyVersionMatch {
			logger.Process("Reusing cached layer %s", gemsLayer.Path)
			logger.Break()

			return packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{entry},
				},
				Layers: []packit.Layer{gemsLayer},
			}, nil
		}

		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			return installProcess.Execute(context.WorkingDir, gemsLayer.Path)
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

		gemsLayer.SharedEnv.Default("BUNDLE_PATH", gemsLayer.Path)
		gemsLayer.SharedEnv.Default("BUNDLE_WITHOUT", "development:test")
		logger.Environment(gemsLayer.SharedEnv)

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{entry},
			},
			Layers: []packit.Layer{gemsLayer},
		}, nil
	}
}
