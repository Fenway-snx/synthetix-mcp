package runtime

import (
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Process-global diagnostic context. Implementations guarantee that
// all accessors return non-nil values; the [DiagnosticContextBuilder]
// enforces this by requiring every required attribute (including
// Logger) to be set before [Build] succeeds. Callers that receive a
// DiagnosticContext may use any accessor without a nil check.
type DiagnosticContext interface {
	Logger() snx_lib_logging.Logger
}

type diagnosticContext struct {
	logger snx_lib_logging.Logger
}

var _ DiagnosticContext = (*diagnosticContext)(nil)

func (dc *diagnosticContext) Logger() snx_lib_logging.Logger {
	return dc.logger
}

// A builder for the creation of an execution context.
type DiagnosticContextBuilder struct {
	diagnosticContext // embed for now (until we cannot do so)
}

// Builds an instance of a type supporting [DiagnosticContext] from the
// current builder state. Panics if any required attribute is unset.
func (builder *DiagnosticContextBuilder) Build() DiagnosticContext {
	// precondition enforcement(s)

	if builder.logger == nil {
		panic("VIOLATION: `logger` must be set before calling Build()")
	}

	dc := new(diagnosticContext)

	*dc = builder.diagnosticContext

	return dc
}

// Resets the builder state completely.
func (builder *DiagnosticContextBuilder) Reset() {
	*builder = DiagnosticContextBuilder{}
}

func (builder *DiagnosticContextBuilder) Logger(logger snx_lib_logging.Logger) *DiagnosticContextBuilder {
	// precondition enforcement(s)

	if logger == nil {
		panic("VIOLATION: parameter `logger` may not be `nil`")
	}
	if builder.logger != nil {
		panic("VIOLATION: `logger` field already specified in builder")
	}

	builder.logger = logger

	return builder
}

// An interface that provides a `DiagnosticContext`, usually used as a
// convenience method on a process state type.
type DiagnosticContextProvider interface {
	DC() DiagnosticContext
}
