package pricing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PriceUpdateMode_VALUES(t *testing.T) {

	assert.Equal(t, 0x02, int(PriceUpdateMode_Separate))
	assert.Equal(t, 0x04, int(PriceUpdateMode_Batched))
	assert.Equal(t, 0x06, int(PriceUpdateMode_All))
}

func Test_ParsePriceUpdateMode(t *testing.T) {

	tests := []struct {
		name                      string
		input                     string
		expectedMode              PriceUpdateMode
		shouldFail                bool
		expectedErrStringFragment string
	}{
		{
			name:         "empty string",
			input:        "",
			expectedMode: PriceUpdateMode_None,
		},
		{
			name:         "whitespace only",
			input:        "",
			expectedMode: PriceUpdateMode_None,
		},
		{
			name:         "none",
			input:        "none",
			expectedMode: PriceUpdateMode_None,
		},
		{
			name:         "separate",
			input:        "separate",
			expectedMode: PriceUpdateMode_Separate,
		},
		{
			name:         "batched",
			input:        "batched",
			expectedMode: PriceUpdateMode_Batched,
		},
		{
			name:         "batched|separate",
			input:        "batched|separate",
			expectedMode: PriceUpdateMode_Batched | PriceUpdateMode_Separate,
		},
		{
			name:         "separate | batched",
			input:        "separate | batched",
			expectedMode: PriceUpdateMode_Batched | PriceUpdateMode_Separate,
		},
		{
			name:         "*",
			input:        "*",
			expectedMode: PriceUpdateMode_Batched | PriceUpdateMode_Separate,
		},
		{
			name:         "* | batched",
			input:        "*",
			expectedMode: PriceUpdateMode_Batched | PriceUpdateMode_Separate,
		},
		{
			name:                      "invalid: single unrecognised value",
			input:                     " unknown",
			shouldFail:                true,
			expectedErrStringFragment: "invalid mode specifier 'unknown'",
		},
		{
			name:                      "invalid: one unrecognised value out of set",
			input:                     "separate | batch",
			shouldFail:                true,
			expectedErrStringFragment: "invalid mode specifier 'batch'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if mode, err := ParsePriceUpdateMode(tt.input); err != nil {

				require.True(t, tt.shouldFail, "case failed but was expected to succeed")

				assert.Contains(t, err.Error(), tt.expectedErrStringFragment)
			} else {

				require.False(t, tt.shouldFail, "case succeeded but was expected to fail")

				assert.Equal(t, tt.expectedMode, mode)
			}
		})
	}
}
