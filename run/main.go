package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	bundleinstall "github.com/paketo-community/bundle-install"
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
		),
	)
}
