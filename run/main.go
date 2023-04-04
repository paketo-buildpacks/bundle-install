package main

import (
	"fmt"
	"log"
	"os"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type Generator struct{}

func (f Generator) Generate(dir string) (sbom.SBOM, error) {
	return sbom.Generate(dir)
}

func main() {
	logEmitter := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	environment, err := bundleinstall.ParseEnvironment(os.Environ())
	if err != nil {
		log.Fatal(fmt.Errorf("failed to parse environment configuration: %w", err))
	}

	packit.Run(
		bundleinstall.Detect(
			bundleinstall.NewGemfileParser(),
		),
		bundleinstall.Build(
			draft.NewPlanner(),
			bundleinstall.NewBundleInstallProcess(
				pexec.NewExecutable("bundle"),
				logEmitter,
				bundleinstall.NewRubyVersionResolver(
					pexec.NewExecutable("ruby"),
				),
				fs.NewChecksumCalculator(),
			),
			Generator{},
			logEmitter,
			chronos.DefaultClock,
			environment,
		),
	)
}
