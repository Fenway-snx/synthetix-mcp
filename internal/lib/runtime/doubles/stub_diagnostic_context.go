package doubles

import (
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_runtime "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime"
)

// Creates a [snx_lib_runtime.DiagnosticContext] for tests. If logger
// is nil, a [snx_lib_logging_doubles.NewStubLogger] is used.
func NewStubDiagnosticContext(
	logger snx_lib_logging.Logger,
) snx_lib_runtime.DiagnosticContext {
	if logger == nil {
		logger = snx_lib_logging_doubles.NewStubLogger()
	}
	return new(snx_lib_runtime.DiagnosticContextBuilder).
		Logger(logger).
		Build()
}
