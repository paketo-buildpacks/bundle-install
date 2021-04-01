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

		installProcess = &fakes.InstallProcess{}
		installProcess.ShouldRunCall.Returns.Should = true
		installProcess.ShouldRunCall.Returns.RubyVersion = "some-version"
		installProcess.ShouldRunCall.Returns.Checksum = "some-checksum"

		buffer = bytes.NewBuffer(nil)
		logEmitter := bundleinstall.NewLogEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time { return timeStamp })

		entryResolver = &fakes.EntryResolver{}
		entryResolver.MergeLayerTypesCall.Returns.Launch = true
		entryResolver.MergeLayerTypesCall.Returns.Build = true

		build = bundleinstall.Build(installProcess, logEmitter, clock, entryResolver)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs gems", func() {
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
					Name:      "gems",
					Path:      filepath.Join(layersDir, "gems"),
					LaunchEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					SharedEnv: packit.Environment{
						"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "gems", "config"),
					},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            true,
					Launch:           true,
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

		Expect(installProcess.ShouldRunCall.Receives.Layer).To(Equal(packit.Layer{
			Path:      filepath.Join(layersDir, "gems"),
			Name:      "gems",
			LaunchEnv: packit.Environment{},
			BuildEnv:  packit.Environment{},
			SharedEnv: packit.Environment{
				"BUNDLE_USER_CONFIG.default": filepath.Join(layersDir, "gems", "config"),
			},
			ProcessLaunchEnv: map[string]packit.Environment{},
		}))
		Expect(installProcess.ShouldRunCall.Receives.WorkingDir).To(Equal(workingDir))

		Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))
		Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(installProcess.ExecuteCall.Receives.Config).To(Equal(map[string]string{
			"path":    filepath.Join(layersDir, "gems"),
			"without": "development:test",
			"clean":   "true",
		}))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Configuring environment"))
	})

	context("when reusing a layer", func() {
		it.Before(func() {
			installProcess.ShouldRunCall.Returns.Should = false

			err := ioutil.WriteFile(filepath.Join(layersDir, fmt.Sprintf("%s.toml", bundleinstall.LayerNameGems)), []byte(fmt.Sprintf(`
build = true
launch = true
cache = true

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
						Name:             "gems",
						Path:             filepath.Join(layersDir, "gems"),
						LaunchEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						SharedEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
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

			Expect(installProcess.ShouldRunCall.Receives.Layer).To(Equal(packit.Layer{
				Path:             filepath.Join(layersDir, "gems"),
				Name:             "gems",
				LaunchEnv:        packit.Environment{},
				BuildEnv:         packit.Environment{},
				SharedEnv:        packit.Environment{},
				ProcessLaunchEnv: map[string]packit.Environment{},
				Build:            true,
				Launch:           true,
				Cache:            true,
				Metadata: map[string]interface{}{
					"built_at":     timeStamp.Format(time.RFC3339Nano),
					"cache_sha":    "some-checksum",
					"ruby_version": "some-version",
				},
			}))
			Expect(installProcess.ShouldRunCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(installProcess.ExecuteCall.CallCount).To(Equal(0))

			Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
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

		context("when the install process fails to determine if it should run", func() {
			it.Before(func() {
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
