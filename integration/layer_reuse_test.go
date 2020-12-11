package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLayerReuse(t *testing.T, context spec.G, it spec.S) {
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
	})

	it.After(func() {
		for id := range containerIDs {
			Expect(docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id := range imageIDs {
			Expect(docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when an app is rebuilt and does not change", func() {
		it("reuses a layer from a previous build", func() {
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
					settings.Buildpacks.BuildPlan.Online,
				)

			firstImage, logs, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(4))

			Expect(firstImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
			Expect(firstImage.Buildpacks[2].Layers).To(HaveKey("gems"))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  Executing build process",
				MatchRegexp(fmt.Sprintf("    Running 'bundle config path /layers/%s/gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				"    Running 'bundle config clean true'",
				"    Running 'bundle config cache_path --parseable'",
				"    Running 'bundle install'",
				MatchRegexp(`      Completed in \d+\.?\d*`),
				"",
				"  Configuring environment",
				MatchRegexp(fmt.Sprintf(`    BUNDLE_PATH -> "/layers/%s/gems"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
			))

			firstContainer, err = docker.Container.Run.
				WithCommand("bundle exec rackup -o 0.0.0.0").
				WithEnv(map[string]string{"PORT": "9292"}).
				WithPublish("9292").
				WithPublishAll().
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[firstContainer.ID] = struct{}{}

			Eventually(firstContainer).Should(BeAvailable())

			// Second pack build
			secondImage, logs, err = build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(4))

			Expect(secondImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
			Expect(secondImage.Buildpacks[2].Layers).To(HaveKey("gems"))

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				MatchRegexp(fmt.Sprintf("  Reusing cached layer /layers/%s/gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
			))

			secondContainer, err = docker.Container.Run.
				WithCommand("bundle exec rackup -o 0.0.0.0").
				WithEnv(map[string]string{"PORT": "9292"}).
				WithPublish("9292").
				WithPublishAll().
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}

			Eventually(secondContainer).Should(BeAvailable())

			Expect(secondImage.Buildpacks[2].Layers["gems"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[2].Layers["gems"].Metadata["built_at"]))
		})
	})

	context("when an app is rebuilt and there is a change", func() {
		context("when the Gemfile.lock changes", func() {
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
						settings.Buildpacks.BuildPlan.Online,
					)

				firstImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				imageIDs[firstImage.ID] = struct{}{}

				Expect(firstImage.Buildpacks).To(HaveLen(4))

				Expect(firstImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
				Expect(firstImage.Buildpacks[2].Layers).To(HaveKey("gems"))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
					"  Executing build process",
					MatchRegexp(fmt.Sprintf("    Running 'bundle config path /layers/%s/gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
					"    Running 'bundle config clean true'",
					"    Running 'bundle config cache_path --parseable'",
					"    Running 'bundle install'",
					MatchRegexp(`      Completed in \d+\.?\d*`),
					"",
					"  Configuring environment",
					MatchRegexp(fmt.Sprintf(`    BUNDLE_PATH -> "/layers/%s/gems"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				))

				firstContainer, err = docker.Container.Run.
					WithCommand("bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(firstImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[firstContainer.ID] = struct{}{}

				Eventually(firstContainer).Should(BeAvailable())

				contents, err := ioutil.ReadFile(filepath.Join(source, "Gemfile.lock"))
				Expect(err).NotTo(HaveOccurred())

				err = ioutil.WriteFile(filepath.Join(source, "Gemfile.lock"),
					[]byte(string(contents)+"\nbreak checksum"), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Second pack build
				secondImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				imageIDs[secondImage.ID] = struct{}{}

				Expect(secondImage.Buildpacks).To(HaveLen(4))

				Expect(secondImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
				Expect(secondImage.Buildpacks[2].Layers).To(HaveKey("gems"))

				Expect(logs.String()).NotTo(ContainSubstring(fmt.Sprintf("  Reusing cached layer /layers/%s/gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))))

				secondContainer, err = docker.Container.Run.
					WithCommand("bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(secondImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[secondContainer.ID] = struct{}{}

				Eventually(secondContainer).Should(BeAvailable())

				Expect(secondImage.Buildpacks[2].Layers["gems"].Metadata["built_at"]).NotTo(Equal(firstImage.Buildpacks[2].Layers["gems"].Metadata["built_at"]))
			})
		})

		context("when the Gemfile changes", func() {
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
						settings.Buildpacks.BuildPlan.Online,
					)

				firstImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				imageIDs[firstImage.ID] = struct{}{}

				Expect(firstImage.Buildpacks).To(HaveLen(4))

				Expect(firstImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
				Expect(firstImage.Buildpacks[2].Layers).To(HaveKey("gems"))

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
					"  Executing build process",
					MatchRegexp(fmt.Sprintf("    Running 'bundle config path /layers/%s/gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
					"    Running 'bundle config clean true'",
					"    Running 'bundle config cache_path --parseable'",
					"    Running 'bundle install'",
					MatchRegexp(`      Completed in \d+\.?\d*`),
					"",
					"  Configuring environment",
					MatchRegexp(fmt.Sprintf(`    BUNDLE_PATH -> "/layers/%s/gems"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				))

				firstContainer, err = docker.Container.Run.
					WithCommand("bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(firstImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[firstContainer.ID] = struct{}{}

				Eventually(firstContainer).Should(BeAvailable())

				contents, err := ioutil.ReadFile(filepath.Join(source, "Gemfile"))
				Expect(err).NotTo(HaveOccurred())

				err = ioutil.WriteFile(filepath.Join(source, "Gemfile"),
					[]byte(strings.ReplaceAll(string(contents),
						`gem 'sinatra', '~>2.1.0'`,
						`gem 'sinatra', '~>2.0.8'`,
					)), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Second pack build
				secondImage, logs, err = build.Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				imageIDs[secondImage.ID] = struct{}{}

				Expect(secondImage.Buildpacks).To(HaveLen(4))

				Expect(secondImage.Buildpacks[2].Key).To(Equal(settings.Buildpack.ID))
				Expect(secondImage.Buildpacks[2].Layers).To(HaveKey("gems"))

				Expect(logs.String()).To(ContainSubstring("  Executing build process"))
				Expect(logs.String()).NotTo(ContainSubstring(fmt.Sprintf("  Reusing cached layer /layers/%s/gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))))

				secondContainer, err = docker.Container.Run.
					WithCommand("bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(secondImage.ID)
				Expect(err).NotTo(HaveOccurred())

				containerIDs[secondContainer.ID] = struct{}{}

				Eventually(secondContainer).Should(BeAvailable())

				Expect(secondImage.Buildpacks[2].Layers["gems"].Metadata["built_at"]).NotTo(Equal(firstImage.Buildpacks[2].Layers["gems"].Metadata["built_at"]))
			})
		})
	})
}
