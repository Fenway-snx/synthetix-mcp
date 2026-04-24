package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollateralExchangeType_String(t *testing.T) {
	assert.Equal(t, "auto", CollateralExchangeType_Auto.String())
	assert.Equal(t, "voluntary", CollateralExchangeType_Voluntary.String())
	assert.Equal(t, "unknown", CollateralExchangeType_Unknown.String())
	assert.Equal(t, "unknown", CollateralExchangeType(99).String())
}

func TestCollateralExchangeType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    CollateralExchangeType
		expected string
	}{
		{"auto", CollateralExchangeType_Auto, `"auto"`},
		{"voluntary", CollateralExchangeType_Voluntary, `"voluntary"`},
		{"unknown", CollateralExchangeType_Unknown, `"unknown"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestCollateralExchangeType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected CollateralExchangeType
		wantErr  bool
	}{
		{"auto", `"auto"`, CollateralExchangeType_Auto, false},
		{"voluntary", `"voluntary"`, CollateralExchangeType_Voluntary, false},
		{"unknown", `"unknown"`, CollateralExchangeType_Unknown, false},
		{"unrecognized", `"bogus"`, 0, true},
		{"invalid json", `123`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got CollateralExchangeType
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
