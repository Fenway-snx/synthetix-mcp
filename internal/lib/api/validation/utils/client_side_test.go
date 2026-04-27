package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// localTestClientLabel is a package-local string new-type used only to
// exercise [ValidateClientSideString]'s T ~string constraint.
type localTestClientLabel string

func Test_ClientSideOption_String(t *testing.T) {
	tests := []struct {
		name     string
		options  ClientSideOption
		expected string
	}{
		{
			name:     "None",
			options:  ClientSideOption_None,
			expected: "None",
		},
		{
			name:     "RejectEmpty",
			options:  ClientSideOption_RejectEmpty,
			expected: "RejectEmpty",
		},
		{
			name:     "Trim",
			options:  ClientSideOption_Trim,
			expected: "Trim",
		},
		{
			name:     "RejectEmpty|Trim",
			options:  ClientSideOption_RejectEmpty | ClientSideOption_Trim,
			expected: "RejectEmpty|Trim",
		},
		{
			name:     "Trim|RejectEmpty",
			options:  ClientSideOption_Trim | ClientSideOption_RejectEmpty,
			expected: "RejectEmpty|Trim",
		},
		{
			name:     "3 (=== Trim|RejectEmpty)",
			options:  3,
			expected: "RejectEmpty|Trim",
		},
		{
			name:     "127 (invalid)",
			options:  127,
			expected: "0x7f",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			actual := tt.options.String()

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_ValidateClientSideString(t *testing.T) {

	tests := []struct {
		name                    string
		input                   string
		maxLen                  int
		options                 ClientSideOption
		shouldFail              bool
		expectedValidatedForm   string
		expectedFailureContains string
	}{
		// various forms of empty
		{
			name:                  "empty string",
			input:                 "",
			options:               ClientSideOption_None,
			expectedValidatedForm: "",
		},
		{
			name:                    "empty string (empty-rejected)",
			input:                   "",
			options:                 ClientSideOption_RejectEmpty,
			shouldFail:              true,
			expectedFailureContains: "empty input",
		},
		{
			name:                    "whitespace only",
			input:                   "  ",
			options:                 ClientSideOption_None,
			shouldFail:              true,
			expectedFailureContains: "invalid string",
		},
		{
			name:                  "whitespace only (trimmed)",
			input:                 "  ",
			options:               ClientSideOption_Trim,
			expectedValidatedForm: "",
		},
		{
			name:                    "whitespace only (trimmed, empty-rejected)",
			input:                   "  ",
			options:                 ClientSideOption_Trim | ClientSideOption_RejectEmpty,
			shouldFail:              true,
			expectedFailureContains: "empty input",
		},
		// valid
		{
			name:                  "letters and numbers",
			input:                 "TheCatSatOnTheMat123",
			options:               ClientSideOption_Trim,
			expectedValidatedForm: "TheCatSatOnTheMat123",
		},
		{
			name:                  "letters and numbers (sufficient length)",
			input:                 "TheCatSatOnTheMat123",
			maxLen:                20,
			options:               ClientSideOption_Trim,
			expectedValidatedForm: "TheCatSatOnTheMat123",
		},
		{
			name:                    "letters and numbers (too-long)",
			input:                   "TheCatSatOnTheMat123",
			maxLen:                  19,
			options:                 ClientSideOption_Trim,
			shouldFail:              true,
			expectedFailureContains: "too long",
		},
		{
			name:                  "all valid characters",
			input:                 "The-Cat.Sat=On/The+Mat_123",
			options:               ClientSideOption_Trim,
			expectedValidatedForm: "The-Cat.Sat=On/The+Mat_123",
		},
		{
			name:                    "letters and numbers (too-long)",
			input:                   "TheCatSatOnTheMat123",
			maxLen:                  19,
			options:                 ClientSideOption_Trim,
			shouldFail:              true,
			expectedFailureContains: "too long",
		},
		// invalid
		{
			name:                    "letters and numbers (and *)",
			input:                   "TheCatSatOnTheMat*123",
			options:                 ClientSideOption_Trim,
			shouldFail:              true,
			expectedFailureContains: "invalid",
		},
		{
			name:                    "letters and numbers (and %)",
			input:                   "The%CatSatOnTheMat123",
			options:                 ClientSideOption_Trim,
			shouldFail:              true,
			expectedFailureContains: "invalid",
		},
		{
			name:                    "letters and numbers (and ' ')",
			input:                   "The Cat Sat On The Mat 123",
			options:                 ClientSideOption_Trim,
			shouldFail:              true,
			expectedFailureContains: "invalid",
		},
		{
			name:                  "base64 encoded string with spaces passes strict validation",
			input:                 "VGhlIENhdCBTYXQgT24gVGhlIE1hdCAxMjM=",
			options:               ClientSideOption_Trim,
			expectedValidatedForm: "VGhlIENhdCBTYXQgT24gVGhlIE1hdCAxMjM=",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			validatedForm, err := ValidateClientSideString(tt.input, tt.maxLen, tt.options)

			if tt.shouldFail {

				assert.NotNil(t, err, "expected `err` not to be `nil`, but it was")

				assert.Contains(t, err.Error(), tt.expectedFailureContains)

				assert.Equal(t, "", validatedForm)
			} else {

				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

				assert.Equal(t, tt.expectedValidatedForm, validatedForm)
			}
		})
	}
}

func Test_ValidateClientSideString_LOCAL_STRING_NEW_TYPE(t *testing.T) {
	t.Parallel()

	t.Run("valid value preserves nominal type", func(t *testing.T) {
		t.Parallel()

		in := localTestClientLabel("request-label-01")
		out, err := ValidateClientSideString(in, 64, ClientSideOption_Trim)

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, localTestClientLabel("request-label-01"), out)

		var asIface any = out
		_, isPlainString := asIface.(string)

		assert.False(t, isPlainString, "result should remain localTestClientLabel, not string")
	})

	t.Run("trim applies before validation", func(t *testing.T) {
		t.Parallel()

		in := localTestClientLabel("  trimmed-id  ")
		out, err := ValidateClientSideString(in, 64, ClientSideOption_Trim)

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, localTestClientLabel("trimmed-id"), out)
	})

	t.Run("reject empty after trim", func(t *testing.T) {
		t.Parallel()

		in := localTestClientLabel("   ")
		out, err := ValidateClientSideString(
			in,
			64,
			ClientSideOption_Trim|ClientSideOption_RejectEmpty,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty input")

		var zero localTestClientLabel

		assert.Equal(t, zero, out)
	})

	t.Run("invalid characters returns zero value of new-type", func(t *testing.T) {
		t.Parallel()

		in := localTestClientLabel("bad value")
		out, err := ValidateClientSideString(in, 64, ClientSideOption_Trim)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")

		var zero localTestClientLabel

		assert.Equal(t, zero, out)
	})
}
