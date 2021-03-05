package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testSimpleApp(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building a simple app", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("creates a working OCI image", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			image, _, err = pack.WithVerbose().Build.
				WithBuildpacks(
					settings.Buildpacks.MRI.Online,
					settings.Buildpacks.Bundler.Online,
					settings.Buildpacks.BundleInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithCommand("bundle exec rackup -o 0.0.0.0").
				WithEnv(map[string]string{"PORT": "9292"}).
				WithPublish("9292").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())
			Eventually(container).Should(Serve(ContainSubstring("Hello world!")).OnPort(9292))
		})

		context("the version of bundler in the Gemfile.lock is 1.17.x", func() {
			it("creates a working OCI image", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "bundler_version_1_17"))
				Expect(err).NotTo(HaveOccurred())

				image, _, err = pack.WithVerbose().Build.
					WithBuildpacks(
						settings.Buildpacks.MRI.Online,
						settings.Buildpacks.Bundler.Online,
						settings.Buildpacks.BundleInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithPullPolicy("never").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithCommand("bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())
				Eventually(container).Should(Serve(ContainSubstring("Hello world!")).OnPort(9292))
			})
		})

		context("when the Gemfile contains development and test dependencies", func() {
			it("does not install those dependencies", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "simple_app"))
				Expect(err).NotTo(HaveOccurred())

				image, _, err = pack.WithVerbose().Build.
					WithBuildpacks(
						settings.Buildpacks.MRI.Online,
						settings.Buildpacks.Bundler.Online,
						settings.Buildpacks.BundleInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithPullPolicy("never").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithCommand("bundle list && bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())

				logs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs.String()).To(ContainSubstring("sinatra"))
				Expect(logs.String()).NotTo(ContainSubstring("rspec"))
				Expect(logs.String()).NotTo(ContainSubstring("pry"))
			})
		})
	})
}
