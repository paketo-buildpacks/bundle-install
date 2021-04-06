package bundleinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/bundle-install/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
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
		timeStamp  time.Time

		clock chronos.Clock

		installProcess *fakes.InstallProcess
		entryResolver  *fakes.EntryResolver

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Mkdir(filepath.Join(workingDir, ".bundle"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workingDir, ".bundle", "config"), nil, 0600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workingDir, ".bundle", "config.bak"), nil, 0600)).To(Succeed())

		installProcess = &fakes.InstallProcess{}
		installProcess.ShouldRunCall.Returns.Should = true
		installProcess.ShouldRunCall.Returns.Checksum = "some-checksum"
		installProcess.ShouldRunCall.Returns.RubyVersion = "some-version"

		buffer = bytes.NewBuffer(nil)
		logEmitter := bundleinstall.NewLogEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time { return timeStamp })

		entryResolver = &fakes.EntryResolver{}

		build = bundleinstall.Build(installProcess, logEmitter, clock, entryResolver)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when required during build", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("returns a result that installs build gems", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "gems",
							Metadata: map[string]interface{}{
								"build": true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:      "build-gems",
						Path:      filepath.Join(layersDir, "build-gems"),
						LaunchEnv: packit.Environment{},
						BuildEnv: packit.Environment{
							"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "build-gems", "config"),
						},
						SharedEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"built_at":     timeStamp.Format(time.RFC3339Nano),
							"cache_sha":    "some-checksum",
							"ruby_version": "some-version",
						},
					},
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

			Expect(buffer).To(ContainLines(
				"Some Buildpack some-version",
				"  Executing build environment install process",
				"      Completed in 0s",
				"",
				"  Configuring build environment",
				fmt.Sprintf("    BUNDLE_USER_CONFIG -> %q", filepath.Join(layersDir, "build-gems", "config")),
			))
		})
	})

	context("when required during launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})

		it("returns a result that installs launch gems", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "gems",
							Metadata: map[string]interface{}{
								"launch": true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name: "launch-gems",
						Path: filepath.Join(layersDir, "launch-gems"),
						LaunchEnv: packit.Environment{
							"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "launch-gems", "config"),
						},
						BuildEnv:         packit.Environment{},
						SharedEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Launch:           true,
						Metadata: map[string]interface{}{
							"built_at":     timeStamp.Format(time.RFC3339Nano),
							"cache_sha":    "some-checksum",
							"ruby_version": "some-version",
						},
					},
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

			Expect(buffer).To(ContainLines(
				"Some Buildpack some-version",
				"  Executing launch environment install process",
				"      Completed in 0s",
				"",
				"  Configuring launch environment",
				fmt.Sprintf("    BUNDLE_USER_CONFIG -> %q", filepath.Join(layersDir, "launch-gems", "config")),
			))
		})
	})

	context("when not required during either build or launch", func() {
		it("returns a result that has no layers", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{{Name: "gems"}},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{}))
		})
	})

	context("when required during both build and launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
			entryResolver.MergeLayerTypesCall.Returns.Launch = true

			Expect(os.MkdirAll(filepath.Join(layersDir, "build-gems", "ruby"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layersDir, "build-gems", "ruby", "some-file"), []byte("some-file-contents"), 0600)).To(Succeed())
		})

		it("copies gems from the build layer into the launch layer for performance", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{{Name: "gems"}},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:      "build-gems",
						Path:      filepath.Join(layersDir, "build-gems"),
						LaunchEnv: packit.Environment{},
						BuildEnv: packit.Environment{
							"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "build-gems", "config"),
						},
						SharedEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"built_at":     timeStamp.Format(time.RFC3339Nano),
							"cache_sha":    "some-checksum",
							"ruby_version": "some-version",
						},
					},
					{
						Name: "launch-gems",
						Path: filepath.Join(layersDir, "launch-gems"),
						LaunchEnv: packit.Environment{
							"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "launch-gems", "config"),
						},
						BuildEnv:         packit.Environment{},
						SharedEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Launch:           true,
						Metadata: map[string]interface{}{
							"built_at":     timeStamp.Format(time.RFC3339Nano),
							"cache_sha":    "some-checksum",
							"ruby_version": "some-version",
						},
					},
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

			err := ioutil.WriteFile(filepath.Join(layersDir, "build-gems.toml"), []byte(fmt.Sprintf(`
build = true
cache = true

[metadata]
	cache_sha = "some-checksum"
	ruby_version = "some-version"
	built_at = "%s"
`, timeStamp.Format(time.RFC3339Nano))), 0600)
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(layersDir, "launch-gems.toml"), []byte(fmt.Sprintf(`
launch = true

[metadata]
	cache_sha = "some-checksum"
	ruby_version = "some-version"
	built_at = "%s"
`, timeStamp.Format(time.RFC3339Nano))), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns a result that reuses the existing layer", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "gems",
							Metadata: map[string]interface{}{
								"launch": true,
								"build":  true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             "build-gems",
						Path:             filepath.Join(layersDir, "build-gems"),
						LaunchEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						SharedEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"built_at":     timeStamp.Format(time.RFC3339Nano),
							"cache_sha":    "some-checksum",
							"ruby_version": "some-version",
						},
					},
					{
						Name:             "launch-gems",
						Path:             filepath.Join(layersDir, "launch-gems"),
						LaunchEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						SharedEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Launch:           true,
						Metadata: map[string]interface{}{
							"built_at":     timeStamp.Format(time.RFC3339Nano),
							"cache_sha":    "some-checksum",
							"ruby_version": "some-version",
						},
					},
				},
			}))

			Expect(filepath.Join(workingDir, ".bundle", "config")).NotTo(BeAnExistingFile())

			Expect(installProcess.ShouldRunCall.Receives.Metadata).To(Equal(map[string]interface{}{
				"built_at":     timeStamp.Format(time.RFC3339Nano),
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
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{{Name: "gems"}},
						},
						Layers: packit.Layers{Path: layersDir},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when a launch layer is required", func() {
				it.Before(func() {
					entryResolver.MergeLayerTypesCall.Returns.Launch = true
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{{Name: "gems"}},
						},
						Layers: packit.Layers{Path: layersDir},
					})
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
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{{Name: "gems"}},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("failed to check if should run"))
			})
		})

		context("when the install process fails to execute", func() {
			it.Before(func() {
				entryResolver.MergeLayerTypesCall.Returns.Build = true
				installProcess.ExecuteCall.Returns.Error = errors.New("failed to execute")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{{Name: "gems"}},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("failed to execute"))
			})
		})
	})
}
