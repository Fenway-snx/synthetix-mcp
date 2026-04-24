package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_dlq_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq/doubles"
	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

// Minimal test double that satisfies DeadLetterQueueProvider by wrapping a
// stub DeadLetterQueue.
type stubDLQProvider struct {
	dlq snx_lib_dlq.DeadLetterQueue
}

var _ snx_lib_dlq.DeadLetterQueueProvider = (*stubDLQProvider)(nil)

func (p *stubDLQProvider) DLQ() snx_lib_dlq.DeadLetterQueue {
	return p.dlq
}

func newStubDLQProvider() *stubDLQProvider {
	return &stubDLQProvider{
		dlq: snx_lib_dlq_doubles.NewStubDeadLetterQueue(),
	}
}

func newStubDLQ() snx_lib_dlq.DeadLetterQueue {
	return snx_lib_dlq_doubles.NewStubDeadLetterQueue()
}

func Test_executionContext_SATISFIES_ExecutionContext(t *testing.T) {
	t.Parallel()

	var ec ExecutionContext = &executionContext{}

	assert.NotNil(t, ec)
}

func Test_executionContext_Context_RETURNS_nil_WHEN_UNSET(t *testing.T) {
	t.Parallel()

	ec := &executionContext{}

	assert.Nil(t, ec.Context())
}

func Test_ExecutionContextBuilder_ZERO_VALUE(t *testing.T) {
	t.Parallel()

	var b ExecutionContextBuilder

	assert.Nil(t, b.dlqProvider)
}

// --- #DLQ() ----------------------------------------------------------------

func Test_ExecutionContextBuilder_DLQ_SETS_PROVIDER(t *testing.T) {
	t.Parallel()

	dlq := newStubDLQ()
	var b ExecutionContextBuilder

	result := b.DLQ(dlq)

	require.Same(t, &b, result, "DLQ() should return the same builder for fluent chaining")
	require.NotNil(t, b.dlqProvider)
	assert.Same(t, dlq, b.dlqProvider.DLQ())
}

func Test_ExecutionContextBuilder_DLQ_CHAINING(t *testing.T) {
	t.Parallel()

	dlq := newStubDLQ()

	b := (&ExecutionContextBuilder{}).DLQ(dlq)

	require.NotNil(t, b)
	require.NotNil(t, b.dlqProvider)
	assert.Same(t, dlq, b.dlqProvider.DLQ())
}

func Test_ExecutionContextBuilder_DLQ_PANICS_ON_DOUBLE_SET(t *testing.T) {
	t.Parallel()

	var b ExecutionContextBuilder
	b.DLQ(newStubDLQ())

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: `dlqProvider` field already specified in builder",
		func() { b.DLQ(newStubDLQ()) },
	)
}

// --- #DLQProvider() ---------------------------------------------------------

func Test_ExecutionContextBuilder_DLQProvider_SETS_PROVIDER(t *testing.T) {
	t.Parallel()

	provider := newStubDLQProvider()
	var b ExecutionContextBuilder

	result := b.DLQProvider(provider)

	require.Same(t, &b, result, "DLQProvider() should return the same builder for fluent chaining")
	assert.Same(t, provider, b.dlqProvider)
}

func Test_ExecutionContextBuilder_DLQProvider_CHAINING(t *testing.T) {
	t.Parallel()

	provider := newStubDLQProvider()

	b := (&ExecutionContextBuilder{}).DLQProvider(provider)

	require.NotNil(t, b)
	assert.Same(t, provider, b.dlqProvider)
}

func Test_ExecutionContextBuilder_DLQProvider_PANICS_ON_DOUBLE_SET(t *testing.T) {
	t.Parallel()

	var b ExecutionContextBuilder
	b.DLQProvider(newStubDLQProvider())

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: `dlqProvider` field already specified in builder",
		func() { b.DLQProvider(newStubDLQProvider()) },
	)
}

// --- #DLQ() / #DLQProvider() cross-exclusion --------------------------------

func Test_ExecutionContextBuilder_DLQ_PANICS_AFTER_DLQProvider(t *testing.T) {
	t.Parallel()

	var b ExecutionContextBuilder
	b.DLQProvider(newStubDLQProvider())

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: `dlqProvider` field already specified in builder",
		func() { b.DLQ(newStubDLQ()) },
	)
}

func Test_ExecutionContextBuilder_DLQProvider_PANICS_AFTER_DLQ(t *testing.T) {
	t.Parallel()

	var b ExecutionContextBuilder
	b.DLQ(newStubDLQ())

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: `dlqProvider` field already specified in builder",
		func() { b.DLQProvider(newStubDLQProvider()) },
	)
}

// --- #Build() ---------------------------------------------------------------

func Test_ExecutionContextBuilder_Build_PANICS_WITHOUT_DLQ(t *testing.T) {
	t.Parallel()

	b := &ExecutionContextBuilder{}

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: cannot build an execution context without a DLQ",
		func() { b.Build() },
	)
}

func Test_ExecutionContextBuilder_Build_RETURNS_CONTEXT(t *testing.T) {
	t.Parallel()

	dlq := newStubDLQ()
	b := (&ExecutionContextBuilder{}).DLQ(dlq)

	ec := b.Build()

	require.NotNil(t, ec)
	assert.Nil(t, ec.Context())
	assert.Nil(t, ec.CancelFunc())
	assert.Same(t, dlq, ec.DLQ())
}

// --- #Reset() ---------------------------------------------------------------

func Test_ExecutionContextBuilder_Reset_CLEARS_STATE(t *testing.T) {
	t.Parallel()

	var b ExecutionContextBuilder
	b.DLQ(newStubDLQ())

	require.NotNil(t, b.dlqProvider, "precondition: provider must be set before reset")

	b.Reset()

	assert.Nil(t, b.dlqProvider)
}

func Test_ExecutionContextBuilder_Reset_IS_IDEMPOTENT(t *testing.T) {
	t.Parallel()

	var b ExecutionContextBuilder

	b.Reset()
	b.Reset()
}
