package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testOffline(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		image     occam.Image
		container occam.Container

		name   string
		source string
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()

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

	context("when building a simple app offline", func() {

		it("creates a working OCI image", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "offline_app"))
			Expect(err).NotTo(HaveOccurred())

			image, _, err = pack.WithVerbose().Build.
				WithBuildpacks(
					settings.Buildpacks.MRI.Offline,
					settings.Buildpacks.Bundler.Offline,
					settings.Buildpacks.BundleInstall.Offline,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithNetwork("none").
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

			Eventually(container, time.Second*30).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("9292")))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello world!"))
		})
	})

	context("when building a simple app offline with a non-default cache location", func() {
		it("creates a working OCI image", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "offline_app_non_default_cache"))
			Expect(err).NotTo(HaveOccurred())

			image, _, err = pack.WithVerbose().Build.
				WithBuildpacks(
					settings.Buildpacks.MRI.Offline,
					settings.Buildpacks.Bundler.Offline,
					settings.Buildpacks.BundleInstall.Offline,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithNetwork("none").
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

			Eventually(container, time.Second*30).Should(BeAvailable())
			Eventually(container).Should(Serve(ContainSubstring("Hello world!")).OnPort(9292))
		})
	})
}
