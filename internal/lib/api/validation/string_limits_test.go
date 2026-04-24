package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

func Test_ValidateStringMaxLength_STRING_FIELD_NAME(t *testing.T) {
	t.Parallel()

	err := ValidateStringMaxLength(strings.Repeat("a", MaxEnumFieldLength+1), MaxEnumFieldLength, "side")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "side exceeds maximum length")
}

func Test_ValidateStringMaxLength_FUNC_FIELD_NAME(t *testing.T) {
	t.Parallel()

	err := ValidateStringMaxLength(strings.Repeat("a", MaxEnumFieldLength+1), MaxEnumFieldLength, func() string {
		return "order 3: side"
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order 3: side exceeds maximum length")
}

func Test_ValidateCanonicalSymbol_STRING_FIELD_NAME_SYMBOL_UNWRAPPED(t *testing.T) {
	t.Parallel()

	err := ValidateCanonicalSymbol("!", "symbol")
	require.Error(t, err)
	assert.ErrorIs(t, err, snx_lib_core.ErrInvalidSymbol)
}

func Test_ValidateCanonicalSymbol_FUNC_LABEL_THUNK_NOT_CALLED_WHEN_VALID(t *testing.T) {
	t.Parallel()

	called := false
	err := ValidateCanonicalSymbol("BTC-USDT", func() string {
		called = true

		return "symbols[0]"
	})
	require.NoError(t, err)
	assert.False(t, called)
}

func Test_ValidateCanonicalSymbol_FUNC_LABEL_WHEN_NOT_CANONICAL(t *testing.T) {
	t.Parallel()

	err := ValidateCanonicalSymbol("btc-usdt", func() string {
		return "symbols[1]"
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbols[1] must use canonical uppercase format")
}

func Test_ValidateCanonicalSymbol_FUNC_LABEL_ON_NORMALIZE_ERROR(t *testing.T) {
	t.Parallel()

	err := ValidateCanonicalSymbol("!!!", func() string {
		return "symbols[2]"
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbols[2]:")
}

func Test_ValidateAndNormalizeSymbol_OK(t *testing.T) {
	t.Parallel()

	// Exactly MaxSymbolLength: 18 × 'A', hyphen, trailing 'B' (word chars + hyphen).
	const maxLenPair = "AAAAAAAAAAAAAAAAAA-B"

	tests := []struct {
		name  string
		input Symbol
		want  Symbol
	}{
		{
			name:  "CANONICAL_PAIR",
			input: "BTC-USDT",
			want:  "BTC-USDT",
		},
		{
			name:  "LOWERCASE_NORMALIZED",
			input: "btc-usdt",
			want:  "BTC-USDT",
		},
		{
			name:  "MIXED_CASE_NORMALIZED",
			input: "Eth-Usd",
			want:  "ETH-USD",
		},
		{
			name:  "LEADING_TRAILING_SPACE_TRIMMED_THEN_NORMALIZED",
			input: "  sol-usdt  ",
			want:  "SOL-USDT",
		},
		{
			name:  "MIN_LENGTH_WITH_HYPHEN",
			input: "a-a",
			want:  "A-A",
		},
		{
			name:  "DIGITS_AT_ENDS",
			input: "1-2",
			want:  "1-2",
		},
		{
			name:  "UNDERSCORE_ALLOWED_IN_BODY_WITH_HYPHEN",
			input: "perp_btc-usdt",
			want:  "PERP_BTC-USDT",
		},
		{
			name:  "MAX_LENGTH_WIRE",
			input: maxLenPair,
			want:  maxLenPair,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ValidateAndNormalizeSymbol(tt.input)
			require.NoError(t, err, "ValidateAndNormalizeSymbol")
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_ValidateAndNormalizeSymbol_ERR_SYMBOL_REQUIRED(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input Symbol
	}{
		{name: "EMPTY", input: ""},
		{name: "ONLY_ASCII_SPACES", input: "   "},
		{name: "ONLY_WHITESPACE", input: "\t\n\r "},
		{name: "TWENTY_SPACES_TRIMS_TO_EMPTY_CORE_PATH", input: Symbol(strings.Repeat(" ", MaxSymbolLength))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ValidateAndNormalizeSymbol(tt.input)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrSymbolRequired)
		})
	}
}

func Test_ValidateAndNormalizeSymbol_ERR_INVALID_SYMBOL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input Symbol
	}{
		{name: "TOO_SHORT_NO_HYPHEN", input: "ab"},
		{name: "LEN_THREE_BUT_NO_HYPHEN", input: "abc"},
		{name: "NO_HYPHEN_LONGER", input: "BTCUSDT"},
		{name: "UNDERSCORE_INSTEAD_OF_HYPHEN", input: "btc_usdt"},
		{name: "LEADING_HYPHEN", input: "-USDT"},
		{name: "TRAILING_HYPHEN", input: "BTC-"},
		{name: "NON_WORD_FIRST", input: "!-USDT"},
		{name: "NON_WORD_LAST", input: "BTC-!"},
		{name: "ONLY_HYPHEN", input: "-"},
		{name: "DOUBLE_HYPHEN_EMPTY_SEGMENT_STILL_INVALID_END", input: "BTC--"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ValidateAndNormalizeSymbol(tt.input)
			require.Error(t, err)
			require.ErrorIs(t, err, snx_lib_core.ErrInvalidSymbol)
		})
	}
}

func Test_ValidateAndNormalizeSymbol_EXCEEDS_MAX_LENGTH_BEFORE_TRIM(t *testing.T) {
	t.Parallel()

	// 21 characters: valid market shape if length were ignored; must fail on raw len.
	input := Symbol(strings.Repeat("A", 18) + "-BC")
	require.Greater(t, len(input), MaxSymbolLength, "test data must exceed MaxSymbolLength")

	_, err := ValidateAndNormalizeSymbol(input)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrSymbolRequired))
	require.False(t, errors.Is(err, snx_lib_core.ErrInvalidSymbol))
	assert.Contains(t, err.Error(), "symbol exceeds maximum length of 20 characters")
}

func Test_ValidateAndNormalizeSymbol_EXCEEDS_MAX_LENGTH_SPACES_ONLY(t *testing.T) {
	t.Parallel()

	// Length runs before core validation; long whitespace is not ErrSymbolRequired.
	input := Symbol(strings.Repeat(" ", MaxSymbolLength+1))

	_, err := ValidateAndNormalizeSymbol(input)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrSymbolRequired))
	assert.Contains(t, err.Error(), "symbol exceeds maximum length of 20 characters")
}

func Test_ValidateAndNormalizeSymbol_EXCEEDS_MAX_LENGTH_EVEN_WHEN_TRIM_WOULD_BE_VALID(t *testing.T) {
	t.Parallel()

	// Raw length is checked before TrimSpace in core; one leading space makes the
	// wire 21 bytes while the trimmed body is exactly MaxSymbolLength and valid.
	inner := Symbol(strings.Repeat("A", 18) + "-B")
	require.Equal(t, MaxSymbolLength, len(inner))
	input := " " + inner
	require.Equal(t, MaxSymbolLength+1, len(input))

	_, err := ValidateAndNormalizeSymbol(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbol exceeds maximum length of 20 characters")
}
