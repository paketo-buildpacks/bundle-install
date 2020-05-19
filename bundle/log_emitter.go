package bundle

import (
	"io"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

type LogEmitter struct {
	// Logger is embedded and therefore delegates all of its functions to the
	// LogEmitter.
	scribe.Logger
}

func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Logger: scribe.NewLogger(output),
	}
}

func (l LogEmitter) Environment(env packit.Environment) {
	l.Process("Configuring environment")
	l.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(env))
	l.Break()
}
