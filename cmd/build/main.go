package main

import (
	"os"
	"time"

	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/paketo-community/bundle-install/bundle"
)

func main() {
	executable := pexec.NewExecutable("bundle")
	logEmitter := bundle.NewLogEmitter(os.Stdout)
	clock := bundle.NewClock(time.Now)
	installProcess := bundle.NewBundleInstallProcess(executable)

	packit.Build(bundle.Build(installProcess, logEmitter, clock))
}
