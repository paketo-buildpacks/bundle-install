package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testReproducibleBuilds(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	var (
		image occam.Image

		name   string
		source string
	)

	it.Before(func() {
		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		source, err = occam.Source(filepath.Join("testdata", "reproducible_builds"))
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("creates a reproducible image", func() {
		build := pack.Build.
			WithBuildpacks(
				settings.Buildpacks.MRI.Online,
				settings.Buildpacks.Bundler.Online,
				settings.Buildpacks.BundleInstall.Online,
				settings.Buildpacks.BundleList.Online,
			).
			WithPullPolicy("never")

		var err error
		image, _, err = build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		firstID := image.ID

		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

		image, _, err = build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		Expect(firstID).To(Equal(image.ID))
	})

	context("when given the BP_KEEP_GEM_EXTENSION_BUILD_FILES env var", func() {
		it("does not create a reproducible build", func() {
			build := pack.Build.
				WithBuildpacks(
					settings.Buildpacks.MRI.Online,
					settings.Buildpacks.Bundler.Online,
					settings.Buildpacks.BundleInstall.Online,
					settings.Buildpacks.BundleList.Online,
				).
				WithEnv(map[string]string{"BP_KEEP_GEM_EXTENSION_BUILD_FILES": "true"}).
				WithPullPolicy("never")

			var err error
			image, _, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			firstID := image.ID

			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

			image, _, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(firstID).NotTo(Equal(image.ID))
		})
	})
}
