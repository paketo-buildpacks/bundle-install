package bundleinstall_test

import (
	"errors"
	"fmt"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/bundle-install/fakes"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRubyVersionResolver(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("RubyVersionResolver", func() {
		var (
			executable *fakes.Executable

			rubyVersionResolver bundleinstall.RubyVersionResolver
		)

		it.Before(func() {
			executable = &fakes.Executable{}
			executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
				fmt.Fprintf(execution.Stdout, "ruby 2.7.7p57 (2018-03-29 revision 63029) [x86_64-linux-gnu]")
				return nil
			}

			rubyVersionResolver = bundleinstall.NewRubyVersionResolver(executable)
		})

		context("Lookup", func() {

			it("returns the ruby version", func() {
				version, err := rubyVersionResolver.Lookup()
				Expect(err).NotTo(HaveOccurred())

				Expect(version).To(Equal("2.7.7"))

				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{"--version"}))
			})

			context("failure cases", func() {
				context("fails to execute `ruby --version`", func() {
					it.Before(func() {
						executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
							fmt.Fprintf(execution.Stderr, "failed to execute")
							return errors.New("exit status 1")
						}
					})

					it("returns an error", func() {
						_, err := rubyVersionResolver.Lookup()
						Expect(err).To(MatchError(ContainSubstring("failed to obtain ruby version")))
						Expect(err).To(MatchError(ContainSubstring("exit status 1")))
						Expect(err).To(MatchError(ContainSubstring("failed to execute")))
					})
				})

				context("no ruby match is found", func() {
					it.Before(func() {
						executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
							fmt.Fprintf(execution.Stdout, "")
							return nil
						}
					})

					it("returns an error", func() {
						_, err := rubyVersionResolver.Lookup()
						Expect(err).To(MatchError(ContainSubstring("no string matching 'ruby (\\d+\\.\\d+\\.\\d+)' found")))
					})
				})
			})
		})

		context("CompareMajorMinor", func() {
			context("versions are an exact match", func() {
				it("returns true", func() {
					Expect(rubyVersionResolver.CompareMajorMinor("1.2.3", "1.2.3")).To(BeTrue())
				})
			})

			context("the patch versions differ", func() {
				it("returns true", func() {
					Expect(rubyVersionResolver.CompareMajorMinor("1.2.3", "1.2.4")).To(BeTrue())
				})
			})

			context("the minor versions differ", func() {
				it("returns false", func() {
					Expect(rubyVersionResolver.CompareMajorMinor("1.2.3", "1.3.5")).To(BeFalse())
				})
			})

			context("the major versions differ", func() {
				it("returns false", func() {
					Expect(rubyVersionResolver.CompareMajorMinor("1.2.3", "2.5.5")).To(BeFalse())
				})
			})

			context("failure cases", func() {
				context("cached version is not semver compatible", func() {
					it("returns an error", func() {
						_, err := rubyVersionResolver.CompareMajorMinor("weird-version", "1.2.3")
						Expect(err).To(HaveOccurred())
					})
				})

				context("new version version is not semver compatible", func() {
					it("returns an error", func() {
						_, err := rubyVersionResolver.CompareMajorMinor("1.2.3", "weird-version")
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})
	})
}
