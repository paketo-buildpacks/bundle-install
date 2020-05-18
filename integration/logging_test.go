package integration_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLogging(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose()
		docker = occam.NewDocker()
	})

	context("when the buildpack is run with pack build", func() {
		var (
			image occam.Image
			name  string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		})

		it("logs useful information for the user", func() {
			var err error
			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithNoPull().
				WithBuildpacks(mriURI, bundlerURI, bundleInstallURI, buildPlanURI).
				Execute(name, filepath.Join("testdata", "simple_app"))
			Expect(err).ToNot(HaveOccurred(), logs.String)

			buildpackVersion, err := GetGitVersion()
			Expect(err).ToNot(HaveOccurred())

			Expect(logs).To(ContainLines(
				fmt.Sprintf("Bundle Install Buildpack %s", buildpackVersion),
				"  Executing build process",
				`    Running 'bundle install'`,
				MatchRegexp(`      Completed in \d+\.?\d*`),
				"",
				"  Configuring environment",
				`    BUNDLE_PATH -> "/layers/paketo-community_bundle-install/gems"`,
			))
		})
	})
}
