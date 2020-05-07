package main

import (
	"github.com/cloudfoundry/packit"
	"github.com/paketo-community/bundle-install/bundle"
)

func main() {
	gemfileParser := bundle.NewGemfileParser()

	packit.Detect(bundle.Detect(gemfileParser))
}
