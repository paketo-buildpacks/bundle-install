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

			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("creates a working OCI image", func() {
			var logs fmt.Stringer
			var err error

			image, logs, err = pack.Build.
				WithBuildpacks(
					settings.Buildpacks.MRI.Online,
					settings.Buildpacks.Bundler.Online,
					settings.Buildpacks.BundleInstall.Online,
					settings.Buildpacks.BundleList.Online,
				).
				WithEnv(map[string]string{"BP_LOG_LEVEL": "DEBUG"}).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
				"  Getting the layer associated with build-gems",
				fmt.Sprintf("    /layers/%s/build-gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
			Expect(logs).To(ContainLines(
				"  Checking if the build environment install process should run",
			))
			Expect(logs).To(ContainLines(
				"  Executing build environment install process",
				"    Setting up bundle install config paths:",
				"      Local config path: /workspace/.bundle/config",
				"      Backup config path: /workspace/.bundle/config.bak",
				MatchRegexp(fmt.Sprintf(`      Global config path: /layers/%s/build-gems/config`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				"    Adding global config path to $BUNDLE_USER_CONFIG",
			))
			Expect(logs).To(ContainLines(
				"    Running 'bundle config --global clean true'",
			))
			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf("    Running 'bundle config --global path /layers/%s/build-gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
			))
			Expect(logs).To(ContainLines("    Running 'bundle config --global cache_path --parseable'"))
			Expect(logs).To(ContainLines("    Running 'bundle install'"))
			Expect(logs).To(ContainLines(
				MatchRegexp(`      Completed in \d+\.?\d*`),
			))
			Expect(logs).To(ContainLines(
				"  Getting the layer associated with launch-gems",
				fmt.Sprintf("    /layers/%s/launch-gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
			Expect(logs).To(ContainLines(
				"  Checking if the launch environment install process should run",
			))
			Expect(logs).To(ContainLines(
				"  Executing launch environment install process",
				"    Setting up bundle install config paths:",
				"      Local config path: /workspace/.bundle/config",
				"      Backup config path: /workspace/.bundle/config.bak",
				MatchRegexp(fmt.Sprintf(`      Global config path: /layers/%s/launch-gems/config`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				"    Adding global config path to $BUNDLE_USER_CONFIG",
			))
			Expect(logs).To(ContainLines(
				"    Running 'bundle config --global clean true'",
			))
			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf("    Running 'bundle config --global path /layers/%s/launch-gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
			))
			Expect(logs).To(ContainLines("    Running 'bundle config --global without development:test'"))
			Expect(logs).To(ContainLines("    Running 'bundle config --global cache_path --parseable'"))
			Expect(logs).To(ContainLines("    Running 'bundle install'"))
			Expect(logs).To(ContainLines(
				MatchRegexp(`      Completed in \d+\.?\d*`),
			))
			Expect(logs).To(ContainLines(
				"  Configuring build environment",
				MatchRegexp(fmt.Sprintf(`    BUNDLE_USER_CONFIG -> "/layers/%s/build-gems/config"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
			))
			Expect(logs).To(ContainLines(
				"  Configuring launch environment",
				MatchRegexp(fmt.Sprintf(`    BUNDLE_USER_CONFIG -> "/layers/%s/launch-gems/config"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
			))
			Expect(logs).To(ContainLines(
				"  Cleaning up /workspace/.bundle/config",
				"  Cleaning up /workspace/.bundle/config.bak",
				"",
			))

			Expect(logs).To(ContainLines("Paketo Buildpack for Bundle List"))
			Expect(logs).To(ContainLines(
				"  Gems included by the bundle:",
				MatchRegexp(`    \* coderay`),
				MatchRegexp(`    \* diff-lcs`),
				MatchRegexp(`    \* method_source`),
				MatchRegexp(`    \* mustermann`),
				MatchRegexp(`    \* nio4r`),
				MatchRegexp(`    \* pry`),
				MatchRegexp(`    \* puma`),
				MatchRegexp(`    \* rack`),
				MatchRegexp(`    \* rack-protection`),
				MatchRegexp(`    \* rspec`),
				MatchRegexp(`    \* rspec-core`),
				MatchRegexp(`    \* rspec-expectations`),
				MatchRegexp(`    \* rspec-mocks`),
				MatchRegexp(`    \* rspec-support`),
				MatchRegexp(`    \* ruby2_keywords`),
				MatchRegexp(`    \* sinatra`),
				MatchRegexp(`    \* tilt`),
			))

			container, err = docker.Container.Run.
				WithCommand("bundle config && bundle list && bundle exec rackup -o 0.0.0.0").
				WithEnv(map[string]string{"PORT": "9292"}).
				WithPublish("9292").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())
			Eventually(container).Should(Serve(ContainSubstring("Hello world!")).OnPort(9292))

			logs, err = docker.Container.Logs.Execute(container.ID)
			Expect(err).NotTo(HaveOccurred())
			layerPath := fmt.Sprintf("/layers/%s", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))
			Expect(logs).To(ContainLines(
				"clean",
				`Set for the current user (`+layerPath+`/launch-gems/config): true`,
			))
			Expect(logs).To(ContainLines(
				"path",
				`Set for the current user (`+layerPath+`/launch-gems/config): "`+layerPath+`/launch-gems"`,
			))
			Expect(logs).To(ContainLines(
				"retry",
				"Set for the current user ("+layerPath+"/launch-gems/config): 5",
			))
			Expect(logs).To(ContainLines(
				"user_config",
				`Set via BUNDLE_USER_CONFIG: "`+layerPath+`/launch-gems/config"`,
			))
			Expect(logs).To(ContainLines(
				"without",
				"Set for the current user ("+layerPath+"/launch-gems/config): [:development, :test]",
			))

			Expect(logs).To(ContainLines(
				"Gems included by the bundle:",
				MatchRegexp(`  \* mustermann`),
				MatchRegexp(`  \* nio4r`),
				MatchRegexp(`  \* puma`),
				MatchRegexp(`  \* rack`),
				MatchRegexp(`  \* rack-protection`),
				MatchRegexp(`  \* ruby2_keywords`),
				MatchRegexp(`  \* sinatra`),
				MatchRegexp(`  \* tilt`),
			))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* coderay`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* diff-lcs`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* method_source`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* pry`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* rspec`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* rspec-core`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* rspec-expectations`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* rspec-mocks`)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(`\* rspec-support`)))
		})

		context("the version of bundler in the Gemfile.lock is 1.17.x", func() {
			it("creates a working OCI image", func() {
				var logs fmt.Stringer
				var err error

				source, err = occam.Source(filepath.Join("testdata", "bundler_version_1_17"))
				Expect(err).NotTo(HaveOccurred())

				// must build with MRI 3.0 or 3.1, as some Bundler 1.17 code is not compatible with Ruby 3.2
				// for example: `/usr/local/bundle/gems/bundler-1.17.3/lib/bundler/shared_helpers.rb:118: warning: Pathname#untaint is deprecated and will be removed in Ruby 3.2.`
				image, logs, err = pack.WithVerbose().Build.
					WithBuildpacks(
						settings.Buildpacks.MRI.Online,
						settings.Buildpacks.Bundler.Online,
						settings.Buildpacks.BundleInstall.Online,
						settings.Buildpacks.BundleList.Online,
					).
					WithEnv(map[string]string{"BP_MRI_VERSION": "3.1"}).
					WithPullPolicy("never").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.Buildpack.Name)),
					"  Executing build environment install process",
					"    Running 'bundle config --global clean true'",
				))
				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf("    Running 'bundle config --global path /layers/%s/build-gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				))
				Expect(logs).To(ContainLines("    Running 'bundle config --global cache_path --parseable'"))
				Expect(logs).To(ContainLines("    Running 'bundle install'"))
				Expect(logs).To(ContainLines(
					MatchRegexp(`      Completed in \d+\.?\d*`),
				))
				Expect(logs).To(ContainLines(
					"  Executing launch environment install process",
					"    Running 'bundle config --global clean true'",
				))
				Expect(logs).To(ContainLines(MatchRegexp(fmt.Sprintf("    Running 'bundle config --global path /layers/%s/launch-gems'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")))))
				Expect(logs).To(ContainLines("    Running 'bundle config --global without development:test'"))
				Expect(logs).To(ContainLines("    Running 'bundle config --global cache_path --parseable'"))
				Expect(logs).To(ContainLines("    Running 'bundle install'"))
				Expect(logs).To(ContainLines(
					MatchRegexp(`      Completed in \d+\.?\d*`),
				))
				Expect(logs).To(ContainLines(
					"  Configuring build environment",
					MatchRegexp(fmt.Sprintf(`    BUNDLE_USER_CONFIG -> "/layers/%s/build-gems/config"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				))
				Expect(logs).To(ContainLines(
					"  Configuring launch environment",
					MatchRegexp(fmt.Sprintf(`    BUNDLE_USER_CONFIG -> "/layers/%s/launch-gems/config"`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))),
				))

				Expect(logs).To(ContainLines(
					"Paketo Buildpack for Bundle List",
				))
				Expect(logs).To(ContainLines(
					"  Gems included by the bundle:",
					MatchRegexp(`    \* bundler`),
					MatchRegexp(`    \* coderay`),
					MatchRegexp(`    \* diff-lcs`),
					MatchRegexp(`    \* method_source`),
					MatchRegexp(`    \* mustermann`),
					MatchRegexp(`    \* nio4r`),
					MatchRegexp(`    \* pry`),
					MatchRegexp(`    \* puma`),
					MatchRegexp(`    \* rack`),
					MatchRegexp(`    \* rack-protection`),
					MatchRegexp(`    \* rspec`),
					MatchRegexp(`    \* rspec-core`),
					MatchRegexp(`    \* rspec-expectations`),
					MatchRegexp(`    \* rspec-mocks`),
					MatchRegexp(`    \* rspec-support`),
					MatchRegexp(`    \* ruby2_keywords`),
					MatchRegexp(`    \* sinatra`),
					MatchRegexp(`    \* tilt`),
				))

				container, err = docker.Container.Run.
					WithCommand("bundle config && bundle list && bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())
				Eventually(container).Should(Serve(ContainSubstring("Hello world!")).OnPort(9292))

				logs, err = docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				layerPath := fmt.Sprintf("/layers/%s", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"))
				Expect(logs).To(ContainLines(
					"retry",
					"Set for the current user ("+layerPath+"/launch-gems/config): 5",
				))
				Expect(logs).To(ContainLines(
					"clean",
					`Set for the current user (`+layerPath+`/launch-gems/config): "true"`,
				))
				Expect(logs).To(ContainLines(
					"path",
					`Set for the current user (`+layerPath+`/launch-gems/config): "`+layerPath+`/launch-gems"`,
				))
				Expect(logs).To(ContainLines(
					"without",
					"Set for the current user ("+layerPath+"/launch-gems/config): [:development, :test]",
				))
				Expect(logs).To(ContainLines(
					"user_config",
					`Set via BUNDLE_USER_CONFIG: "`+layerPath+`/launch-gems/config"`,
				))

				Expect(logs).To(ContainLines(
					"Gems included by the bundle:",
					MatchRegexp(`  \* bundler`),
					MatchRegexp(`  \* mustermann`),
					MatchRegexp(`  \* nio4r`),
					MatchRegexp(`  \* puma`),
					MatchRegexp(`  \* rack`),
					MatchRegexp(`  \* rack-protection`),
					MatchRegexp(`  \* ruby2_keywords`),
					MatchRegexp(`  \* sinatra`),
					MatchRegexp(`  \* tilt`),
				))
			})
		})

		context("validating SBOM", func() {
			var (
				sbomDir string
			)

			it.Before(func() {
				var err error
				sbomDir, err = os.MkdirTemp("", "sbom")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(sbomDir, os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(sbomDir)).To(Succeed())
			})

			it("writes SBOM files to the layer and label metadata", func() {
				var err error
				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.MRI.Online,
						settings.Buildpacks.Bundler.Online,
						settings.Buildpacks.BundleInstall.Online,
						settings.Buildpacks.BundleList.Online,
					).
					WithEnv(map[string]string{
						"BP_LOG_LEVEL": "DEBUG",
					}).
					WithSBOMOutputDir(sbomDir).
					Execute(name, source)
				Expect(err).ToNot(HaveOccurred(), logs.String)

				container, err = docker.Container.Run.
					WithCommand("bundle config && bundle list && bundle exec rackup -o 0.0.0.0").
					WithEnv(map[string]string{"PORT": "9292"}).
					WithPublish("9292").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())
				Eventually(container).Should(Serve(ContainSubstring("Hello world!")).OnPort(9292))

				Expect(logs).To(ContainLines(
					fmt.Sprintf("  Generating SBOM for /layers/%s/build-gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
					MatchRegexp(`      Completed in \d+(\.?\d+)*`),
				))
				Expect(logs).To(ContainLines(
					fmt.Sprintf("  Generating SBOM for /layers/%s/launch-gems", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
					MatchRegexp(`      Completed in \d+(\.?\d+)*`),
				))
				Expect(logs).To(ContainLines(
					"  Writing SBOM in the following format(s):",
					"    application/vnd.cyclonedx+json",
					"    application/spdx+json",
					"    application/vnd.syft+json",
				))

				// check that all required SBOM files are present
				Expect(filepath.Join(sbomDir, "sbom", "build", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "build-gems", "sbom.cdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "build", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "build-gems", "sbom.spdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "build", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "build-gems", "sbom.syft.json")).To(BeARegularFile())

				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "launch-gems", "sbom.cdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "launch-gems", "sbom.spdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "launch-gems", "sbom.syft.json")).To(BeARegularFile())

				// check an SBOM file to make sure it has an entry for a dependency from Gemfile.lock
				contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(settings.Buildpack.ID, "/", "_"), "launch-gems", "sbom.cdx.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(`"name": "sinatra"`))
			})
		})
	})
}
