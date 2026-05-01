package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

// =========================================================================
// MarshalJSON tests
// =========================================================================

func Test_ClientOrderId_MarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		cloid          ClientOrderId
		expectedJSON   string
		shouldFail     bool
		expectedErrMsg string
	}{
		{
			name:         "empty string",
			cloid:        ClientOrderId_Empty,
			expectedJSON: `""`,
		},
		{
			name:         "simple numeric string",
			cloid:        ClientOrderId("12345"),
			expectedJSON: `"12345"`,
		},
		{
			name:         "alphanumeric string",
			cloid:        ClientOrderId("abc123"),
			expectedJSON: `"abc123"`,
		},
		{
			name:         "string with special characters",
			cloid:        ClientOrderId("test-123.456=789/abc+def_ghi"),
			expectedJSON: `"test-123.456=789/abc+def_ghi"`,
		},
		{
			name:         "long string",
			cloid:        ClientOrderId("a"),
			expectedJSON: `"a"`,
		},
		{
			name:         "unicode characters (should marshal but may fail validation)",
			cloid:        ClientOrderId("test-ü-123"),
			expectedJSON: `"test-ü-123"`,
		},
		{
			name:         "whitespace only",
			cloid:        ClientOrderId("   "),
			expectedJSON: `"   "`,
		},
		{
			name:         "string with leading/trailing whitespace",
			cloid:        ClientOrderId("  test123  "),
			expectedJSON: `"  test123  "`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := tt.cloid.MarshalJSON()

			if tt.shouldFail {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tt.expectedJSON, string(bytes))
			}
		})
	}
}

// =========================================================================
// UnmarshalJSON tests
// =========================================================================

