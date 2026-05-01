package types

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_QuantityFromStringUnvalidated(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Quantity
	}{
		{
			name: "empty",
			in:   "",
			want: Quantity_None,
		},
		{
			name: "zero_digit",
			in:   "0",
			want: Quantity("0"),
		},
		{
			name: "positive_decimal",
			in:   "1.25",
			want: Quantity("1.25"),
		},
		{
			name: "negative_decimal",
			in:   "-0.5",
			want: Quantity("-0.5"),
		},
		{
			name: "leading_zeros_preserved_as_given",
			in:   "007.0",
			want: Quantity("007.0"),
		},
		{
			name: "non_numeric_trusted_garbage_still_wrapped",
			in:   "not-a-number",
			want: Quantity("not-a-number"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuantityFromStringUnvalidated(tt.in)

			assert.Equal(t, tt.want, got, "QuantityFromStringUnvalidated(%q)", tt.in)
		})
	}
}

func Test_QuantityFromDecimalOrBlankWhenZero(t *testing.T) {
	tests := []struct {
		name string
		v    shopspring_decimal.Decimal
		want Quantity
	}{
		{
			name: "shopspring_zero_is_blank",
			v:    shopspring_decimal.Zero,
			want: Quantity_None,
		},
		{
			name: "parsed_zero_not_identical_to_decimal_Zero_keeps_string_zero",
			v:    shopspring_decimal.RequireFromString("0"),
			want: Quantity("0"),
		},
		{
			name: "parsed_zero_with_fractional_zeros_not_identical_to_decimal_Zero",
			v:    shopspring_decimal.RequireFromString("0.000"),
			want: Quantity("0"),
		},
		{
			name: "one",
			v:    shopspring_decimal.New(1, 0),
			want: Quantity("1"),
		},
		{
			name: "scaled_one_tenth",
			v:    shopspring_decimal.New(1, 1),
			want: Quantity("10"),
		},
		{
			name: "twelve_point_three_four_five",
			v:    shopspring_decimal.New(12345, -3),
			want: Quantity("12.345"),
		},
		{
			name: "negative",
			v:    shopspring_decimal.New(-12345678, -5),
			want: Quantity("-123.45678"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuantityFromDecimalOrBlankWhenZero(tt.v)

			assert.Equal(t, tt.want, got, "QuantityFromDecimalOrBlankWhenZero(%v)", tt.v)
		})
	}
}

func Test_QuantityPtrFromStringUnvalidated(t *testing.T) {
	t.Run("non_empty_returns_distinct_heap_pointers", func(t *testing.T) {
		p1 := QuantityPtrFromStringUnvalidated("1.5")
		p2 := QuantityPtrFromStringUnvalidated("1.5")

		require.NotNil(t, p1, "expected `p1` to be non-nil, but it was `nil`")
		require.NotNil(t, p2, "expected `p2` to be non-nil, but it was `nil`")
		assert.NotSame(t, p1, p2, "each call should allocate a new *Quantity")
		assert.Equal(t, Quantity("1.5"), *p1)
		assert.Equal(t, Quantity("1.5"), *p2)
	})

	t.Run("empty_string_returns_non_nil_pointer_to_empty_quantity", func(t *testing.T) {
		p := QuantityPtrFromStringUnvalidated("")

		require.NotNil(t, p, "expected `p` to be non-nil, but it was `nil`")
		assert.Equal(t, Quantity_None, *p)
	})
}

func Test_QuantityPtrFromStringPtrUnvalidated(t *testing.T) {
	t.Run("nil_string_pointer_returns_nil", func(t *testing.T) {
		assert.Nil(t, QuantityPtrFromStringPtrUnvalidated(nil))
	})

	t.Run("non_nil_string_pointer_returns_pointer_to_quantity", func(t *testing.T) {
		s := "42.0"
		p := QuantityPtrFromStringPtrUnvalidated(&s)

		require.NotNil(t, p, "expected `p` to be non-nil, but it was `nil`")
		assert.Equal(t, Quantity("42.0"), *p)
	})

	t.Run("empty_string_pointer_returns_non_nil_empty_quantity", func(t *testing.T) {
		s := ""
		p := QuantityPtrFromStringPtrUnvalidated(&s)

		require.NotNil(t, p, "expected `p` to be non-nil, but it was `nil`")
		assert.Equal(t, Quantity_None, *p)
	})

	t.Run("source_string_mutation_after_call_does_not_change_quantity", func(t *testing.T) {
		s := "9"
		p := QuantityPtrFromStringPtrUnvalidated(&s)
		s = "changed"

		require.NotNil(t, p, "expected `p` to be non-nil, but it was `nil`")
		assert.Equal(t, Quantity("9"), *p)
	})
}

func Test_QuantityPtrToStringPtr(t *testing.T) {
	t.Run("nil_quantity_pointer_returns_nil", func(t *testing.T) {
		assert.Nil(t, QuantityPtrToStringPtr(nil))
	})

	t.Run("non_nil_quantity_returns_equivalent_string_pointer", func(t *testing.T) {
		q := Quantity("3.14159")
		sp := QuantityPtrToStringPtr(&q)

		require.NotNil(t, sp, "expected `sp` to be non-nil, but it was `nil`")
		assert.Equal(t, "3.14159", *sp)
	})

	t.Run("empty_quantity_returns_pointer_to_empty_string", func(t *testing.T) {
		q := Quantity_None
		sp := QuantityPtrToStringPtr(&q)

		require.NotNil(t, sp, "expected `sp` to be non-nil, but it was `nil`")
		assert.Equal(t, "", *sp)
	})
}

func Test_Quantity_ptr_helpers_roundTrip_string_pointer(t *testing.T) {
	original := "88.125"
	qp := QuantityPtrFromStringPtrUnvalidated(&original)
	sp := QuantityPtrToStringPtr(qp)

	require.NotNil(t, sp, "expected `sp` to be non-nil, but it was `nil`")
	assert.Equal(t, original, *sp)
}

func Test_Quantity_ptr_helpers_roundTrip_value_through_unvalidated_ptr_ctor(t *testing.T) {
	q := QuantityFromStringUnvalidated("0.001")
	p := QuantityPtrFromStringUnvalidated(string(q))
	sp := QuantityPtrToStringPtr(p)

	require.NotNil(t, sp, "expected `sp` to be non-nil, but it was `nil`")
	assert.Equal(t, "0.001", *sp)
}
