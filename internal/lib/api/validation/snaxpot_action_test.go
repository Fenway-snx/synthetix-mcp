package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateSaveSnaxpotTicketsAction(t *testing.T) {
	t.Run("accepts valid entries", func(t *testing.T) {
		err := ValidateSaveSnaxpotTicketsAction(&SaveSnaxpotTicketsActionPayload{
			Action: "saveSnaxpotTickets",
			Entries: []SnaxpotTicketMutationEntryPayload{
				{
					Ball1:        1,
					Ball2:        2,
					Ball3:        3,
					Ball4:        4,
					Ball5:        5,
					SnaxBall:     1,
					TicketSerial: "42",
				},
			},
		})

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("rejects invalid standard balls", func(t *testing.T) {
		err := ValidateSaveSnaxpotTicketsAction(&SaveSnaxpotTicketsActionPayload{
			Action: "saveSnaxpotTickets",
			Entries: []SnaxpotTicketMutationEntryPayload{
				{
					Ball1:        1,
					Ball2:        2,
					Ball3:        2,
					Ball4:        4,
					Ball5:        5,
					SnaxBall:     1,
					TicketSerial: "42",
				},
			},
		})

		require.Error(t, err)
		assert.Equal(t, "entries[0]: balls must use five unique standard balls in ascending order", err.Error())
	})

	t.Run("rejects invalid snax ball", func(t *testing.T) {
		err := ValidateSaveSnaxpotTicketsAction(&SaveSnaxpotTicketsActionPayload{
			Action: "saveSnaxpotTickets",
			Entries: []SnaxpotTicketMutationEntryPayload{
				{
					Ball1:        1,
					Ball2:        2,
					Ball3:        3,
					Ball4:        4,
					Ball5:        5,
					SnaxBall:     0,
					TicketSerial: "42",
				},
			},
		})

		require.Error(t, err)
		assert.Equal(t, "entries[0]: snaxBall must be between 1 and 5", err.Error())
	})

	t.Run("rejects invalid ticket serial", func(t *testing.T) {
		err := ValidateSaveSnaxpotTicketsAction(&SaveSnaxpotTicketsActionPayload{
			Action: "saveSnaxpotTickets",
			Entries: []SnaxpotTicketMutationEntryPayload{
				{
					Ball1:        1,
					Ball2:        2,
					Ball3:        3,
					Ball4:        4,
					Ball5:        5,
					SnaxBall:     1,
					TicketSerial: "not-a-number",
				},
			},
		})

		require.Error(t, err)
		assert.Equal(t, "entries[0]: ticketSerial must be a valid number", err.Error())
	})
}

func Test_ValidateSetSnaxpotPreferenceAction(t *testing.T) {
	t.Run("accepts valid persistent snax ball", func(t *testing.T) {
		err := ValidateSetSnaxpotPreferenceAction(&SetSnaxpotPreferenceActionPayload{
			Action:   "setSnaxpotPreference",
			Scope:    "persistent",
			SnaxBall: 5,
		})

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("accepts valid current-epoch snax ball", func(t *testing.T) {
		err := ValidateSetSnaxpotPreferenceAction(&SetSnaxpotPreferenceActionPayload{
			Action:   "setSnaxpotPreference",
			Scope:    "currentEpoch",
			SnaxBall: 1,
		})

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("rejects zero snax ball regardless of scope", func(t *testing.T) {
		err := ValidateSetSnaxpotPreferenceAction(&SetSnaxpotPreferenceActionPayload{
			Action:   "setSnaxpotPreference",
			Scope:    "persistent",
			SnaxBall: 0,
		})

		require.Error(t, err)
		assert.Equal(t, "snaxBall must be between 1 and 5", err.Error())
	})

	t.Run("rejects invalid scope", func(t *testing.T) {
		err := ValidateSetSnaxpotPreferenceAction(&SetSnaxpotPreferenceActionPayload{
			Action:   "setSnaxpotPreference",
			Scope:    "always",
			SnaxBall: 3,
		})

		require.Error(t, err)
		assert.Equal(t, "scope must be 'currentEpoch' or 'persistent'", err.Error())
	})
}

func Test_ValidateClearSnaxpotPreferenceAction(t *testing.T) {
	t.Run("accepts persistent scope", func(t *testing.T) {
		err := ValidateClearSnaxpotPreferenceAction(&ClearSnaxpotPreferenceActionPayload{
			Action: "clearSnaxpotPreference",
			Scope:  "persistent",
		})

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("accepts currentEpoch scope", func(t *testing.T) {
		err := ValidateClearSnaxpotPreferenceAction(&ClearSnaxpotPreferenceActionPayload{
			Action: "clearSnaxpotPreference",
			Scope:  "currentEpoch",
		})

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("rejects invalid scope", func(t *testing.T) {
		err := ValidateClearSnaxpotPreferenceAction(&ClearSnaxpotPreferenceActionPayload{
			Action: "clearSnaxpotPreference",
			Scope:  "",
		})

		require.Error(t, err)
		assert.Equal(t, "scope must be 'currentEpoch' or 'persistent'", err.Error())
	})

	t.Run("rejects wrong action type", func(t *testing.T) {
		err := ValidateClearSnaxpotPreferenceAction(&ClearSnaxpotPreferenceActionPayload{
			Action: "setSnaxpotPreference",
			Scope:  "persistent",
		})

		require.Error(t, err)
		assert.Equal(t, "action type must be 'clearSnaxpotPreference'", err.Error())
	})
}
