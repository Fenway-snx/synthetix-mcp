package logger

// Logger is the Synthetix logger interface.
type Logger interface {
	// Debug takes a message and a set of key/value pairs and logs with level DEBUG.
	Debug(msg string, keyVals ...any)

	// Info takes a message and a set of key/value pairs and logs with level INFO.
	Info(msg string, keyVals ...any)

	// Warn takes a message and a set of key/value pairs and logs with level WARN.
	Warn(msg string, keyVals ...any)

	// Error takes a message and a set of key/value pairs and logs with level ERR.
	Error(msg string, keyVals ...any)

	// With returns a new wrapped logger with additional context provided by a set.
	With(keyVals ...any) Logger
}
