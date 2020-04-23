package bundle_test

import (
	"os"
	"testing"

	"io/ioutil"

	"github.com/cloudfoundry/packit"
	. "github.com/onsi/gomega"
	"github.com/paketo-community/bundle-install/bundle"
	"github.com/sclevine/spec"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		workingDir string
		detect     packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())
		detect = bundle.Detect()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a plan that provides gems", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{Name: "gems"},
			},
			Requires: []packit.BuildPlanRequirement{
				{
					Name: "gems",
					Metadata: bundle.BuildPlanMetadata{
						Launch: true,
					},
				},
			},
		}))

	})
}
