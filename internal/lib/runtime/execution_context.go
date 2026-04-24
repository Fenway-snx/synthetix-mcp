package runtime

import (
	"context"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
)

// Process-global executiion context.
type ExecutionContext interface {
	Context() context.Context
	CancelFunc() context.CancelFunc
	snx_lib_dlq.DeadLetterQueueProvider // DLQ()
}

type executionContext struct {
	ctx         context.Context
	cancel      context.CancelFunc
	dlqProvider snx_lib_dlq.DeadLetterQueueProvider
}

var _ ExecutionContext = (*executionContext)(nil)

func (ec *executionContext) Context() context.Context {
	return ec.ctx
}

func (ec *executionContext) CancelFunc() context.CancelFunc {
	return ec.cancel
}

func (ec *executionContext) DLQ() snx_lib_dlq.DeadLetterQueue {
	return ec.dlqProvider.DLQ()
}

// A builder for the creation of an execution context.
type ExecutionContextBuilder struct {
	executionContext // embed for now (until we cannot do so)
}

// Called to build an instance of a type supporting `ExecutionContext` from
// the current builder state.
//
// Note:
// This will panic if no DLQ has been set.
func (builder *ExecutionContextBuilder) Build() ExecutionContext {
	// precondition enforcement(s)

	if builder.dlqProvider == nil {
		panic("VIOLATION: cannot build an execution context without a DLQ")
	}

	ec := new(executionContext)

	*ec = builder.executionContext

	return ec
}

// Resets the builder state completely.
func (builder *ExecutionContextBuilder) Reset() {
	*builder = ExecutionContextBuilder{}
}

func (builder *ExecutionContextBuilder) ContextAndCancel(
	ctx context.Context,
	cancel context.CancelFunc,
) *ExecutionContextBuilder {
	// precondition enforcement(s)

	if builder.ctx != nil {
		panic("VIOLATION: `ctx` field already specified in builder")
	}
	if builder.cancel != nil {
		panic("VIOLATION: `cancel` field already specified in builder")
	}

	builder.ctx = ctx
	builder.cancel = cancel

	return builder
}

// Set the builder's `DLQProvider` attribute indirectly by providing a
// [DeadLetterQueue] reference.
//
// Preconditions (checked at runtime; typed nils are not detected):
//   - `dlq != nil`;
//   - neither `#DLQ()` nor `#DLQProvider()` have previously been called;
func (builder *ExecutionContextBuilder) DLQ(dlq snx_lib_dlq.DeadLetterQueue) *ExecutionContextBuilder {
	// precondition enforcement(s)

	if builder.dlqProvider != nil {
		panic("VIOLATION: `dlqProvider` field already specified in builder")
	}

	builder.dlqProvider = snx_lib_dlq.NewDeadLetterQueueProvider(dlq)

	return builder
}

// Set the builder's `DLQProvider` attribute directly.
//
// Preconditions (checked at runtime; typed nils are not detected):
//   - `dlqp != nil`;
//   - neither `#DLQ()` nor `#DLQProvider()` have previously been called;
func (builder *ExecutionContextBuilder) DLQProvider(dlqp snx_lib_dlq.DeadLetterQueueProvider) *ExecutionContextBuilder {
	// precondition enforcement(s)

	if builder.dlqProvider != nil {
		panic("VIOLATION: `dlqProvider` field already specified in builder")
	}

	builder.dlqProvider = dlqp

	return builder
}

// An interface that provides a `ExecutionContext`, usually used as a
// convenience method on a process state type.
type ExecutionContextProvider interface {
	EC() ExecutionContext
}
