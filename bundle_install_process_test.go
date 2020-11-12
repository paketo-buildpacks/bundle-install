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
	var Expect = NewWithT(t).Expect

	context("Execute", func() {
		var (
			workingDir string
			path       string
			executions []pexec.Execution
			executable *fakes.Executable

			installProcess bundleinstall.BundleInstallProcess
		)

		it.Before(func() {
			var err error
			workingDir, err = ioutil.TempDir("", "working-dir")
			Expect(err).NotTo(HaveOccurred())

			executions = []pexec.Execution{}
			executable = &fakes.Executable{}
			executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
				executions = append(executions, execution)

				return nil
			}

			path = os.Getenv("PATH")
			os.Setenv("PATH", "/some/bin")

			logEmitter := bundleinstall.NewLogEmitter(bytes.NewBuffer(nil))

			installProcess = bundleinstall.NewBundleInstallProcess(executable, logEmitter)
		})

		it.After(func() {
			os.Setenv("PATH", path)

			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		context("when there is no vendor/cache directory present", func() {
			it("runs the bundle install process", func() {
				err := installProcess.Execute(workingDir, "some-dir")
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(4))
				Expect(executions[0].Args).To(Equal([]string{"config", "path", "some-dir"}))
				Expect(executions[1].Args).To(Equal([]string{"config", "clean", "true"}))
				Expect(executions[2].Args).To(Equal([]string{"config", "cache_path", "--parseable"}))
				Expect(executions[3].Args).To(Equal([]string{"install"}))
			})
		})

		context("when there is a vendor/cache directory present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "vendor", "cache"), os.ModePerm)).To(Succeed())
			})

			it("runs the bundle install process", func() {
				err := installProcess.Execute(workingDir, "some-dir")
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(4))
				Expect(executions[0].Args).To(Equal([]string{"config", "path", "some-dir"}))
				Expect(executions[1].Args).To(Equal([]string{"config", "clean", "true"}))
				Expect(executions[2].Args).To(Equal([]string{"config", "cache_path", "--parseable"}))
				Expect(executions[3].Args).To(Equal([]string{"install", "--local"}))
			})
		})

		context("when the vendor/cache directory is in a non-default location", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "other_dir", "other_cache"), os.ModePerm)).To(Succeed())
				executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
					executions = append(executions, execution)

					if strings.Contains(strings.Join(execution.Args, " "), "config cache_path --parseable") {
						fmt.Fprintf(execution.Stdout, "cache_path=other_dir/other_cache")
					}

					return nil
				}
			})

			it("runs the bundle install process", func() {
				err := installProcess.Execute(workingDir, "some-dir")
				Expect(err).NotTo(HaveOccurred())

				Expect(executions).To(HaveLen(4))
				Expect(executions[0].Args).To(Equal([]string{"config", "path", "some-dir"}))
				Expect(executions[1].Args).To(Equal([]string{"config", "clean", "true"}))
				Expect(executions[2].Args).To(Equal([]string{"config", "cache_path", "--parseable"}))
				Expect(executions[3].Args).To(Equal([]string{"install", "--local"}))
			})
		})

		context("failure cases", func() {
			context("when bundle config fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "config") {
							fmt.Fprint(execution.Stdout, "stdout output")
							fmt.Fprint(execution.Stderr, "stderr output")

							return errors.New("bundle config failed")
						}

						return nil
					}
				})
				it("prints the execution output and returns an error", func() {
					err := installProcess.Execute(workingDir, "some-dir")
					Expect(err).To(MatchError(ContainSubstring("failed to execute bundle config")))
					Expect(err).To(MatchError(ContainSubstring("bundle config failed")))
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
					err := installProcess.Execute(workingDir, "some-dir")
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
					err := installProcess.Execute(workingDir, "some-dir")
					Expect(err).To(MatchError(ContainSubstring("failed to execute bundle install")))
					Expect(err).To(MatchError(ContainSubstring("bundle install failed")))
				})
			})
		})
	})
}
