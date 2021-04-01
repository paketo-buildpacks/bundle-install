package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("logs useful information for the user", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.MRI.Online,
					settings.Buildpacks.Bundler.Online,
					settings.Buildpacks.BundleInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  Executing build process",
				"    Running 'bundle config --global clean true'",
				MatchRegexp(fmt.Sprintf("    Running 'bundle config --global path /layers/%s/gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				"    Running 'bundle config --global without development:test'",
				"    Running 'bundle config --global cache_path --parseable'",
				"    Running 'bundle install'",
				MatchRegexp(`      Completed in \d+\.?\d*`),
				"",
				"  Configuring environment",
				MatchRegexp(fmt.Sprintf(`    BUNDLE_USER_CONFIG -> "/layers/%s/gems/config"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				"",
			))
		})
	})
}
