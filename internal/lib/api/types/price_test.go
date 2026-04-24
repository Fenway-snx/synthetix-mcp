package types

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PriceFromDecimal(t *testing.T) {
}

func Test_PriceFromDecimalOrBlankWhenZero(t *testing.T) {
	tests := []struct {
		name string
		v    shopspring_decimal.Decimal
		want Price
	}{
		{
			name: "shopspring_zero_is_blank",
			v:    shopspring_decimal.Zero,
			want: Price_None,
		},
		{
			name: "parsed_zero_not_identical_to_decimal_Zero_keeps_string_zero",
			v:    shopspring_decimal.RequireFromString("0"),
			want: Price("0"),
		},
		{
			name: "parsed_zero_with_fractional_zeros_not_identical_to_decimal_Zero",
			v:    shopspring_decimal.RequireFromString("0.000"),
			want: Price("0"),
		},
		{
			name: "one",
			v:    shopspring_decimal.New(1, 0),
			want: Price("1"),
		},
		{
			name: "scaled_one_tenth",
			v:    shopspring_decimal.New(1, 1),
			want: Price("10"),
		},
		{
			name: "twelve_point_three_four_five",
			v:    shopspring_decimal.New(12345, -3),
			want: Price("12.345"),
		},
		{
			name: "negative",
			v:    shopspring_decimal.New(-12345678, -5),
			want: Price("-123.45678"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriceFromDecimalOrBlankWhenZero(tt.v)

			assert.Equal(t, tt.want, got, "PriceFromDecimalOrBlankWhenZero(%v)", tt.v)
		})
	}
}

func Test_PriceFromDecimalUnvalidated(t *testing.T) {
	tests := []struct {
		name string
		v    shopspring_decimal.Decimal
		want Price
	}{
		{
			name: "shopspring_zero_strings_as_zero",
			v:    shopspring_decimal.Zero,
			want: Price("0"),
		},
		{
			name: "parsed_zero",
			v:    shopspring_decimal.RequireFromString("0"),
			want: Price("0"),
		},
		{
			name: "parsed_zero_with_fractional_zeros_normalizes_per_shopspring_String",
			v:    shopspring_decimal.RequireFromString("0.000"),
			want: Price(shopspring_decimal.RequireFromString("0.000").String()),
		},
		{
			name: "one",
			v:    shopspring_decimal.New(1, 0),
			want: Price("1"),
		},
		{
			name: "scaled_one_tenth",
			v:    shopspring_decimal.New(1, 1),
			want: Price("10"),
		},
		{
			name: "twelve_point_three_four_five",
			v:    shopspring_decimal.New(12345, -3),
			want: Price("12.345"),
		},
		{
			name: "negative",
			v:    shopspring_decimal.New(-12345678, -5),
			want: Price("-123.45678"),
		},
		{
			name: "matches_decimal_String_invariant",
			v:    shopspring_decimal.RequireFromString("50000.25"),
			want: Price("50000.25"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriceFromDecimalUnvalidated(tt.v)

			assert.Equal(t, tt.want, got, "PriceFromDecimalUnvalidated(%v)", tt.v)
			assert.Equal(t, Price(tt.v.String()), got, "must match Decimal.String()")
		})
	}
}

func Test_PriceFromStringUnvalidated(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Price
	}{
		{
			name: "empty",
			in:   "",
			want: Price_None,
		},
		{
			name: "zero_digit",
			in:   "0",
			want: Price("0"),
		},
		{
			name: "positive_decimal",
			in:   "1.25",
			want: Price("1.25"),
		},
		{
			name: "negative_decimal",
			in:   "-0.5",
			want: Price("-0.5"),
		},
		{
			name: "leading_zeros_preserved_as_given",
			in:   "007.0",
			want: Price("007.0"),
		},
		{
			name: "non_numeric_trusted_garbage_still_wrapped",
			in:   "not-a-number",
			want: Price("not-a-number"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriceFromStringUnvalidated(tt.in)

			assert.Equal(t, tt.want, got, "PriceFromStringUnvalidated(%q)", tt.in)
		})
	}
}

func Test_PricePtrFromStringUnvalidated(t *testing.T) {
	t.Run("non_empty_returns_distinct_heap_pointers", func(t *testing.T) {
		p1 := PricePtrFromStringUnvalidated("1.5")
		p2 := PricePtrFromStringUnvalidated("1.5")

		require.NotNil(t, p1, "expected `p1` to be non-nil, but it was `nil`")
		require.NotNil(t, p2, "expected `p2` to be non-nil, but it was `nil`")
		assert.NotSame(t, p1, p2, "each call should allocate a new *Price")
		assert.Equal(t, Price("1.5"), *p1)
		assert.Equal(t, Price("1.5"), *p2)
	})

	t.Run("empty_string_returns_non_nil_pointer_to_empty_price", func(t *testing.T) {
		p := PricePtrFromStringUnvalidated("")

		require.NotNil(t, p, "expected `p` to be non-nil, but it was `nil`")
		assert.Equal(t, Price_None, *p)
	})
}

func Test_PricePtrToStringPtr(t *testing.T) {
	t.Run("nil_price_pointer_returns_nil", func(t *testing.T) {
		assert.Nil(t, PricePtrToStringPtr(nil))
	})

	t.Run("non_nil_price_returns_equivalent_string_pointer", func(t *testing.T) {
		p := Price("3.14159")
		sp := PricePtrToStringPtr(&p)

		require.NotNil(t, sp, "expected `sp` to be non-nil, but it was `nil`")
		assert.Equal(t, "3.14159", *sp)
	})

	t.Run("empty_price_returns_pointer_to_empty_string", func(t *testing.T) {
		p := Price_None
		sp := PricePtrToStringPtr(&p)

		require.NotNil(t, sp, "expected `sp` to be non-nil, but it was `nil`")
		assert.Equal(t, "", *sp)
	})
}

func Test_Price_PTR_HELPERS_ROUND_TRIP_VALUE_THROUGH_UNVALIDATED_PTR_CTOR(t *testing.T) {
	p := PriceFromStringUnvalidated("0.001")
	ptr := PricePtrFromStringUnvalidated(string(p))
	sp := PricePtrToStringPtr(ptr)

	require.NotNil(t, sp, "expected `sp` to be non-nil, but it was `nil`")
	assert.Equal(t, "0.001", *sp)
}
