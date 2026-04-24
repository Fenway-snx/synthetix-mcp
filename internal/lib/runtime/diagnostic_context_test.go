package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_diagnosticContext_SATISFIES_DiagnosticContext(t *testing.T) {
	t.Parallel()

	var dc DiagnosticContext = &diagnosticContext{}

	assert.NotNil(t, dc)
}

func Test_DiagnosticContextBuilder_ZERO_VALUE(t *testing.T) {
	t.Parallel()

	var b DiagnosticContextBuilder

	assert.Equal(t, DiagnosticContextBuilder{}, b)
}

func Test_DiagnosticContextBuilder_Build_PANICS_WITHOUT_LOGGER(t *testing.T) {
	t.Parallel()

	var b DiagnosticContextBuilder

	assert.PanicsWithValue(t,
		"VIOLATION: `logger` must be set before calling Build()",
		func() { b.Build() },
	)
}

func Test_DiagnosticContextBuilder_Logger_PANICS_ON_NIL(t *testing.T) {
	t.Parallel()

	var b DiagnosticContextBuilder

	assert.PanicsWithValue(t,
		"VIOLATION: parameter `logger` may not be `nil`",
		func() { b.Logger(nil) },
	)
}

func Test_DiagnosticContextBuilder_Reset_CLEARS_STATE(t *testing.T) {
	t.Parallel()

	var b DiagnosticContextBuilder

	b.Reset()

	assert.Equal(t, DiagnosticContextBuilder{}, b)
}

func Test_DiagnosticContextBuilder_Reset_IS_IDEMPOTENT(t *testing.T) {
	t.Parallel()

	var b DiagnosticContextBuilder

	b.Reset()
	b.Reset()

	assert.Equal(t, DiagnosticContextBuilder{}, b)
}
