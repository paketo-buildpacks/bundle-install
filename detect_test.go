package bundleinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/bundle-install/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir            string
		gemfileParser         *fakes.VersionParser
		rubyVersionFileParser *fakes.VersionParser
		detect                packit.DetectFunc
		buffer                *bytes.Buffer
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(filepath.Join(workingDir, "Gemfile"), []byte{}, 0644)
		Expect(err).NotTo(HaveOccurred())

		gemfileParser = &fakes.VersionParser{}
		rubyVersionFileParser = &fakes.VersionParser{}
		buffer = bytes.NewBuffer(nil)

		detect = bundleinstall.Detect(gemfileParser, rubyVersionFileParser, scribe.NewEmitter(buffer))
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
						Name: "mri",
						Metadata: bundleinstall.BuildPlanMetadata{
							Version:       "2.6.x",
							VersionSource: "Gemfile",
							Build:         true,
						},
					},
				},
			}))
		})
	})

	context("when a Gemfile does not exist", func() {
		it.Before(func() {
			_, err := os.Stat("/no/gemfile")
			gemfileParser.ParseVersionCall.Returns.Err = fmt.Errorf("failed to parse Gemfile: %w", err)
		})

		it("fails detection", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail.WithMessage("Gemfile is not present")))
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
			Expect(err).To(MatchError("some-error"))
		})
	})

	context("when the Gemfile has no version it falls back to .ruby-version", func() {
		it.Before(func() {
			gemfileParser.ParseVersionCall.Returns.Version = ""
			rubyVersionFileParser.ParseVersionCall.Returns.Version = "3.2.2"
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
						Name: "mri",
						Metadata: bundleinstall.BuildPlanMetadata{
							Version:       "3.2.2",
							VersionSource: ".ruby-version",
							Build:         true,
						},
					},
				},
			}))
		})
	})

	context("when the Gemfile has no version and the .ruby-version is invalid", func() {
		it.Before(func() {
			gemfileParser.ParseVersionCall.Returns.Version = ""
			rubyVersionFileParser.ParseVersionCall.Returns.Err = errors.New("invalid version")
		})

		it("the mri version is empty", func() {
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
							Version:       "",
							VersionSource: "",
							Build:         true,
						},
					},
				},
			}))
			Expect(buffer.String()).To(ContainSubstring("Could not parse the .ruby-version file"))
		})
	})
}
