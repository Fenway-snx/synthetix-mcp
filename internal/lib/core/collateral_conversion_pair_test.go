package core

import "testing"

func Test_NormalizeCollateralConversionPair(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty defaults to nominated collateral lower", "", "usdt"},
		{"whitespace only defaults", "   ", "usdt"},
		{"trim and lower", "  USDT  ", "usdt"},
		{"already normalized", "usdt", "usdt"},
		{"mixed case pair", "UsDc", "usdc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeCollateralConversionPair(tt.in); got != tt.want {
				t.Fatalf("NormalizeCollateralConversionPair(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
