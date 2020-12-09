package bundleinstall_test

import (
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPlanEntryResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		resolver bundleinstall.PlanEntryResolver
	)

	it.Before(func() {
		resolver = bundleinstall.NewPlanEntryResolver()
	})

	context("when a buildpack.yml entry and BP_MRI_VERSION are included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "gems",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
				{
					Name: "gems",
					Metadata: map[string]interface{}{
						"launch": true,
					},
				},
				{
					Name: "gems",
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "gems",
				Metadata: map[string]interface{}{
					"build":  true,
					"launch": true,
				},
			}))
		})
	})
}
