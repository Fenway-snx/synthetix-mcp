package auth

import (
	"testing"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
)

func Test_ShortAddress(t *testing.T) {
	tests := []struct {
		in   snx_lib_api_types.WalletAddress
		out  string
		name string
	}{
		{"", "", "empty"},
		{"0xABCD", "0xABCD", "short_preserve_case"},
		{"ABCDEF", "ABCDEF", "no0x_short_preserve_case"},
		{"0xABCDEF0123456789abcdef0123456789ABCDEF", "0xABC...DEF", "long_truncated_preserve_case"},
	}

	for _, tt := range tests {
		got := ShortAddress(tt.in)
		if got != tt.out {
			t.Fatalf("%s: ShortAddress(%q) = %q, want %q", tt.name, tt.in, got, tt.out)
		}
	}
}
