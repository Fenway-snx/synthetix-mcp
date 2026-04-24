package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DirectionToString(t *testing.T) {
	// Returned strings for defined values must stay aligned with
	// lib/api/types.Direction_short, Direction_long, Direction_closeShort,
	// Direction_closeLong (used by DirectionFromInternalDirection).
	tests := []struct {
		name      string
		dir       Direction
		wantS     string
		wantKnown bool
	}{
		{
			name:      "short_opening",
			dir:       Direction_Short,
			wantS:     "short",
			wantKnown: true,
		},
		{
			name:      "long_opening",
			dir:       Direction_Long,
			wantS:     "long",
			wantKnown: true,
		},
		{
			name:      "close_short",
			dir:       Direction_CloseShort,
			wantS:     "closeShort",
			wantKnown: true,
		},
		{
			name:      "close_long",
			dir:       Direction_CloseLong,
			wantS:     "closeLong",
			wantKnown: true,
		},
		{
			name:      "unknown_positive",
			dir:       Direction(99),
			wantS:     "UNKNOWN-Direction<v=99>",
			wantKnown: false,
		},
		{
			name:      "unknown_negative",
			dir:       Direction(-1),
			wantS:     "UNKNOWN-Direction<v=-1>",
			wantKnown: false,
		},
		{
			name:      "unknown_large_iota_gap",
			dir:       Direction(1000),
			wantS:     "UNKNOWN-Direction<v=1000>",
			wantKnown: false,
		},
		{
			name: "unknown_first_value_after_last_defined_enum",
			// Direction_CloseLong is the last iota (3); the next integer is not a
			// defined direction.
			dir:       Direction(4),
			wantS:     "UNKNOWN-Direction<v=4>",
			wantKnown: false,
		},
		{
			name:      "unknown_min_int32",
			dir:       Direction(math.MinInt32),
			wantS:     "UNKNOWN-Direction<v=-2147483648>",
			wantKnown: false,
		},
		{
			name:      "unknown_max_int32",
			dir:       Direction(math.MaxInt32),
			wantS:     "UNKNOWN-Direction<v=2147483647>",
			wantKnown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotS, gotKnown := DirectionToString(tt.dir)
			assert.Equal(t, tt.wantS, gotS)
			assert.Equal(t, tt.wantKnown, gotKnown)
		})
	}
}

func Test_TriggerPriceType_FromString(t *testing.T) {

	t.Run("mark", func(t *testing.T) {
		tpt, err := CoreTriggerPriceTypeFromString("mark")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, TriggerPriceTypeMarkPrice, tpt)
	})

	t.Run("last", func(t *testing.T) {
		tpt, err := CoreTriggerPriceTypeFromString("last")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, TriggerPriceTypeLastPrice, tpt)
	})

	t.Run("unknown returns error", func(t *testing.T) {
		_, err := CoreTriggerPriceTypeFromString("anything")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTriggerPriceType)
	})

	t.Run("empty returns error", func(t *testing.T) {
		_, err := CoreTriggerPriceTypeFromString("")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTriggerPriceType)
	})
}
