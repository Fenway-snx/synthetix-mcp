package doubles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type simpleLetter struct {
	recipient string
	body      string
}

// --- NewSpyDeadLetterQueue --------------------------------------------------

func Test_NewSpyDeadLetterQueue_RETURNS_EMPTY_INSTANCE(t *testing.T) {
	t.Parallel()

	dlq, err := NewSpyDeadLetterQueue()

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, dlq)
	assert.Empty(t, dlq.Entries)
}

// --- SpyDeadLetterQueue#Post ------------------------------------------------

func Test_SpyDeadLetterQueue_Post_SINGLE_ENTRY(t *testing.T) {
	t.Parallel()

	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	letter := simpleLetter{recipient: "Bart", body: "Cowabunga!"}

	err = dlq.Post(letter, Envelope{
		System:    "DLQTest",
		Subsystem: "testing",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, dlq.Entries, 1)

	entry := dlq.Entries[0]

	assert.Equal(t, letter, entry.Letter)
	assert.Equal(t, "", entry.Envelope.Application)
	assert.Equal(t, "DLQTest", entry.Envelope.System)
	assert.Equal(t, "testing", entry.Envelope.Subsystem)
}

func Test_SpyDeadLetterQueue_Post_MULTIPLE_ENTRIES_PRESERVE_ORDER(t *testing.T) {
	t.Parallel()

	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	letters := []simpleLetter{
		{recipient: "Bart", body: "Cowabunga!"},
		{recipient: "Lisa", body: "If anyone wants me, I'll be in my room."},
		{recipient: "Homer", body: "D'oh!"},
	}

	for _, l := range letters {
		err = dlq.Post(l, Envelope{})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	}

	require.Len(t, dlq.Entries, 3)

	for i, l := range letters {
		assert.Equal(t, l, dlq.Entries[i].Letter, "letter mismatch at index %d", i)
	}
}

func Test_SpyDeadLetterQueue_Post_EMPTY_ENVELOPE(t *testing.T) {
	t.Parallel()

	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("help!", Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, dlq.Entries, 1)

	entry := dlq.Entries[0]

	assert.Equal(t, "help!", entry.Letter)
	assert.Equal(t, "", entry.Envelope.Application)
	assert.Equal(t, "", entry.Envelope.System)
	assert.Equal(t, "", entry.Envelope.Subsystem)
}

func Test_SpyDeadLetterQueue_Post_CAPTURES_ENVELOPE_VERBATIM(t *testing.T) {
	t.Parallel()

	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("test", Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	entry := dlq.Entries[0]

	assert.True(t, entry.Envelope.PostTime.IsZero(), "PostTime should not be populated by spy")
	assert.Empty(t, entry.Envelope.HostName(), "HostName should not be populated by spy")
	assert.Zero(t, entry.Envelope.PId(), "PId should not be populated by spy")
	assert.Empty(t, entry.Envelope.ProcessName(), "ProcessName should not be populated by spy")
}

func Test_SpyDeadLetterQueue_Post_nil_LETTER(t *testing.T) {
	t.Parallel()

	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post(nil, Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, dlq.Entries, 1)
	assert.Nil(t, dlq.Entries[0].Letter)
}

// --- SpyDeadLetterQueue#WithOnPost ------------------------------------------

func Test_SpyDeadLetterQueue_WithOnPost_CALLBACK_CONTROLS_ERROR(t *testing.T) {
	t.Parallel()

	sentinel := assert.AnError
	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq.WithOnPost(func(_ any, _ Envelope) error { return sentinel })

	err = dlq.Post("letter", Envelope{})

	assert.ErrorIs(t, err, sentinel)
	require.Len(t, dlq.Entries, 1, "entry should still be recorded despite error")
}

func Test_SpyDeadLetterQueue_WithOnPost_CALLBACK_RECEIVES_ARGUMENTS(t *testing.T) {
	t.Parallel()

	var gotLetter any
	var gotEnvelope Envelope
	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	dlq.WithOnPost(func(letter any, envelope Envelope) error {
		gotLetter = letter
		gotEnvelope = envelope
		return nil
	})

	env := Envelope{Application: "test-app"}
	err = dlq.Post("hello", env)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, "hello", gotLetter)
	assert.Equal(t, "test-app", gotEnvelope.Application)
}

func Test_SpyDeadLetterQueue_WithOnPost_RETURNS_RECEIVER_FOR_CHAINING(t *testing.T) {
	t.Parallel()

	dlq, err := NewSpyDeadLetterQueue()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	result := dlq.WithOnPost(func(_ any, _ Envelope) error { return nil })

	assert.Same(t, dlq, result)
}
