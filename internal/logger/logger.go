package logger

import (
	"io"
	"os"
	
	"github.com/samber/do/v2"

	"github.com/rs/zerolog"

	"github.com/codozor/fwkeeper/internal/config"
)

var Package = do.Package(
	do.Lazy(loggerProvider),
)

func loggerProvider(injector do.Injector) (zerolog.Logger, error) {
	var output io.Writer = os.Stderr

	configuration := do.MustInvoke[config.Configuration](injector)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	// Set log level
	switch configuration.Logs.Level {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		// Invalid level, fallback to info
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if configuration.Logs.Pretty {
		output = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006/01/02 15:04:05.000" }
	}

	return zerolog.New(output).With().Timestamp().Logger(), nil
}
