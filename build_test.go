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
		cnbDir     string
		buffer     *bytes.Buffer
		timeStamp  time.Time

		clock chronos.Clock

		installProcess *fakes.InstallProcess
		calculator     *fakes.Calculator
		entryResolver  *fakes.EntryResolver

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		installProcess = &fakes.InstallProcess{}

		buffer = bytes.NewBuffer(nil)
		logEmitter := bundleinstall.NewLogEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		calculator = &fakes.Calculator{}
		calculator.SumCall.Returns.String = "some-calculator-sha"

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "gems",
			Metadata: map[string]interface{}{
				"launch": true,
				"build":  true,
			},
		}

		entryResolver.MergeLayerTypesCall.Returns.Launch = true
		entryResolver.MergeLayerTypesCall.Returns.Build = true

		build = bundleinstall.Build(installProcess, calculator, logEmitter, clock, entryResolver)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs gems", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
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

		Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))
		Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(installProcess.ExecuteCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "gems")))
		Expect(result).To(Equal(packit.BuildResult{
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
			Layers: []packit.Layer{
				{
					Name:      "gems",
					Path:      filepath.Join(layersDir, "gems"),
					LaunchEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					SharedEnv: packit.Environment{
						"BUNDLE_PATH.default": filepath.Join(layersDir, "gems"),
					},
					Build:  true,
					Launch: true,
					Cache:  true,
					Metadata: map[string]interface{}{
						"built_at":  timeStamp.Format(time.RFC3339Nano),
						"cache_sha": "",
					},
				},
			},
		}))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Configuring environment"))
	})

	context("when rebuilding a layer", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(layersDir, fmt.Sprintf("%s.toml", bundleinstall.LayerNameGems)), []byte(fmt.Sprintf(`[metadata]
			cache_sha = "some-calculator-sha"
			built_at = "%s"
			`, timeStamp.Format(time.RFC3339Nano))), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		context("when working dir has no Gemfile.lock", func() {
			it.Before(func() {
				entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
					Name: "gems",
					Metadata: map[string]interface{}{
						"launch": true,
					},
				}

				entryResolver.MergeLayerTypesCall.Returns.Launch = true
				entryResolver.MergeLayerTypesCall.Returns.Build = false
			})

			it("runs the install process", func() {
				result, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
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
					Layers: []packit.Layer{
						{
							Name:      "gems",
							Path:      filepath.Join(layersDir, "gems"),
							LaunchEnv: packit.Environment{},
							BuildEnv:  packit.Environment{},
							SharedEnv: packit.Environment{
								"BUNDLE_PATH.default": filepath.Join(layersDir, "gems"),
							},
							Build:  false,
							Launch: true,
							Cache:  false,
							Metadata: map[string]interface{}{
								"built_at":  timeStamp.Format(time.RFC3339Nano),
								"cache_sha": "",
							},
						},
					},
				}))
				Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))
			})
		})

		context("when working dir has Gemfile.lock", func() {
			it.Before(func() {
				err := ioutil.WriteFile(filepath.Join(workingDir, "Gemfile"), nil, 0600)
				Expect(err).NotTo(HaveOccurred())

				err = ioutil.WriteFile(filepath.Join(workingDir, "Gemfile.lock"), nil, 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				err := os.RemoveAll(filepath.Join(workingDir, "Gemfile"))
				Expect(err).NotTo(HaveOccurred())

				err = os.RemoveAll(filepath.Join(workingDir, "Gemfile.lock"))
				Expect(err).NotTo(HaveOccurred())
			})

			context("when checksum matches", func() {
				it.Before(func() {
					entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{Name: "gems"}
				})

				it("does not run the install process", func() {
					result, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						CNBPath:    cnbDir,
						Stack:      "some-stack",
						BuildpackInfo: packit.BuildpackInfo{
							Name:    "Some Buildpack",
							Version: "some-version",
						},
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "gems"},
							},
						},
						Layers: packit.Layers{Path: layersDir},
					})
					Expect(err).NotTo(HaveOccurred())

					Expect(installProcess.ExecuteCall.CallCount).To(Equal(0))
					Expect(calculator.SumCall.CallCount).To(Equal(1))
					Expect(calculator.SumCall.Receives.Paths).To(Equal([]string{filepath.Join(workingDir, "Gemfile"), filepath.Join(workingDir, "Gemfile.lock")}))

					Expect(result).To(Equal(packit.BuildResult{
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "gems"},
							},
						},
						Layers: []packit.Layer{
							{
								Name:      "gems",
								Path:      filepath.Join(layersDir, "gems"),
								LaunchEnv: packit.Environment{},
								BuildEnv:  packit.Environment{},
								SharedEnv: packit.Environment{},
								Build:     false,
								Launch:    false,
								Cache:     false,
								Metadata: map[string]interface{}{
									"built_at":  timeStamp.Format(time.RFC3339Nano),
									"cache_sha": "some-calculator-sha",
								},
							},
						},
					}))
				})
			})

			context("when checksum does not match", func() {
				it.Before(func() {
					err := ioutil.WriteFile(filepath.Join(layersDir, fmt.Sprintf("%s.toml", bundleinstall.LayerNameGems)), []byte(fmt.Sprintf(`[metadata]
					cache_sha = "some-sha-that-does-not-match"
					built_at = "%s"
					`, timeStamp.Format(time.RFC3339Nano))), 0600)
					Expect(err).NotTo(HaveOccurred())

					entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
						Name: "gems",
						Metadata: map[string]interface{}{
							"build": true,
						},
					}

					entryResolver.MergeLayerTypesCall.Returns.Launch = false
					entryResolver.MergeLayerTypesCall.Returns.Build = true
				})

				it("does the install process", func() {
					result, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						CNBPath:    cnbDir,
						Stack:      "some-stack",
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
					Expect(installProcess.ExecuteCall.CallCount).To(Equal(1))
					Expect(calculator.SumCall.CallCount).To(Equal(1))
					Expect(calculator.SumCall.Receives.Paths).To(Equal([]string{filepath.Join(workingDir, "Gemfile"), filepath.Join(workingDir, "Gemfile.lock")}))

					Expect(result).To(Equal(packit.BuildResult{
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
						Layers: []packit.Layer{
							{
								Name:      "gems",
								Path:      filepath.Join(layersDir, "gems"),
								LaunchEnv: packit.Environment{},
								BuildEnv:  packit.Environment{},
								SharedEnv: packit.Environment{
									"BUNDLE_PATH.default": filepath.Join(layersDir, "gems"),
								},
								Build:  true,
								Launch: false,
								Cache:  true,
								Metadata: map[string]interface{}{
									"built_at":  timeStamp.Format(time.RFC3339Nano),
									"cache_sha": "some-calculator-sha",
								},
							},
						},
					}))
				})
			})
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
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "gems",
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when install process returns an error", func() {
			it.Before(func() {
				installProcess.ExecuteCall.Returns.Error = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "gems",
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("some-error")))
			})
		})

		context("when the Gemfile.lock in the working dir is not readable", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(workingDir, 0644)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "gems",
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to stat Gemfile.lock")))
			})
		})
	})
}
