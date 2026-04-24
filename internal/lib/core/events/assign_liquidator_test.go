package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SLPMappingType_String(t *testing.T) {
	tests := []struct {
		name     string
		value    SLPMappingType
		expected string
	}{
		{"unknown", SLPMappingType_Unknown, "unknown"},
		{"market", SLPMappingType_Market, "market"},
		{"collateral", SLPMappingType_Collateral, "collateral"},
		{"out of range", SLPMappingType(42), "SLPMappingType(42)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.String())
		})
	}
}

func Test_SLPMappingType_Valid(t *testing.T) {
	assert.False(t, SLPMappingType_Unknown.Valid())
	assert.True(t, SLPMappingType_Market.Valid())
	assert.True(t, SLPMappingType_Collateral.Valid())
	assert.False(t, SLPMappingType(99).Valid())
}

func Test_SLPMappingType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    SLPMappingType
		expected string
	}{
		{"market", SLPMappingType_Market, `"market"`},
		{"collateral", SLPMappingType_Collateral, `"collateral"`},
		{"unknown", SLPMappingType_Unknown, `"unknown"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func Test_SLPMappingType_UnmarshalJSON(t *testing.T) {
	t.Run("market", func(t *testing.T) {
		var v SLPMappingType
		require.NoError(t, json.Unmarshal([]byte(`"market"`), &v))
		assert.Equal(t, SLPMappingType_Market, v)
	})

	t.Run("collateral", func(t *testing.T) {
		var v SLPMappingType
		require.NoError(t, json.Unmarshal([]byte(`"collateral"`), &v))
		assert.Equal(t, SLPMappingType_Collateral, v)
	})

	t.Run("unknown string returns error", func(t *testing.T) {
		var v SLPMappingType
		assert.Error(t, json.Unmarshal([]byte(`"garbage"`), &v))
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		var v SLPMappingType
		assert.Error(t, json.Unmarshal([]byte(`123`), &v))
	})
}

func Test_SLPMappingType_RoundTrip(t *testing.T) {
	for _, original := range []SLPMappingType{SLPMappingType_Market, SLPMappingType_Collateral} {
		t.Run(original.String(), func(t *testing.T) {
			data, err := json.Marshal(original)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			var decoded SLPMappingType
			require.NoError(t, json.Unmarshal(data, &decoded))
			assert.Equal(t, original, decoded)
		})
	}
}

func Test_AssignLiquidatorRequest_JSON_RoundTrip(t *testing.T) {
	req := AssignLiquidatorRequest{
		MappingType:  SLPMappingType_Market,
		RequestID:    "test-123",
		SubAccountId: 500,
		Symbol:       "BTC-USDT",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// Verify the JSON contains the string representation
	assert.Contains(t, string(data), `"mapping_type":"market"`)

	var decoded AssignLiquidatorRequest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, req, decoded)
}
