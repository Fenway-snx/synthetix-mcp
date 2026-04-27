package zerolog

import (
	rs_zerolog "github.com/rs/zerolog"
)

// Option is a function that configures a logger.
type Option func(*config)

type config struct {
	OutputJSON bool
	Level      rs_zerolog.Level
}

var defaultConfig = config{
	OutputJSON: true,
	Level:      rs_zerolog.InfoLevel,
}

// WithOutputJSON configures the logger to output JSON format.
func WithOutputJSON(outputJSON bool) Option {
	return func(c *config) {
		c.OutputJSON = outputJSON
	}
}

// WithLevel configures the log level for the logger.
func WithLevel(level rs_zerolog.Level) Option {
	return func(c *config) {
		c.Level = level
	}
}

// ParseLogLevel converts a string log level to zerolog.Level.
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
