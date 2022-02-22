package main

import (
	"os"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func main() {
	logEmitter := scribe.NewEmitter(os.Stdout)

	packit.Run(
		bundleinstall.Detect(
			bundleinstall.NewGemfileParser(),
		),
		bundleinstall.Build(
			bundleinstall.NewBundleInstallProcess(
				pexec.NewExecutable("bundle"),
				logEmitter,
				bundleinstall.NewRubyVersionResolver(
					pexec.NewExecutable("ruby"),
				),
				fs.NewChecksumCalculator(),
			),
			logEmitter,
			chronos.DefaultClock,
			draft.NewPlanner(),
		),
	)
}
