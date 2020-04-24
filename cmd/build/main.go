package main

import (
	"github.com/cloudfoundry/packit"
	"github.com/paketo-community/bundle-install/bundle"
)

func main() {
	packit.Build(bundle.Build())
}
