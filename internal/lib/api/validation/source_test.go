package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateSource(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
		expected    string // expected value after validation (trimmed)
	}{
		{
			name:        "empty source is valid",
			source:      "",
			expectError: false,
			expected:    "",
		},
		{
			name:        "valid alphanumeric source",
			source:      "web-ui",
			expectError: false,
			expected:    "web-ui",
		},
		{
			name:        "valid source with allowed special chars",
			source:      "mobile-app_v2.0",
			expectError: false,
			expected:    "mobile-app_v2.0",
		},
		{
			name:        "valid source with equals and plus",
			source:      "api=v1+test",
			expectError: false,
			expected:    "api=v1+test",
		},
		{
			name:        "valid source with forward slash",
			source:      "ui/dashboard/main",
			expectError: false,
			expected:    "ui/dashboard/main",
		},
		{
			name:        "source with leading/trailing whitespace gets trimmed",
			source:      "  web-ui  ",
			expectError: false,
			expected:    "web-ui",
		},
		{
			name:        "whitespace-only string becomes empty (valid)",
			source:      "   ",
			expectError: false,
			expected:    "",
		},
		{
			name:        "SQL injection attempt with single quote",
			source:      "'; DROP TABLE orders; --",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "SQL injection attempt with semicolon",
			source:      "web-ui; DELETE FROM users",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "SQL injection with comment delimiter",
			source:      "web-ui/**/UNION/**/SELECT",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "SQL injection with parentheses",
			source:      "web-ui OR 1=1 --",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "source with space is invalid",
			source:      "web ui",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "source with special characters",
			source:      "web@ui.com",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "source with percent encoding attempt",
			source:      "%27%20OR%201=1",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "source with dollar sign",
			source:      "$mobile",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "source with ampersand",
			source:      "web&mobile",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "source with brackets",
			source:      "web[0]",
			expectError: true,
			errorMsg:    "invalid source attribute",
		},
		{
			name:        "source exceeding max length",
			source:      strings.Repeat("a", SourceMaxLength+1),
			expectError: true,
			errorMsg:    "100",
		},
		{
			name:        "source at exact max length",
			source:      strings.Repeat("a", SourceMaxLength),
			expectError: false,
			expected:    strings.Repeat("a", SourceMaxLength),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateSource(tt.source)

			if tt.expectError {
				require.Error(t, err, "expected error but got none")
				assert.Contains(t, err.Error(), tt.errorMsg,
					"error message should contain '%s' but got: %v", tt.errorMsg, err,
				)
			} else {
				require.NoError(t, err, "unexpected error: %v", err)
				assert.Equal(t, tt.expected, result,
					"validated source should match expected value",
				)
			}
		})
	}
}
