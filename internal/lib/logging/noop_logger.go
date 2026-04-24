package logger

// noOpLogger is a Logger that silently discards all output.
type noOpLogger struct{}

var _ Logger = (*noOpLogger)(nil)

// Returns a [Logger] whose methods silently discard every message. Useful
// in example binaries, integration harnesses, and anywhere a real logger is
// unnecessary but the interface must be satisfied. In tests, prefer
// [github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles.NewStubLogger],
// which allows assertions on logged output.
func NewNoOpLogger() Logger {
	return noOpLogger{}
}

func (noOpLogger) Debug(string, ...any) {}
func (noOpLogger) Info(string, ...any)  {}
func (noOpLogger) Warn(string, ...any)  {}
func (noOpLogger) Error(string, ...any) {}
func (noOpLogger) With(...any) Logger   { return noOpLogger{} }
