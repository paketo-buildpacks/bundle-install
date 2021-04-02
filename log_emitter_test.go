package bundleinstall_test

import (
	"bytes"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
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
			emitter.Environment(packit.Environment{
				"BUNDLE_USER_CONFIG.default": "/some/path",
			})

			Expect(buffer.String()).To(ContainSubstring("  Configuring environment"))
			Expect(buffer.String()).To(ContainSubstring("    BUNDLE_USER_CONFIG -> \"/some/path\""))
		})
	})
}
