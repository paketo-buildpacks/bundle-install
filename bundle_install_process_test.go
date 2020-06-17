package bundleinstall_test

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/packit/pexec"
	bundleinstall "github.com/paketo-community/bundle-install"
	"github.com/paketo-community/bundle-install/fakes"
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

			installProcess = bundleinstall.NewBundleInstallProcess(executable)
		})

		it.After(func() {
			os.Setenv("PATH", path)

			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("runs the bundle install process", func() {
			err := installProcess.Execute(workingDir, "some-dir")
			Expect(err).NotTo(HaveOccurred())

			Expect(executions).To(HaveLen(2))
			Expect(executions[0].Args).To(Equal([]string{"config", "path", "some-dir"}))
			Expect(executions[1].Args).To(Equal([]string{"install"}))
		})

		context("failure cases", func() {
			context("when bundle config fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "config") {
							execution.Stdout.Write([]byte("stdout output"))
							execution.Stderr.Write([]byte("stderr output"))

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

			context("when bundle install fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						if strings.Contains(strings.Join(execution.Args, " "), "install") {
							execution.Stdout.Write([]byte("stdout output"))
							execution.Stderr.Write([]byte("stderr output"))

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
