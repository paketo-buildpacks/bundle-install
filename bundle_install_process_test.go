package bundleinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/bundle-install/fakes"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBundleInstallProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir      string
		layerPath       string
		executions      []pexec.Execution
		executable      *fakes.Executable
		versionResolver *fakes.VersionResolver
		calculator      *fakes.Calculator

		installProcess bundleinstall.BundleInstallProcess
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		layerPath, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.RemoveAll(layerPath)).To(Succeed())

		executions = []pexec.Execution{}
		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			executions = append(executions, execution)

			return nil
		}

		logEmitter := bundleinstall.NewLogEmitter(bytes.NewBuffer(nil))
		versionResolver = &fakes.VersionResolver{}
		calculator = &fakes.Calculator{}

		installProcess = bundleinstall.NewBundleInstallProcess(executable, logEmitter, versionResolver, calculator)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(layerPath)).To(Succeed())
	})

	context("ShouldRun", func() {
		it.Before(func() {
			versionResolver.LookupCall.Returns.Version = "2.3.4"
			versionResolver.CompareMajorMinorCall.Returns.Bool = true

			calculator.SumCall.Returns.String = "other-checksum"

			Expect(os.WriteFile(filepath.Join(workingDir, "Gemfile.lock"), nil, 0600)).To(Succeed())
		})

		it("indicates that the install process should run", func() {
			ok, checksum, rubyVersion, err := installProcess.ShouldRun(map[string]interface{}{
				"cache_sha":    "some-checksum",
				"ruby_version": "1.2.3",
			}, workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(checksum).To(Equal("other-checksum"))
			Expect(rubyVersion).To(Equal("2.3.4"))

			Expect(versionResolver.LookupCall.CallCount).To(Equal(1))
			Expect(versionResolver.CompareMajorMinorCall.Receives.Left).To(Equal("1.2.3"))
			Expect(versionResolver.CompareMajorMinorCall.Receives.Right).To(Equal("2.3.4"))

			Expect(calculator.SumCall.Receives.Paths).To(Equal([]string{
				filepath.Join(workingDir, "Gemfile"),
				filepath.Join(workingDir, "Gemfile.lock"),
			}))
		})

		context("when the checksum matches, but the ruby version does not", func() {
			it.Before(func() {
				versionResolver.LookupCall.Returns.Version = "2.3.4"
				versionResolver.CompareMajorMinorCall.Returns.Bool = false

				calculator.SumCall.Returns.String = "some-checksum"
			})

			it("indicates that the install process should run", func() {
				ok, checksum, rubyVersion, err := installProcess.ShouldRun(map[string]interface{}{
					"cache_sha":    "some-checksum",
					"ruby_version": "1.2.3",
				}, workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(checksum).To(Equal("some-checksum"))
				Expect(rubyVersion).To(Equal("2.3.4"))
			})
		})

		context("when the checksum doesn't match, but the ruby version does", func() {
			it.Before(func() {
				versionResolver.LookupCall.Returns.Version = "1.2.3"
				versionResolver.CompareMajorMinorCall.Returns.Bool = true

				calculator.SumCall.Returns.String = "other-checksum"
			})

			it("indicates that the install process should run", func() {
				ok, checksum, rubyVersion, err := installProcess.ShouldRun(map[string]interface{}{
					"cache_sha":    "some-checksum",
					"ruby_version": "1.2.3",
				}, workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(checksum).To(Equal("other-checksum"))
				Expect(rubyVersion).To(Equal("1.2.3"))
			})
		})

		context("when the checksum and ruby version matches", func() {
			it.Before(func() {
				versionResolver.LookupCall.Returns.Version = "1.2.3"
				versionResolver.CompareMajorMinorCall.Returns.Bool = true

				calculator.SumCall.Returns.String = "some-checksum"
			})

			it("indicates that the install process should not run", func() {
				ok, checksum, rubyVersion, err := installProcess.ShouldRun(map[string]interface{}{
					"cache_sha":    "some-checksum",
					"ruby_version": "1.2.3",
				}, workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeFalse())
				Expect(checksum).To(Equal("some-checksum"))
				Expect(rubyVersion).To(Equal("1.2.3"))
			})
		})

		context("failure cases", func() {
			context("when the ruby version cannot be looked up", func() {
				it.Before(func() {
					versionResolver.LookupCall.Returns.Err = errors.New("failed to lookup ruby version")
				})

				it("returns an error", func() {
					_, _, _, err := installProcess.ShouldRun(map[string]interface{}{
						"cache_sha":    "some-checksum",
						"ruby_version": "1.2.3",
					}, workingDir)
					Expect(err).To(MatchError("failed to lookup ruby version"))
				})
			})

			context("when the ruby version cannot be compared", func() {
				it.Before(func() {
					versionResolver.CompareMajorMinorCall.Returns.Error = errors.New("failed to compare ruby version")
				})

				it("returns an error", func() {
					_, _, _, err := installProcess.ShouldRun(map[string]interface{}{
						"cache_sha":    "some-checksum",
						"ruby_version": "1.2.3",
					}, workingDir)
					Expect(err).To(MatchError("failed to compare ruby version"))
				})
			})

			context("when the Gemfile.lock cannot be stat'd", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, _, _, err := installProcess.ShouldRun(map[string]interface{}{
						"cache_sha":    "some-checksum",
						"ruby_version": "1.2.3",
					}, workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when a checksum cannot be calculated", func() {
				it.Before(func() {
					calculator.SumCall.Returns.Error = errors.New("failed to calculate checksum")
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, _, _, err := installProcess.ShouldRun(map[string]interface{}{
						"cache_sha":    "some-checksum",
						"ruby_version": "1.2.3",
					}, workingDir)
					Expect(err).To(MatchError("failed to calculate checksum"))
				})
			})
		})
	})

	context("Execute", func() {
		context("when there is no vendor/cache directory present", func() {
			it("runs the bundle install process", func() {
				err := installProcess.Execute(workingDir, layerPath, map[string]string{"path": "some-dir"})
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(3))

				Expect(executions[0].Args).To(Equal([]string{"config", "--global", "path", "some-dir"}))
				Expect(executions[0].Env).To(ContainElement(fmt.Sprintf("BUNDLE_USER_CONFIG=%s", filepath.Join(layerPath, "config"))))

				Expect(executions[1].Args).To(Equal([]string{"config", "--global", "cache_path", "--parseable"}))
				Expect(executions[1].Env).To(ContainElement(fmt.Sprintf("BUNDLE_USER_CONFIG=%s", filepath.Join(layerPath, "config"))))

				Expect(executions[2].Args).To(Equal([]string{"install"}))
				Expect(executions[2].Env).To(ContainElement(fmt.Sprintf("BUNDLE_USER_CONFIG=%s", filepath.Join(layerPath, "config"))))
			})
		})

		context("when there is a vendor/cache directory present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "vendor", "cache"), os.ModePerm)).To(Succeed())
			})

			it("runs the bundle install process", func() {
				err := installProcess.Execute(workingDir, layerPath, map[string]string{"clean": "true"})
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(3))
				Expect(executions[0].Args).To(Equal([]string{"config", "--global", "clean", "true"}))
				Expect(executions[1].Args).To(Equal([]string{"config", "--global", "cache_path", "--parseable"}))
				Expect(executions[2].Args).To(Equal([]string{"install", "--local"}))
			})
		})

		context("when the vendor/cache directory is in a non-default location", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "other_dir", "other_cache"), os.ModePerm)).To(Succeed())
				executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
					executions = append(executions, execution)

					if strings.Contains(strings.Join(execution.Args, " "), "config --global cache_path --parseable") {
						fmt.Fprintf(execution.Stdout, "cache_path=other_dir/other_cache")
					}

					return nil
				}
			})

			it("runs the bundle install process", func() {
				err := installProcess.Execute(workingDir, layerPath, map[string]string{
					"without": "development:test",
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(3))
				Expect(executions[0].Args).To(Equal([]string{"config", "--global", "without", "development:test"}))
				Expect(executions[1].Args).To(Equal([]string{"config", "--global", "cache_path", "--parseable"}))
				Expect(executions[2].Args).To(Equal([]string{"install", "--local"}))
			})
		})

		context("when there is local bundle config", func() {
			it.Before(func() {
				Expect(os.Mkdir(filepath.Join(workingDir, ".bundle"), os.ModePerm)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, ".bundle", "config"), []byte("some-bundle-config"), 0600)).To(Succeed())
			})

			it("copies that config into the global config", func() {
				err := installProcess.Execute(workingDir, layerPath, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(2))
				Expect(executions[0].Args).To(Equal([]string{"config", "--global", "cache_path", "--parseable"}))
				Expect(executions[1].Args).To(Equal([]string{"install"}))

				contents, err := os.ReadFile(filepath.Join(layerPath, "config"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal("some-bundle-config"))
			})

			it("makes a backup of that local config", func() {
				err := installProcess.Execute(workingDir, layerPath, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(2))
				Expect(executions[0].Args).To(Equal([]string{"config", "--global", "cache_path", "--parseable"}))
				Expect(executions[1].Args).To(Equal([]string{"install"}))

				contents, err := os.ReadFile(filepath.Join(workingDir, ".bundle", "config.bak"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(Equal("some-bundle-config"))
			})

			context("when there is also a backup of the config", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, ".bundle", "config.bak"), []byte("other-bundle-config"), 0600)).To(Succeed())
				})

				it("replaces the local config with the backup", func() {
					err := installProcess.Execute(workingDir, layerPath, nil)
					Expect(err).NotTo(HaveOccurred())

					Expect(executions).To(HaveLen(2))
					Expect(executions[0].Args).To(Equal([]string{"config", "--global", "cache_path", "--parseable"}))
					Expect(executions[1].Args).To(Equal([]string{"install"}))

					contents, err := os.ReadFile(filepath.Join(layerPath, "config"))
					Expect(err).NotTo(HaveOccurred())
					Expect(string(contents)).To(Equal("other-bundle-config"))

					contents, err = os.ReadFile(filepath.Join(workingDir, ".bundle", "config"))
					Expect(err).NotTo(HaveOccurred())
					Expect(string(contents)).To(Equal("other-bundle-config"))
				})
			})
		})

		context("failure cases", func() {
			context("when the config cannot be copied into the layer", func() {
				it.Before(func() {
					Expect(os.Mkdir(filepath.Join(workingDir, ".bundle"), os.ModePerm)).To(Succeed())
					Expect(os.WriteFile(filepath.Join(workingDir, ".bundle", "config"), nil, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					err := installProcess.Execute(workingDir, layerPath, nil)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when bundle config fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "config --global path") {
							fmt.Fprint(execution.Stdout, "stdout output")
							fmt.Fprint(execution.Stderr, "stderr output")

							return errors.New("bundle config path failed")
						}

						return nil
					}
				})

				it("prints the execution output and returns an error", func() {
					err := installProcess.Execute(workingDir, layerPath, map[string]string{"path": "some-dir"})
					Expect(err).To(MatchError(ContainSubstring("failed to execute bundle config")))
					Expect(err).To(MatchError(ContainSubstring("bundle config path failed")))
				})
			})

			context("when the vendor/cache directory is un-statable", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, "vendor"), 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, "vendor"), os.ModePerm)).To(Succeed())
				})

				it("runs the bundle install process", func() {
					err := installProcess.Execute(workingDir, layerPath, map[string]string{"path": "some-dir"})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when bundle install fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "install") {
							fmt.Fprint(execution.Stdout, "stdout output")
							fmt.Fprint(execution.Stderr, "stderr output")

							return errors.New("bundle install failed")
						}

						return nil
					}
				})

				it("prints the execution output and returns an error", func() {
					err := installProcess.Execute(workingDir, layerPath, map[string]string{"path": "some-dir"})
					Expect(err).To(MatchError(ContainSubstring("failed to execute bundle install")))
					Expect(err).To(MatchError(ContainSubstring("bundle install failed")))
				})
			})
		})
	})
}
