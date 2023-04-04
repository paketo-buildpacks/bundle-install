package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/occam"
	"github.com/pelletier/go-toml"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

var settings struct {
	Buildpacks struct {
		BundleInstall struct {
			Online  string
			Offline string
		}
		Bundler struct {
			Online  string
			Offline string
		}
		MRI struct {
			Online  string
			Offline string
		}
		BundleList struct {
			Online string
		}
	}

	Buildpack struct {
		ID   string
		Name string
	}

	Config struct {
		Bundler string `json:"bundler"`
		MRI     string `json:"mri"`
	}
}

var builder occam.Builder

func TestIntegration(t *testing.T) {
	// Do not truncate Gomega matcher output
	// The buildpack output text can be large and we often want to see all of it.
	format.MaxLength = 0

	Expect := NewWithT(t).Expect
	pack := occam.NewPack()

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())
	Expect(file.Close()).To(Succeed())

	root, err := filepath.Abs("./..")
	Expect(err).NotTo(HaveOccurred())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	err = toml.NewDecoder(file).Decode(&settings)
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())

	buildpackStore := occam.NewBuildpackStore()

	settings.Buildpacks.BundleInstall.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.BundleInstall.Offline, err = buildpackStore.Get.
		WithOfflineDependencies().
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.Bundler.Online, err = buildpackStore.Get.
		Execute(settings.Config.Bundler)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.Bundler.Offline, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(settings.Config.Bundler)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.MRI.Online, err = buildpackStore.Get.
		Execute(settings.Config.MRI)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.MRI.Offline, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(settings.Config.MRI)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.BundleList.Online = filepath.Join(root, "integration", "testdata", "bundle-list-buildpack")

	SetDefaultEventuallyTimeout(30 * time.Second)

	builder, err = pack.Builder.Inspect.Execute()
	Expect(err).NotTo(HaveOccurred())

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	// This test will only run on the Bionic full stack, because stack upgrade
	// failures have only been observed when upgrading from the Bionic full stack.
	// All other tests will run against the Bionic base stack
	if builder.BuilderName == "paketobuildpacks/builder:buildpackless-full" {
		suite("StackUpgrade", testStackUpgrade)
	}

	suite("LayerReuse", testLayerReuse)
	suite("OfflineApp", testOffline)
	suite("ReproducibleBuilds", testReproducibleBuilds)
	suite("SimpleApp", testSimpleApp)

	suite.Run(t)
}
