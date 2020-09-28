package bundleinstall_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/bundle-install/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir    string
		gemfileParser *fakes.VersionParser
		detect        packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(workingDir, "Gemfile"), []byte{}, 0644)
		Expect(err).NotTo(HaveOccurred())

		gemfileParser = &fakes.VersionParser{}

		detect = bundleinstall.Detect(gemfileParser)
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
					Name: "bundler",
					Metadata: bundleinstall.BuildPlanMetadata{
						Build: true,
					},
				},
				{
					Name: "mri",
					Metadata: bundleinstall.BuildPlanMetadata{
						Build: true,
					},
				},
			},
		}))
	})

	context("when the Gemfile specifies an mri ruby version", func() {
		it.Before(func() {
			gemfileParser.ParseVersionCall.Returns.Version = "2.6.x"
		})

		it("requires that version of mri", func() {
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
						Name: "bundler",
						Metadata: bundleinstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name:    "mri",
						Version: "2.6.x",
						Metadata: bundleinstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
		})
	})

	context("when the buildpack.yml parser fails", func() {
		it.Before(func() {
			gemfileParser.ParseVersionCall.Returns.Err = errors.New("some-error")
		})

		it("returns an error", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError("failed to parse Gemfile: some-error"))
		})
	})
}
