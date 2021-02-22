package main

import (
	"os"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
)

func main() {
	logEmitter := bundleinstall.NewLogEmitter(os.Stdout)

	packit.Run(
		bundleinstall.Detect(
			bundleinstall.NewGemfileParser(),
		),
		bundleinstall.Build(
			bundleinstall.NewBundleInstallProcess(
				pexec.NewExecutable("bundle"),
				logEmitter,
			),
			fs.NewChecksumCalculator(),
			logEmitter,
			chronos.DefaultClock,
			draft.NewPlanner(),
		),
	)
}
