package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type encapsulatedString struct {
	v string
}

func (es encapsulatedString) String() string {
	return es.v
}

func Test_TryGetStringFromAnyOrEmpty(t *testing.T) {

	tests := []struct {
		name     string
		expected string
		input    any
	}{
		{
			name:     "empty string",
			expected: "",
			input:    "",
		},
		{
			name:     "number",
			expected: "",
			input:    -1,
		},
		{
			name:     "number",
			expected: "",
			input: encapsulatedString{
				v: "",
			},
		},
		{
			name:     "small string",
			expected: "abc ",
			input:    "abc ",
		},
		{
			name:     "small string",
			expected: "abcdefghijklmnopqrstuvwxyz",
			input: encapsulatedString{
				v: "abcdefghijklmnopqrstuvwxyz",
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			expected := tt.expected
			actual := TryGetStringFromAnyOrEmpty(tt.input)

			assert.Equal(t, expected, actual)
		})
	}
}
