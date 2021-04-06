package bundleinstall_test

import (
	"bytes"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLogEmitter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer  *bytes.Buffer
		emitter bundleinstall.LogEmitter
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)
		emitter = bundleinstall.NewLogEmitter(buffer)
	})

	context("Environment", func() {
		it("prints details about the environment", func() {
			emitter.Environment(packit.Layer{
				BuildEnv: packit.Environment{
					"BUNDLE_USER_CONFIG.default": "/some/path",
				},
				LaunchEnv: packit.Environment{
					"BUNDLE_USER_CONFIG.default": "/other/path",
				},
			})

			Expect(buffer).To(ContainLines(
				"  Configuring build environment",
				`    BUNDLE_USER_CONFIG -> "/some/path"`,
				"",
				"  Configuring launch environment",
				`    BUNDLE_USER_CONFIG -> "/other/path"`,
				"",
			))
		})
	})
}
