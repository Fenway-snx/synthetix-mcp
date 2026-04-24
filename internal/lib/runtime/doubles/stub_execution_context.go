package doubles

import (
	"context"

	snx_lib_dlq_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq/doubles"
	snx_lib_runtime "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime"
)

// Creates a [snx_lib_runtime.ExecutionContext] for tests. If ctx is nil,
// [context.Background] is used. A stub DLQ is wired automatically.
//
// The returned EC holds a real cancel function from [context.WithCancel].
// Callers may defer ec.CancelFunc()() for explicit cleanup, but this is
// not required for correctness when the parent is a standard library
// context (e.g. context.Background or testing.T.Context), as the GC will
// collect the child context.
func NewStubExecutionContext(
	ctx context.Context,
) snx_lib_runtime.ExecutionContext {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	return new(snx_lib_runtime.ExecutionContextBuilder).
		ContextAndCancel(ctx, cancel).
		DLQ(snx_lib_dlq_doubles.NewStubDeadLetterQueue()).
		Build()
}
