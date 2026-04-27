package doubles

import (
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// A no-op stub satisfying [snx_lib_logging.Logger] for tests that
// need a logger but do not inspect its output.
type stubLogger struct{}

var _ snx_lib_logging.Logger = (*stubLogger)(nil)

func (l *stubLogger) Debug(msg string, fields ...any)            {}
func (l *stubLogger) Info(msg string, fields ...any)             {}
func (l *stubLogger) Error(msg string, fields ...any)            {}
func (l *stubLogger) Warn(msg string, fields ...any)             {}
func (l *stubLogger) With(keyVals ...any) snx_lib_logging.Logger { return l }

// Returns a no-op [snx_lib_logging.Logger] stub for tests that need a
// logger but do not inspect its output.
func NewStubLogger() snx_lib_logging.Logger {
	return &stubLogger{}
}
