package zerolog

import (
	rs_zerolog "github.com/rs/zerolog"
)

// Function that configures a logger.
type Option func(*config)

type config struct {
	OutputJSON bool
	Level      rs_zerolog.Level
}

var defaultConfig = config{
	OutputJSON: true,
	Level:      rs_zerolog.InfoLevel,
}

// Configures JSON output.
func WithOutputJSON(outputJSON bool) Option {
	return func(c *config) {
		c.OutputJSON = outputJSON
	}
}

// Configures the log level.
func WithLevel(level rs_zerolog.Level) Option {
	return func(c *config) {
		c.Level = level
	}
}

// Converts a string log level to the underlying enum.
func ParseLogLevel(level string) rs_zerolog.Level {
	switch level {
	case "debug":
		return rs_zerolog.DebugLevel
	case "info":
		return rs_zerolog.InfoLevel
	case "warn":
		return rs_zerolog.WarnLevel
	case "error":
		return rs_zerolog.ErrorLevel
	case "fatal":
		return rs_zerolog.FatalLevel
	case "panic":
		return rs_zerolog.PanicLevel
	default:
		return rs_zerolog.InfoLevel
	}
}
