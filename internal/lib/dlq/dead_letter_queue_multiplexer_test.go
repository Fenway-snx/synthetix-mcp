package dlq_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_dlq_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq/doubles"
)

// --- NewDeadLetterQueueMultiplexer construction ------------------------------

func Test_NewDeadLetterQueueMultiplexer_RETURNS_NON_nil(t *testing.T) {
	t.Parallel()

	mux := snx_lib_dlq.NewDeadLetterQueueMultiplexer(nil)

	assert.NotNil(t, mux)
}

// --- Post: fan-out -----------------------------------------------------------

func Test_NewDeadLetterQueueMultiplexer_Post_NO_IMPLEMENTORS_RETURNS_nil(t *testing.T) {
	t.Parallel()

	mux := snx_lib_dlq.NewDeadLetterQueueMultiplexer(nil)

	err := mux.Post("letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_NewDeadLetterQueueMultiplexer_Post_FORWARDS_TO_SINGLE_IMPLEMENTOR(t *testing.T) {
	t.Parallel()

	spy, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	mux := snx_lib_dlq.NewDeadLetterQueueMultiplexer([]snx_lib_dlq.DeadLetterQueue{spy})

	letter := "payload"
	env := snx_lib_dlq.Envelope{Application: "app"}

	err = mux.Post(letter, env)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	assert.Equal(t, letter, spy.Entries[0].Letter)
	assert.Equal(t, env, spy.Entries[0].Envelope)
}

func Test_NewDeadLetterQueueMultiplexer_Post_FORWARDS_TO_ALL_IMPLEMENTORS(t *testing.T) {
	t.Parallel()

	spyA, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	spyB, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	mux := snx_lib_dlq.NewDeadLetterQueueMultiplexer([]snx_lib_dlq.DeadLetterQueue{spyA, spyB})

	letter := struct{ ID int }{ID: 42}
	env := snx_lib_dlq.Envelope{Subsystem: "sub"}

	err = mux.Post(letter, env)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spyA.Entries, 1)
	require.Len(t, spyB.Entries, 1)
	assert.Equal(t, letter, spyA.Entries[0].Letter)
	assert.Equal(t, env, spyA.Entries[0].Envelope)
	assert.Equal(t, letter, spyB.Entries[0].Letter)
	assert.Equal(t, env, spyB.Entries[0].Envelope)
}

func Test_NewDeadLetterQueueMultiplexer_Post_CALLS_IMPLEMENTORS_IN_SLICE_ORDER(t *testing.T) {
	t.Parallel()

	spyA, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	spyB, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	callOrder := make([]int, 0, 2)
	spyA.WithOnPost(func(letter any, envelope snx_lib_dlq.Envelope) error {
		callOrder = append(callOrder, 1)

		return nil
	})
	spyB.WithOnPost(func(letter any, envelope snx_lib_dlq.Envelope) error {
		callOrder = append(callOrder, 2)

		return nil
	})

	mux := snx_lib_dlq.NewDeadLetterQueueMultiplexer([]snx_lib_dlq.DeadLetterQueue{spyA, spyB})

	err = mux.Post(nil, snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, []int{1, 2}, callOrder)
}

func Test_NewDeadLetterQueueMultiplexer_Post_RETURNS_nil_WHEN_IMPLEMENTOR_POST_ERR(t *testing.T) {
	t.Parallel()

	spy, err := snx_lib_dlq_doubles.NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	spy.WithOnPost(func(letter any, envelope snx_lib_dlq.Envelope) error {
		return errors.New("implementor post failed")
	})

	mux := snx_lib_dlq.NewDeadLetterQueueMultiplexer([]snx_lib_dlq.DeadLetterQueue{spy})

	err = mux.Post("x", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
}
