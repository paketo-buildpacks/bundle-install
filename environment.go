package bundleinstall

import (
	"fmt"
	"strconv"
	"strings"
)

type Environment struct {
	KeepGemExtensionBuildFiles bool
}

func ParseEnvironment(environ []string) (Environment, error) {
	var environment Environment
	for _, variable := range environ {
		if value, found := strings.CutPrefix(variable, "BP_KEEP_GEM_EXTENSION_BUILD_FILES="); found {
			var err error
			environment.KeepGemExtensionBuildFiles, err = strconv.ParseBool(value)
			if err != nil {
				return Environment{}, fmt.Errorf("failed to parse BP_KEEP_GEM_EXTENSION_BUILD_FILES: %w", err)
			}
		}
	}

	return environment, nil
}
