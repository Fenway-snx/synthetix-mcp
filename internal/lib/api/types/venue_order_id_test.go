package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_VenueOrderId_MarshalJSON(t *testing.T) {

	tests := []struct {
		name     string
		input    VenueOrderId
		err      error
		expected string
	}{
		{
			name:     "1",
			input:    VenueOrderId("1"),
			expected: `"1"`,
		},
		{
			name:     "1234567890",
			input:    VenueOrderId("1234567890"),
			expected: `"1234567890"`,
		},
		{
			name:     "KNOWINGLY INVALID, because we don't police on the way out",
			input:    VenueOrderId(""),
			expected: `""`,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			data, err := tt.input.MarshalJSON()

			if tt.err != nil {

				assert.Error(t, err)
			} else {

				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

				s := string(data)

				assert.Equal(t, tt.expected, s)
			}
		})
	}
}

func Test_VenueOrderId_UnmarshalJSON(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		err      error
		expected VenueOrderId
	}{
		{
			name:     "1",
			input:    `"1"`,
			err:      nil,
			expected: VenueOrderId("1"),
		},
		{
			name:     "12345678",
			input:    `"12345678"`,
			err:      nil,
			expected: VenueOrderId("12345678"),
		},
		{
			name:     "9223372036854775807",
			input:    fmt.Sprintf(`"%d"`, VenueOrderIdMaximumValidValue),
			err:      nil,
			expected: VenueOrderId("9223372036854775807"),
		},
		// invalid values
		{
			name:  "",
			input: `""`,
			err:   errVenueOrderIdEmpty,
		},
		{
			name:  "0",
			input: `"0"`,
			err:   errVenueOrderIdCannotBeZero,
		},
		{
			name:  "-1",
			input: `"-1"`,
			err:   errVenueOrderIdCannotBeNegative,
		},
		// valid but odd
		{
			name:     "+1",
			input:    `"+1"`,
			expected: VenueOrderId("1"),
		},
		{
			name:     " 12345 ",
			input:    `" 12345 "`,
			expected: VenueOrderId("12345"),
		},
		{
			name:  "+ 1",
			input: `"+ 1"`,
			err:   errVenueOrderIdInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var void VenueOrderId
			err := void.UnmarshalJSON([]byte(tt.input))

			if tt.err != nil {

				assert.Equal(t, tt.err, err)
			} else {

				assert.Nil(t, err)

				assert.Equal(t, tt.expected, void)
			}
		})
	}
}

func Test_VenueOrderIdFromUintUnvalidated(t *testing.T) {

	// NOTE: when testing from `Unvalidated`, this means "coming from the core and will be valid"

	tests := []struct {
		name     string
		v        uint64
		expected VenueOrderId
	}{
		{
			name:     "1",
			v:        1,
			expected: VenueOrderId("1"),
		},
		{
			name:     "12345678",
			v:        12345678,
			expected: VenueOrderId("12345678"),
		},
		{
			name:     "9223372036854775807",
			v:        VenueOrderIdMaximumValidValue,
			expected: VenueOrderId("9223372036854775807"),
		},
		{
			name:     "0",
			v:        0,
			expected: "0",
		},
		{
			name:     "9223372036854775808",
			v:        VenueOrderIdMaximumValidValue + 1,
			expected: VenueOrderId("9223372036854775808"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual := VenueOrderIdFromUintUnvalidated(tt.v)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_VenueOrderIdFromUint(t *testing.T) {

	tests := []struct {
		name     string
		v        uint64
		err      error
		expected VenueOrderId
	}{
		{
			name:     "1",
			v:        1,
			err:      nil,
			expected: VenueOrderId("1"),
		},
		{
			name:     "12345678",
			v:        12345678,
			err:      nil,
			expected: VenueOrderId("12345678"),
		},
		{
			name:     "9223372036854775807",
			v:        VenueOrderIdMaximumValidValue,
			err:      nil,
			expected: VenueOrderId("9223372036854775807"),
		},
		{
			name: "0",
			v:    0,
			err:  errVenueOrderIdCannotBeZero,
		},
		{
			name: "9223372036854775808",
			v:    VenueOrderIdMaximumValidValue + 1,
			err:  errVenueOrderIdTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual, err := VenueOrderIdFromUint(tt.v)

			if tt.err != nil {

				assert.Equal(t, tt.err, err)
			} else {
				assert.Nil(t, err)

				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}
