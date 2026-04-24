package doubles

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewSpyDeliverer --------------------------------------------------------

func Test_NewSpyDeliverer_RETURNS_EMPTY_INSTANCE(t *testing.T) {
	t.Parallel()

	sd := NewSpyDeliverer()

	require.NotNil(t, sd)
	assert.Empty(t, sd.Entries)
}

// --- SpyDeliverer#OnPost ----------------------------------------------------

func Test_SpyDeliverer_OnPost_SINGLE_ENTRY(t *testing.T) {
	t.Parallel()

	sd := NewSpyDeliverer()

	envelope := Envelope{
		Application: "test-app",
		System:      "test-system",
	}
	jsonStr := `{"application":"test-app","system":"test-system"}`

	err := sd.OnPost(envelope, jsonStr)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, sd.Entries, 1)

	entry := sd.Entries[0]

	assert.Equal(t, envelope, entry.Envelope)
	assert.Equal(t, jsonStr, entry.EnvelopeJSONString)
}

func Test_SpyDeliverer_OnPost_MULTIPLE_ENTRIES_PRESERVE_ORDER(t *testing.T) {
	t.Parallel()

	sd := NewSpyDeliverer()

	jsonStrings := []string{
		`{"n":1}`,
		`{"n":2}`,
		`{"n":3}`,
	}

	for _, js := range jsonStrings {
		err := sd.OnPost(Envelope{}, js)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	}

	require.Len(t, sd.Entries, 3)

	for i, js := range jsonStrings {
		assert.Equal(t, js, sd.Entries[i].EnvelopeJSONString,
			"EnvelopeJSONString mismatch at index %d", i,
		)
	}
}

func Test_SpyDeliverer_OnPost_DEFAULT_RETURNS_nil(t *testing.T) {
	t.Parallel()

	sd := NewSpyDeliverer()

	err := sd.OnPost(Envelope{}, `{}`)

	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_SpyDeliverer_OnPost_EMPTY_ENVELOPE_AND_JSON(t *testing.T) {
	t.Parallel()

	sd := NewSpyDeliverer()

	err := sd.OnPost(Envelope{}, "")

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, sd.Entries, 1)
	assert.Equal(t, Envelope{}, sd.Entries[0].Envelope)
	assert.Equal(t, "", sd.Entries[0].EnvelopeJSONString)
}

// --- SpyDeliverer#WithOnPost ------------------------------------------------

func Test_SpyDeliverer_WithOnPost_CALLBACK_CONTROLS_ERROR(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("injected failure")

	sd := NewSpyDeliverer().WithOnPost(
		func(Envelope, string) error { return sentinel },
	)

	err := sd.OnPost(Envelope{}, `{}`)

	assert.ErrorIs(t, err, sentinel)
	require.Len(t, sd.Entries, 1, "entry must be recorded even when callback returns an error")
}

func Test_SpyDeliverer_WithOnPost_CALLBACK_RECEIVES_ARGUMENTS(t *testing.T) {
	t.Parallel()

	var capturedEnvelope Envelope
	var capturedJSON string

	sd := NewSpyDeliverer().WithOnPost(
		func(env Envelope, js string) error {
			capturedEnvelope = env
			capturedJSON = js
			return nil
		},
	)

	envelope := Envelope{Application: "capture-test"}
	jsonStr := `{"application":"capture-test"}`

	err := sd.OnPost(envelope, jsonStr)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, envelope, capturedEnvelope)
	assert.Equal(t, jsonStr, capturedJSON)
}

func Test_SpyDeliverer_WithOnPost_RETURNS_RECEIVER_FOR_CHAINING(t *testing.T) {
	t.Parallel()

	sd := NewSpyDeliverer()

	result := sd.WithOnPost(func(Envelope, string) error { return nil })

	assert.Same(t, sd, result)
}
