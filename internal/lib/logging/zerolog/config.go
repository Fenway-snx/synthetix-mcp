package zerolog

import (
	"io"

	rs_zerolog "github.com/rs/zerolog"
)

// Option is a function that configures a logger.
type Option func(*config)

type config struct {
	OutputJSON bool
	Color      bool
	TimeFormat string
	Level      rs_zerolog.Level
	StackTrace bool
	Filter     func(level rs_zerolog.Level) bool
	Hooks      []rs_zerolog.Hook
}

var defaultConfig = config{
	OutputJSON: true,
	Color:      true,
	TimeFormat: "2006-01-02 15:04:05",
	Level:      rs_zerolog.InfoLevel,
	StackTrace: false,
	Filter:     nil,
	Hooks:      nil,
}

// WithOutputJSON configures the logger to output JSON format.
func WithOutputJSON(outputJSON bool) Option {
	return func(c *config) {
		c.OutputJSON = outputJSON
	}
}

// WithColor configures the logger to use color output.
func WithColor(color bool) Option {
	return func(c *config) {
		c.Color = color
	}
}

// WithTimeFormat configures the time format for the logger.
func WithTimeFormat(format string) Option {
	return func(c *config) {
		c.TimeFormat = format
	}
}

// WithLevel configures the log level for the logger.
func WithLevel(level rs_zerolog.Level) Option {
	return func(c *config) {
		c.Level = level
	}
}

// WithStackTrace configures the logger to include stack traces.
func WithStackTrace(stackTrace bool) Option {
	return func(c *config) {
		c.StackTrace = stackTrace
	}
}

// WithFilter configures a filter function for the logger.
func WithFilter(filter func(level rs_zerolog.Level) bool) Option {
	return func(c *config) {
		c.Filter = filter
	}
}

// WithHooks configures hooks for the logger.
func WithHooks(hooks ...rs_zerolog.Hook) Option {
	return func(c *config) {
		c.Hooks = hooks
	}
}

// FilterWriter is a writer that filters log entries based on a
// filter function.
type FilterWriter struct {
	writer io.Writer
	filter func(level rs_zerolog.Level) bool
}

// NewFilterWriter creates a new FilterWriter.
func NewFilterWriter(writer io.Writer, filter func(level rs_zerolog.Level) bool) *FilterWriter {
	return &FilterWriter{
		writer: writer,
		filter: filter,
	}
}

// Write implements io.Writer.
func (w *FilterWriter) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

// ParseLogLevel converts a string log level to zerolog.Level.
//
// TODO: we need to define our own log levels and map them as appropriate to
// a given [Logger] implementation.
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
