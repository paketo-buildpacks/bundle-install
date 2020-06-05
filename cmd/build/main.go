package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-community/bundle-install/bundle"
)

func main() {
	executable := pexec.NewExecutable("bundle")
	logEmitter := bundle.NewLogEmitter(os.Stdout)
	installProcess := bundle.NewBundleInstallProcess(executable)

	packit.Build(bundle.Build(installProcess, logEmitter, chronos.DefaultClock))
}