func Test_ClientOrderId_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		jsonInput      string
		expectedCloid  ClientOrderId
		shouldFail     bool
		expectedErrMsg string
	}{
		// Valid cases
		{
			name:          "valid numeric string",
			jsonInput:     `"12345"`,
			expectedCloid: ClientOrderId("12345"),
		},
		{
			name:          "valid alphanumeric string",
			jsonInput:     `"abc123"`,
			expectedCloid: ClientOrderId("abc123"),
		},
		{
			name:          "valid string with special characters",
			jsonInput:     `"test-123.456=789/abc+def_ghi"`,
			expectedCloid: ClientOrderId("test-123.456=789/abc+def_ghi"),
		},
		{
			name:           "rejects leading and trailing whitespace",
			jsonInput:      `"  test123  "`,
			shouldFail:     true,
			expectedErrMsg: "leading or trailing whitespace",
		},
		{
			name:           "rejects leading whitespace",
			jsonInput:      `"  test123"`,
			shouldFail:     true,
			expectedErrMsg: "leading or trailing whitespace",
		},
		{
			name:           "rejects trailing whitespace",
			jsonInput:      `"test123  "`,
			shouldFail:     true,
			expectedErrMsg: "leading or trailing whitespace",
		},
		{
			name:          "valid single character",
			jsonInput:     `"a"`,
			expectedCloid: ClientOrderId("a"),
		},
		{
			name:           "rejects single character with leading whitespace",
			jsonInput:      `" a"`,
			shouldFail:     true,
			expectedErrMsg: "leading or trailing whitespace",
		},
		{
			name:          "valid single digit",
			jsonInput:     `"1"`,
			expectedCloid: ClientOrderId("1"),
		},
		{
			name:          "valid string with all allowed special characters",
			jsonInput:     `"-._=+/"`,
			expectedCloid: ClientOrderId("-._=+/"),
		},
		{
			name:          "valid long string (255 chars)",
			jsonInput:     `"` + generateValidString(255) + `"`,
			expectedCloid: ClientOrderId(generateValidString(255)),
		},
		{
			name:          "empty string (currently allowed)",
			jsonInput:     `""`,
			expectedCloid: ClientOrderId(""),
		},
		{
			name:           "whitespace only is rejected",
			jsonInput:      `"   "`,
			shouldFail:     true,
			expectedErrMsg: "leading or trailing whitespace",
		},
		// Invalid cases
		{
			name:       "invalid JSON (not a string)",
			jsonInput:  `12345`,
			shouldFail: true,
		},
		{
			name:          "invalid JSON (null, unmarshals to empty)",
			jsonInput:     `null`,
			expectedCloid: ClientOrderId(""),
		},
		{
			name:       "invalid JSON (object)",
			jsonInput:  `{}`,
			shouldFail: true,
		},
		{
			name:       "invalid JSON (array)",
			jsonInput:  `[]`,
			shouldFail: true,
		},
		{
			name:           "invalid characters (space in middle)",
			jsonInput:      `"test 123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "invalid characters (asterisk)",
			jsonInput:      `"test*123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "invalid characters (percent)",
			jsonInput:      `"test%123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "invalid characters (hash)",
			jsonInput:      `"test#123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "invalid characters (at sign)",
			jsonInput:      `"test@123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "invalid characters (exclamation)",
			jsonInput:      `"test!123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "too long (256 chars)",
			jsonInput:      `"` + generateValidString(256) + `"`,
			shouldFail:     true,
			expectedErrMsg: "too long",
		},
		{
			name:           "too long (300 chars)",
			jsonInput:      `"` + generateValidString(300) + `"`,
			shouldFail:     true,
			expectedErrMsg: "too long",
		},
		{
			name:           "unicode characters",
			jsonInput:      `"test-ü-123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "emoji characters",
			jsonInput:      `"test-😀-123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "newline character",
			jsonInput:      `"test\n123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
		{
			name:           "tab character",
			jsonInput:      `"test\t123"`,
			shouldFail:     true,
			expectedErrMsg: "invalid string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cloid ClientOrderId
			err := json.Unmarshal([]byte(tt.jsonInput), &cloid)

			if tt.shouldFail {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tt.expectedCloid, cloid)
			}
		})
	}
}

// =========================================================================
// Round-trip tests (Marshal -> Unmarshal)
// =========================================================================

func Test_ClientOrderId_MarshalUnmarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		cloid ClientOrderId
	}{
		{
			name:  "simple numeric",
			cloid: ClientOrderId("12345"),
		},
		{
			name:  "alphanumeric",
			cloid: ClientOrderId("abc123"),
		},
		{
			name:  "with special characters",
			cloid: ClientOrderId("test-123.456=789/abc+def_ghi"),
		},
		{
			name:  "single character",
			cloid: ClientOrderId("a"),
		},
		{
			name:  "all digits",
			cloid: ClientOrderId("9876543210"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			bytes, err := json.Marshal(tt.cloid)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// Unmarshal
			var unmarshaled ClientOrderId
			err = json.Unmarshal(bytes, &unmarshaled)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// Verify round-trip
			assert.Equal(t, tt.cloid, unmarshaled)
		})
	}
}

// =========================================================================
// Utility function tests
// =========================================================================

func Test_ClientOrderIdFromStringUnvalidated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ClientOrderId
	}{
		{
			name:     "empty string",
			input:    "",
			expected: ClientOrderId(""),
		},
		{
			name:     "valid string",
			input:    "12345",
			expected: ClientOrderId("12345"),
		},
		{
			name:     "string with special characters",
			input:    "test-123",
			expected: ClientOrderId("test-123"),
		},
		{
			name:     "whitespace string",
			input:    "   ",
			expected: ClientOrderId("   "),
		},
		{
			name:     "invalid characters (should still work - unvalidated)",
			input:    "test*123",
			expected: ClientOrderId("test*123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClientOrderIdFromStringUnvalidated(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_ClientOrderIdToStringUnvalidated(t *testing.T) {
	tests := []struct {
		name     string
		cloid    ClientOrderId
		expected string
	}{
		{
			name:     "empty string",
			cloid:    ClientOrderId_Empty,
			expected: "",
		},
		{
			name:     "valid string",
			cloid:    ClientOrderId("12345"),
			expected: "12345",
		},
		{
			name:     "string with special characters",
			cloid:    ClientOrderId("test-123"),
			expected: "test-123",
		},
		{
			name:     "whitespace string",
			cloid:    ClientOrderId("   "),
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClientOrderIdToStringUnvalidated(tt.cloid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_ClientOrderIdToStringPtrUnvalidated(t *testing.T) {
	tests := []struct {
		name     string
		cloid    ClientOrderId
		expected *string
	}{
		{
			name:     "empty string",
			cloid:    ClientOrderId_Empty,
			expected: snx_lib_utils_test.MakePointerOf(""),
		},
		{
			name:     "valid string",
			cloid:    ClientOrderId("12345"),
			expected: snx_lib_utils_test.MakePointerOf("12345"),
		},
		{
			name:     "string with special characters",
			cloid:    ClientOrderId("test-123"),
			expected: snx_lib_utils_test.MakePointerOf("test-123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClientOrderIdToStringPtrUnvalidated(tt.cloid)
			assert.NotNil(t, result)
			assert.Equal(t, *tt.expected, *result)
		})
	}
}

func Test_ClientOrderIdPtrFromStringPtrUnvalidated(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected *ClientOrderId
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty string pointer",
			input:    snx_lib_utils_test.MakePointerOf(""),
			expected: snx_lib_utils_test.MakePointerOf(ClientOrderId("")),
		},
		{
			name:     "valid string pointer",
			input:    snx_lib_utils_test.MakePointerOf("12345"),
			expected: snx_lib_utils_test.MakePointerOf(ClientOrderId("12345")),
		},
		{
			name:     "string with special characters",
			input:    snx_lib_utils_test.MakePointerOf("test-123"),
			expected: snx_lib_utils_test.MakePointerOf(ClientOrderId("test-123")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClientOrderIdPtrFromStringPtrUnvalidated(tt.input)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func Test_ClientOrderIdPtrToStringPtrUnvalidated(t *testing.T) {
	tests := []struct {
		name     string
		input    *ClientOrderId
		expected *string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty string pointer",
			input:    snx_lib_utils_test.MakePointerOf(ClientOrderId_Empty),
			expected: snx_lib_utils_test.MakePointerOf(""),
		},
		{
			name:     "valid string pointer",
			input:    snx_lib_utils_test.MakePointerOf(ClientOrderId("12345")),
			expected: snx_lib_utils_test.MakePointerOf("12345"),
		},
		{
			name:     "string with special characters",
			input:    snx_lib_utils_test.MakePointerOf(ClientOrderId("test-123")),
			expected: snx_lib_utils_test.MakePointerOf("test-123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClientOrderIdPtrToStringPtrUnvalidated(tt.input)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

// =========================================================================
// Helper functions
// =========================================================================

// Creates a string of the requested length using valid client ID characters.
func generateValidString(length int) string {
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._=+/_"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = validChars[i%len(validChars)]
	}
	return string(result)
}
