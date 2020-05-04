package main

import (
	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/paketo-community/bundle-install/bundle"
)

func main() {
	executable := pexec.NewExecutable("bundle")
	installProcess := bundle.NewBundleInstallProcess(executable)

	packit.Build(bundle.Build(installProcess))
}
