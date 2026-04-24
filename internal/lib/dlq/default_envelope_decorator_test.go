package dlq_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_dlq_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq/doubles"
)

// --- NewDefaultEnvelopeDecorator construction -------------------------------

func Test_NewDefaultEnvelopeDecorator_RETURNS_NON_nil(t *testing.T) {
	t.Parallel()

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{})

	assert.NotNil(t, dlq)
}

// --- Post: delegation -------------------------------------------------------

func Test_NewDefaultEnvelopeDecorator_Post_DELEGATES_TO_INNER_DLQ(t *testing.T) {
	t.Parallel()

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{})

	err = dlq.Post("hello", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, inner.Entries, 1)
}

func Test_NewDefaultEnvelopeDecorator_Post_PASSES_LETTER_THROUGH(t *testing.T) {
	t.Parallel()

	type testLetter struct {
		OrderID string
		Reason  string
	}

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{})
	letter := testLetter{OrderID: "ORD-42", Reason: "margin"}

	err = dlq.Post(letter, snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, inner.Entries, 1)
	assert.Equal(t, letter, inner.Entries[0].Letter)
}

func Test_NewDefaultEnvelopeDecorator_Post_MULTIPLE_POSTS_DELEGATE(t *testing.T) {
	t.Parallel()

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{})

	require.NoError(t, dlq.Post("first", snx_lib_dlq.Envelope{}))
	require.NoError(t, dlq.Post("second", snx_lib_dlq.Envelope{}))
	require.NoError(t, dlq.Post("third", snx_lib_dlq.Envelope{}))

	require.Len(t, inner.Entries, 3)
	assert.Equal(t, "first", inner.Entries[0].Letter)
	assert.Equal(t, "second", inner.Entries[1].Letter)
	assert.Equal(t, "third", inner.Entries[2].Letter)
}

// --- Post: merge semantics --------------------------------------------------

func Test_NewDefaultEnvelopeDecorator_Post_MERGES_DEFAULT_INTO_EMPTY_ENVELOPE(t *testing.T) {
	t.Parallel()

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{
		Application: "default-app",
		System:      "default-sys",
		Subsystem:   "default-sub",
	})

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, inner.Entries, 1)
	env := inner.Entries[0].Envelope
	assert.Equal(t, "default-app", env.Application)
	assert.Equal(t, "default-sys", env.System)
	assert.Equal(t, "default-sub", env.Subsystem)
}

func Test_NewDefaultEnvelopeDecorator_Post_DEFAULT_OVERRIDES_CALLER(t *testing.T) {
	t.Parallel()

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{
		Application: "default-app",
		System:      "default-sys",
	})

	err = dlq.Post("letter", snx_lib_dlq.Envelope{
		Application: "caller-app",
		System:      "caller-sys",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, inner.Entries, 1)
	env := inner.Entries[0].Envelope
	assert.Equal(t, "default-app", env.Application, "default overrides caller")
	assert.Equal(t, "default-sys", env.System, "default overrides caller")
}

func Test_NewDefaultEnvelopeDecorator_Post_CALLER_PRESERVED_WHEN_DEFAULT_EMPTY(t *testing.T) {
	t.Parallel()

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{
		Application: "default-app",
	})

	err = dlq.Post("letter", snx_lib_dlq.Envelope{
		System:    "caller-sys",
		Subsystem: "caller-sub",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, inner.Entries, 1)
	env := inner.Entries[0].Envelope
	assert.Equal(t, "default-app", env.Application, "default fills its field")
	assert.Equal(t, "caller-sys", env.System, "caller preserved when default empty")
	assert.Equal(t, "caller-sub", env.Subsystem, "caller preserved when default empty")
}

// --- Post: affix behaviour --------------------------------------------------
//
// The decorator stamps only file_line_function at its own call depth;
// all other diagnostic fields (hostname, PID, byte order, etc.) are
// the responsibility of the inner DeadLetterQueue (in production, a
// dlqHandler that uses its cached invariants).

func Test_NewDefaultEnvelopeDecorator_Post_AFFIXES_FILE_LINE_FUNCTION(t *testing.T) {
	t.Parallel()

	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{})

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, inner.Entries, 1)
	env := inner.Entries[0].Envelope

	assert.NotEmpty(t, env.FileLineFunction(), "file/line/function should be affixed by the decorator")
}

// --- Post: error propagation ------------------------------------------------

func Test_NewDefaultEnvelopeDecorator_Post_PROPAGATES_INNER_DLQ_ERROR(t *testing.T) {
	t.Parallel()

	innerErr := errors.New("inner DLQ failed")
	inner, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	inner.WithOnPost(func(_ any, _ snx_lib_dlq.Envelope) error {
		return innerErr
	})

	dlq := snx_lib_dlq.NewDefaultEnvelopeDecorator(inner, snx_lib_dlq.Envelope{
		Application: "test-app",
	})

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})

	assert.ErrorIs(t, err, innerErr)
	require.Len(t, inner.Entries, 1, "entry should still be recorded despite error")
}
