package bundle

import (
	"fmt"

	"github.com/cloudfoundry/packit"
)

type BuildPlanMetadata struct {
	Build  bool `toml:"build"`
	Launch bool `toml:"launch"`
}

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		return packit.DetectResult{}, fmt.Errorf("always fail")
	}
}
