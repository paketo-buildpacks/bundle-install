package bundleinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/bundle-install/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		buffer     *bytes.Buffer

		clock chronos.Clock

		installProcess *fakes.InstallProcess
		entryResolver  *fakes.EntryResolver
		sbomGenerator  *fakes.SBOMGenerator

		build        packit.BuildFunc
		buildContext packit.BuildContext
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Mkdir(filepath.Join(workingDir, ".bundle"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workingDir, ".bundle", "config"), nil, 0600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workingDir, ".bundle", "config.bak"), nil, 0600)).To(Succeed())

		installProcess = &fakes.InstallProcess{}
		installProcess.ShouldRunCall.Returns.Should = true
		installProcess.ShouldRunCall.Returns.Checksum = "some-checksum"
		installProcess.ShouldRunCall.Returns.RubyVersion = "some-version"

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		clock = chronos.DefaultClock

		entryResolver = &fakes.EntryResolver{}

		build = bundleinstall.Build(
			entryResolver,
			installProcess,
			sbomGenerator,
			logEmitter,
			clock,
		)

		buildContext = packit.BuildContext{
			WorkingDir: workingDir,
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			Plan:   packit.BuildpackPlan{},
			Layers: packit.Layers{Path: layersDir},
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when required during build", func() {
		it.Before(func() {
			buildContext.Plan.Entries = []packit.BuildpackPlanEntry{
				{
					Name: "gems",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
			}
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("returns a result that installs build gems", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(1))

			layer := layers[0]
			Expect(layer.Name).To(Equal("build-gems"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "build-gems")))

			Expect(layer.Build).To(BeTrue())
			Expect(layer.Launch).To(BeFalse())
			Expect(layer.Cache).To(BeTrue())

			Expect(layer.BuildEnv).To(Equal(packit.Environment{
				"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "build-gems", "config"),
			}))
			Expect(layer.LaunchEnv).To(BeEmpty())
			Expect(layer.ProcessLaunchEnv).To(BeEmpty())
			Expect(layer.SharedEnv).To(BeEmpty())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"cache_sha":    "some-checksum",
				"ruby_version": "some-version",
			}))

			Expect(layer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: sbom.Format(sbom.SPDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
			}))

			Expect(filepath.Join(workingDir, ".bundle", "config")).NotTo(BeAnExistingFile())
			Expect(filepath.Join(workingDir, ".bundle", "config.bak")).NotTo(BeAnExistingFile())

			Expect(entryResolver.MergeLayerTypesCall.Receives.String).To(Equal("gems"))
			Expect(entryResolver.MergeLayerTypesCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
				{Name: "gems", Metadata: map[string]interface{}{"build": true}},
			}))

			Expect(installProcess.ShouldRunCall.Receives.Metadata).To(BeNil())
			Expect(installProcess.ShouldRunCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))
			Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(installProcess.ExecuteCall.Receives.Config).To(Equal(map[string]string{
				"path":  filepath.Join(layersDir, "build-gems"),
				"clean": "true",
			}))

			Expect(sbomGenerator.GenerateCall.Receives.Dir).To(Equal(workingDir))

			Expect(buffer).To(ContainLines(
				"Some Buildpack some-version",
				"  Executing build environment install process",
				"      Completed in 0s",
			))
			Expect(buffer).To(ContainLines(
				"  Configuring build environment",
				fmt.Sprintf("    BUNDLE_USER_CONFIG -> %q", filepath.Join(layersDir, "build-gems", "config")),
			))
		})
	})

	context("when required during launch", func() {
		it.Before(func() {
			buildContext.Plan.Entries = []packit.BuildpackPlanEntry{
				{
					Name: "gems",
					Metadata: map[string]interface{}{
						"launch": true,
					},
				},
			}
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})

		it("returns a result that installs launch gems", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(1))

			layer := layers[0]
			Expect(layer.Name).To(Equal("launch-gems"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "launch-gems")))

			Expect(layer.Build).To(BeFalse())
			Expect(layer.Launch).To(BeTrue())
			Expect(layer.Cache).To(BeFalse())

			Expect(layer.BuildEnv).To(BeEmpty())
			Expect(layer.LaunchEnv).To(Equal(packit.Environment{
				"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "launch-gems", "config"),
			}))
			Expect(layer.ProcessLaunchEnv).To(BeEmpty())
			Expect(layer.SharedEnv).To(BeEmpty())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"cache_sha":    "some-checksum",
				"ruby_version": "some-version",
			}))

			Expect(layer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: sbom.Format(sbom.SPDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
			}))

			Expect(filepath.Join(workingDir, ".bundle", "config")).NotTo(BeAnExistingFile())

			Expect(entryResolver.MergeLayerTypesCall.Receives.String).To(Equal("gems"))
			Expect(entryResolver.MergeLayerTypesCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
				{Name: "gems", Metadata: map[string]interface{}{"launch": true}},
			}))

			Expect(installProcess.ShouldRunCall.Receives.Metadata).To(BeNil())
			Expect(installProcess.ShouldRunCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))
			Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(installProcess.ExecuteCall.Receives.Config).To(Equal(map[string]string{
				"path":    filepath.Join(layersDir, "launch-gems"),
				"without": "development:test",
				"clean":   "true",
			}))

			Expect(sbomGenerator.GenerateCall.Receives.Dir).To(Equal(workingDir))

			Expect(buffer).To(ContainLines(
				"Some Buildpack some-version",
				"  Executing launch environment install process",
				"      Completed in 0s",
			))
			Expect(buffer).To(ContainLines(
				"  Configuring launch environment",
				fmt.Sprintf("    BUNDLE_USER_CONFIG -> %q", filepath.Join(layersDir, "launch-gems", "config")),
			))
		})
	})

	context("when not required during either build or launch", func() {
		it("returns a result that has no layers", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{}))
		})
	})

	context("when required during both build and launch", func() {
		it.Before(func() {
			buildContext.Plan.Entries = []packit.BuildpackPlanEntry{
				{
					Name: "gems",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
				{
					Name: "gems",
					Metadata: map[string]interface{}{
						"launch": true,
					},
				},
			}

			entryResolver.MergeLayerTypesCall.Returns.Build = true
			entryResolver.MergeLayerTypesCall.Returns.Launch = true

			Expect(os.MkdirAll(filepath.Join(layersDir, "build-gems", "ruby"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layersDir, "build-gems", "ruby", "some-file"), []byte("some-file-contents"), 0600)).To(Succeed())
		})

		it("copies gems from the build layer into the launch layer for performance", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(2))

			buildLayer := layers[0]
			Expect(buildLayer.Name).To(Equal("build-gems"))
			Expect(buildLayer.Path).To(Equal(filepath.Join(layersDir, "build-gems")))

			Expect(buildLayer.Build).To(BeTrue())
			Expect(buildLayer.Launch).To(BeFalse())
			Expect(buildLayer.Cache).To(BeTrue())

			Expect(buildLayer.BuildEnv).To(Equal(packit.Environment{
				"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "build-gems", "config"),
			}))
			Expect(buildLayer.LaunchEnv).To(BeEmpty())
			Expect(buildLayer.ProcessLaunchEnv).To(BeEmpty())
			Expect(buildLayer.SharedEnv).To(BeEmpty())

			Expect(buildLayer.Metadata).To(Equal(map[string]interface{}{
				"cache_sha":    "some-checksum",
				"ruby_version": "some-version",
			}))

			Expect(buildLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: sbom.Format(sbom.SPDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
			}))

			launchLayer := layers[1]
			Expect(launchLayer.Name).To(Equal("launch-gems"))
			Expect(launchLayer.Path).To(Equal(filepath.Join(layersDir, "launch-gems")))

			Expect(launchLayer.Build).To(BeFalse())
			Expect(launchLayer.Launch).To(BeTrue())
			Expect(launchLayer.Cache).To(BeFalse())

			Expect(launchLayer.BuildEnv).To(BeEmpty())
			Expect(launchLayer.LaunchEnv).To(Equal(packit.Environment{
				"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "launch-gems", "config"),
			}))
			Expect(launchLayer.ProcessLaunchEnv).To(BeEmpty())
			Expect(launchLayer.SharedEnv).To(BeEmpty())

			Expect(launchLayer.Metadata).To(Equal(map[string]interface{}{
				"cache_sha":    "some-checksum",
				"ruby_version": "some-version",
			}))

			Expect(launchLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: sbom.Format(sbom.SPDXFormat).Extension(),
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
			}))

			content, err := os.ReadFile(filepath.Join(layersDir, "launch-gems", "ruby", "some-file"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("some-file-contents"))
		})
	})

	context("when reusing a layer", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
			entryResolver.MergeLayerTypesCall.Returns.Launch = true

			installProcess.ShouldRunCall.Returns.Should = false

			err := os.WriteFile(filepath.Join(layersDir, "build-gems.toml"), []byte(`
build = true
cache = true

[metadata]
	cache_sha = "some-checksum"
	ruby_version = "some-version"
`), 0600)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(filepath.Join(layersDir, "launch-gems.toml"), []byte(`
launch = true

[metadata]
	cache_sha = "some-checksum"
	ruby_version = "some-version"
`), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns a result that reuses the existing layer", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(2))

			buildLayer := layers[0]
			Expect(buildLayer.Name).To(Equal("build-gems"))
			Expect(buildLayer.Path).To(Equal(filepath.Join(layersDir, "build-gems")))

			Expect(buildLayer.Build).To(BeTrue())
			Expect(buildLayer.Launch).To(BeFalse())
			Expect(buildLayer.Cache).To(BeTrue())

			launchLayer := layers[1]
			Expect(launchLayer.Name).To(Equal("launch-gems"))
			Expect(launchLayer.Path).To(Equal(filepath.Join(layersDir, "launch-gems")))

			Expect(launchLayer.Build).To(BeFalse())
			Expect(launchLayer.Launch).To(BeTrue())
			Expect(launchLayer.Cache).To(BeFalse())

			Expect(filepath.Join(workingDir, ".bundle", "config")).NotTo(BeAnExistingFile())

			Expect(installProcess.ShouldRunCall.Receives.Metadata).To(Equal(map[string]interface{}{
				"cache_sha":    "some-checksum",
				"ruby_version": "some-version",
			}))
			Expect(installProcess.ShouldRunCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(installProcess.ExecuteCall.CallCount).To(Equal(0))

			Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
			Expect(buffer).To(ContainLines(
				"Some Buildpack some-version",
				fmt.Sprintf("  Reusing cached layer %s", filepath.Join(layersDir, "build-gems")),
				"",
				fmt.Sprintf("  Reusing cached layer %s", filepath.Join(layersDir, "launch-gems")),
			))
		})
	})

	context("failure cases", func() {
		context("when the layers directory cannot be written to", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			context("when a build layer is required", func() {
				it.Before(func() {
					entryResolver.MergeLayerTypesCall.Returns.Build = true
				})

				it("returns an error", func() {
					_, err := build(buildContext)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when a launch layer is required", func() {
				it.Before(func() {
					entryResolver.MergeLayerTypesCall.Returns.Launch = true
				})

				it("returns an error", func() {
					_, err := build(buildContext)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})

		context("when the install process fails to determine if it should run", func() {
			it.Before(func() {
				entryResolver.MergeLayerTypesCall.Returns.Build = true
				installProcess.ShouldRunCall.Returns.Err = errors.New("failed to check if should run")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("failed to check if should run"))
			})
		})

		context("when the install process fails to execute", func() {
			it.Before(func() {
				entryResolver.MergeLayerTypesCall.Returns.Build = true
				installProcess.ExecuteCall.Returns.Error = errors.New("failed to execute")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("failed to execute"))
			})
		})

		context("when generating the build SBOM returns an error", func() {
			it.Before(func() {
				buildContext.Plan.Entries = []packit.BuildpackPlanEntry{
					{
						Name: "gems",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				}
				entryResolver.MergeLayerTypesCall.Returns.Build = true

				sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the build SBOM returns an error", func() {
			it.Before(func() {
				buildContext.Plan.Entries = []packit.BuildpackPlanEntry{
					{
						Name: "gems",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				}
				entryResolver.MergeLayerTypesCall.Returns.Build = true

				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError(`unsupported SBOM format: 'random-format'`))
			})
		})

		context("when generating the launch SBOM returns an error", func() {
			it.Before(func() {
				buildContext.Plan.Entries = []packit.BuildpackPlanEntry{
					{
						Name: "gems",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
				}
				entryResolver.MergeLayerTypesCall.Returns.Launch = true

				sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the launch SBOM returns an error", func() {
			it.Before(func() {
				buildContext.Plan.Entries = []packit.BuildpackPlanEntry{
					{
						Name: "gems",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
				}
				entryResolver.MergeLayerTypesCall.Returns.Launch = true

				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError(`unsupported SBOM format: 'random-format'`))
			})
		})
	})
}
