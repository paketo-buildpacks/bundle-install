package bundleinstall

import (
	"io"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

// LogEmitter prints formatted log output using the facilities of a
// scribe.Logger.
type LogEmitter struct {
	scribe.Logger
}

// NewLogEmitter initializes an instance of a LogEmitter.
func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Logger: scribe.NewLogger(output),
	}
}

// Environment prints the environment variable settings for the given layer.
func (l LogEmitter) Environment(layer packit.Layer) {
	if len(layer.BuildEnv) > 0 {
		l.Process("Configuring build environment")
		l.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(layer.BuildEnv))
		l.Break()
	}

	if len(layer.LaunchEnv) > 0 {
		l.Process("Configuring launch environment")
		l.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(layer.LaunchEnv))
		l.Break()
	}
}
