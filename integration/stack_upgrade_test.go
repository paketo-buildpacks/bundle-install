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

func testStackUpgrade(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		docker occam.Docker
		pack   occam.Pack

		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}

		name   string
		source string
	)

	it.Before(func() {
		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		docker = occam.NewDocker()
		pack = occam.NewPack()
		imageIDs = map[string]struct{}{}
		containerIDs = map[string]struct{}{}

		Expect(docker.Pull.Execute("paketobuildpacks/builder-jammy-buildpackless-base:latest")).To(Succeed())
		Expect(docker.Pull.Execute("paketobuildpacks/run-jammy-base:latest")).To(Succeed())
	})

	it.After(func() {
		for id := range containerIDs {
			Expect(docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id := range imageIDs {
			Expect(docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(docker.Image.Remove.Execute("paketobuildpacks/builder-jammy-buildpackless-base:latest")).To(Succeed())
		Expect(docker.Image.Remove.Execute("paketobuildpacks/run-jammy-base:latest")).To(Succeed())

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when an app is rebuilt and the underlying stack changes", func() {
		it("rebuilds the layer", func() {
			var (
				err         error
				logs        fmt.Stringer
				firstImage  occam.Image
				secondImage occam.Image

				firstContainer  occam.Container
				secondContainer occam.Container
			)

			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.MRI.Online,
					settings.Buildpacks.Bundler.Online,
					settings.Buildpacks.BundleInstall.Online,
					settings.Buildpacks.BundleList.Online,
				)

			firstImage, _, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(4))

			Expect(firstImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[2].Layers).To(HaveKey("launch-gems"))

			firstContainer, err = docker.Container.Run.
				WithCommand("bundle exec rackup -o 0.0.0.0").
				WithEnv(map[string]string{"PORT": "9292"}).
				WithPublish("9292").
				WithPublishAll().
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[firstContainer.ID] = struct{}{}
			Eventually(firstContainer).Should(BeAvailable())

			// Second pack build, upgrade stack image
			secondImage, logs, err = build.WithBuilder("paketobuildpacks/builder-jammy-buildpackless-base").Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(4))

			Expect(secondImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[2].Layers).To(HaveKey("launch-gems"))

			Expect(logs.String()).To(ContainSubstring("  Executing launch environment install process"))
			Expect(logs.String()).To(ContainSubstring("  Executing build environment install process"))
			Expect(logs.String()).NotTo(ContainSubstring(fmt.Sprintf("  Reusing cached layer /layers/%s/build-gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))))
			Expect(logs.String()).NotTo(ContainSubstring(fmt.Sprintf("  Reusing cached layer /layers/%s/launch-gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))))

			secondContainer, err = docker.Container.Run.
				WithCommand("bundle exec rackup -o 0.0.0.0").
				WithEnv(map[string]string{"PORT": "9292"}).
				WithPublish("9292").
				WithPublishAll().
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}
			Eventually(secondContainer).Should(BeAvailable())

			Expect(secondImage.Buildpacks[2].Layers["launch-gems"].SHA).NotTo(Equal(firstImage.Buildpacks[2].Layers["launch-gems"].SHA))
		})
	})
}
