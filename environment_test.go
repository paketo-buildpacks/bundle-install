package bundleinstall_test

import (
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testEnvironment(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("ParseEnvironment", func() {
		it("parse the environment variables", func() {
			environment, err := bundleinstall.ParseEnvironment([]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(environment).To(Equal(bundleinstall.Environment{
				KeepGemExtensionBuildFiles: false,
			}))
		})

		context("when BP_KEEP_GEM_EXTENSION_BUILD_FILES is set", func() {
			it("parse the environment variables", func() {
				environment, err := bundleinstall.ParseEnvironment([]string{
					"BP_KEEP_GEM_EXTENSION_BUILD_FILES=true",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(environment).To(Equal(bundleinstall.Environment{
					KeepGemExtensionBuildFiles: true,
				}))
			})
		})

		context("failure cases", func() {
			context("when the BP_KEEP_GEM_EXTENSION_BUILD_FILES env var cannot be parsed", func() {
				it("returns an error", func() {
					_, err := bundleinstall.ParseEnvironment([]string{
						"BP_KEEP_GEM_EXTENSION_BUILD_FILES=banana",
					})
					Expect(err).To(MatchError(ContainSubstring("failed to parse BP_KEEP_GEM_EXTENSION_BUILD_FILES:")))
					Expect(err).To(MatchError(ContainSubstring(`parsing "banana": invalid syntax`)))
				})
			})
		})
	})
}
