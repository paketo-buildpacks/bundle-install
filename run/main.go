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
	gemfileParser := bundleinstall.NewGemfileParser()
	executable := pexec.NewExecutable("bundle")
	logEmitter := bundleinstall.NewLogEmitter(os.Stdout)
	installProcess := bundleinstall.NewBundleInstallProcess(executable)
	calculator := fs.NewChecksumCalculator()

	packit.Run(
		bundleinstall.Detect(gemfileParser),
		bundleinstall.Build(
			installProcess,
			calculator,
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
