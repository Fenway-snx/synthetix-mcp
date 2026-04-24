package decimal

import (
	"fmt"
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_ConvertDecimalToFixed(t *testing.T) {
	tests := []struct {
		name        string
		input       shopspring_decimal.Decimal
		exponent    int64
		expected    uint64
		expectError bool
		errorMsg    string
	}{
		// Basic conversions with different exponents
		{
			name:     "zero value",
			input:    shopspring_decimal.Zero,
			exponent: 8,
			expected: 0,
		},
		{
			name:     "one with exponent 0",
			input:    shopspring_decimal.NewFromInt(1),
			exponent: 0,
			expected: 1,
		},
		{
			name:     "one with exponent 8",
			input:    shopspring_decimal.NewFromInt(1),
			exponent: 8,
			expected: 100000000,
		},
		{
			name:     "price with 2 decimals (e.g., USD cents)",
			input:    shopspring_decimal.NewFromFloat(45000.50),
			exponent: 2,
			expected: 4500050,
		},
		{
			name:     "price with 8 decimals (e.g., BTC sats)",
			input:    shopspring_decimal.NewFromFloat(45000.12345678),
			exponent: 8,
			expected: 4500012345678,
		},
		{
			name:     "small quantity with high precision",
			input:    shopspring_decimal.NewFromFloat(0.00000001),
			exponent: 8,
			expected: 1,
		},
		{
			name:     "truncating down",
			input:    shopspring_decimal.NewFromFloat(1.234),
			exponent: 2,
			expected: 123, // 1.234 * 100 = 123.4, truncates to 123
		},
		{
			name:     "truncating at .5",
			input:    shopspring_decimal.NewFromFloat(1.235),
			exponent: 2,
			expected: 123, // 1.235 * 100 = 123.5, truncates to 123 (not 124)
		},
		{
			name:     "truncating at .9",
			input:    shopspring_decimal.NewFromFloat(1.239),
			exponent: 2,
			expected: 123, // 1.239 * 100 = 123.9, truncates to 123
		},
		{
			name:     "large value",
			input:    shopspring_decimal.NewFromFloat(123456789.123456),
			exponent: 6,
			expected: 123456789123456,
		},

		// Error cases
		{
			name:        "negative value",
			input:       shopspring_decimal.NewFromInt(-1),
			exponent:    8,
			expectError: true,
			errorMsg:    "negative value cannot be converted to uint64",
		},
		{
			name:        "value too large for uint64",
			input:       shopspring_decimal.NewFromFloat(1e20), // Very large number
			exponent:    8,
			expectError: true,
			errorMsg:    "value too large for uint64",
		},
		{
			name:        "overflow with high exponent",
			input:       shopspring_decimal.NewFromFloat(1e10),
			exponent:    10, // Would result in 1e20, too large
			expectError: true,
			errorMsg:    "value too large for uint64",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertDecimalToFixed(tc.input, tc.exponent)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func Test_CreateDecimalFromFixed(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		exponent int64
		expected shopspring_decimal.Decimal
	}{
		{
			name:     "zero value",
			value:    0,
			exponent: 8,
			expected: shopspring_decimal.Zero,
		},
		{
			name:     "one with exponent 0",
			value:    1,
			exponent: 0,
			expected: shopspring_decimal.NewFromInt(1),
		},
		{
			name:     "convert from sats to BTC",
			value:    100000000,
			exponent: 8,
			expected: shopspring_decimal.NewFromInt(1),
		},
		{
			name:     "price in cents to dollars",
			value:    4500050,
			exponent: 2,
			expected: shopspring_decimal.NewFromFloat(45000.50),
		},
		{
			name:     "high precision conversion",
			value:    4500012345678,
			exponent: 8,
			expected: shopspring_decimal.NewFromFloat(45000.12345678),
		},
		{
			name:     "small value with high exponent",
			value:    1,
			exponent: 8,
			expected: shopspring_decimal.NewFromFloat(0.00000001),
		},
		{
			name:     "large value with exponent",
			value:    123456789012345,
			exponent: 6,
			expected: shopspring_decimal.NewFromFloat(123456789.012345),
		},
		{
			name:     "typical price exponent 2",
			value:    123456,
			exponent: 2,
			expected: shopspring_decimal.NewFromFloat(1234.56),
		},
		{
			name:     "typical quantity exponent 8",
			value:    123456789,
			exponent: 8,
			expected: shopspring_decimal.NewFromFloat(1.23456789),
		},
		{
			name:     "negative exponent (shifting right)",
			value:    100,
			exponent: -2,
			expected: shopspring_decimal.NewFromInt(10000), // 100 * 10^2
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CreateDecimalFromFixed(tc.value, tc.exponent)
			assert.True(t, tc.expected.Equal(result),
				"expected %s, got '%s'", tc.expected.String(), result.String())
		})
	}
}

func Test_RoundTripConversion(t *testing.T) {
	// Test that converting to fixed and back gives the same value
	tests := []struct {
		name     string
		original shopspring_decimal.Decimal
		exponent int64
	}{
		{
			name:     "simple integer",
			original: shopspring_decimal.NewFromInt(42),
			exponent: 8,
		},
		{
			name:     "decimal with exact precision",
			original: shopspring_decimal.NewFromFloat(123.45),
			exponent: 2,
		},
		{
			name:     "high precision decimal",
			original: shopspring_decimal.NewFromFloat(123.12345678),
			exponent: 8,
		},
		{
			name:     "very small decimal",
			original: shopspring_decimal.NewFromFloat(0.00000001),
			exponent: 8,
		},
		{
			name:     "large decimal",
			original: shopspring_decimal.NewFromFloat(999999.99999999),
			exponent: 8,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Convert to fixed
			fixed, err := ConvertDecimalToFixed(tc.original, tc.exponent)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// Convert back to decimal
			result := CreateDecimalFromFixed(fixed, tc.exponent)

			// For round trip, we need to account for precision loss
			// Truncate original to the exponent precision for comparison
			expected := tc.original.Truncate(int32(tc.exponent))
			assert.True(t, expected.Equal(result),
				"expected %s, got '%s'", expected.String(), result.String())
		})
	}
}

func Test_ConvertDecimalToFixed_WITH_VARIOUS_EXPONENTS(t *testing.T) {
	// Test the same value with different exponents
	value := shopspring_decimal.NewFromFloat(123.456789)

	tests := []struct {
		exponent int64
		expected uint64
	}{
		{0, 123},         // No decimals (truncates)
		{1, 1234},        // 1 decimal place (truncates, not rounds)
		{2, 12345},       // 2 decimal places (truncates)
		{3, 123456},      // 3 decimal places (truncates)
		{4, 1234567},     // 4 decimal places (truncates)
		{5, 12345678},    // 5 decimal places (truncates)
		{6, 123456789},   // 6 decimal places (exact)
		{7, 1234567890},  // 7 decimal places
		{8, 12345678900}, // 8 decimal places
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("exponent_%d", tc.exponent), func(t *testing.T) {
			result, err := ConvertDecimalToFixed(value, tc.exponent)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_CreateDecimalFromFixed_PRECISION(t *testing.T) {
	// Test that we maintain precision correctly
	tests := []struct {
		name     string
		value    uint64
		exponent int64
		checkStr string // Expected string representation
	}{
		{
			name:     "no trailing zeros",
			value:    12345,
			exponent: 2,
			checkStr: "123.45",
		},
		{
			name:     "with trailing zeros",
			value:    12300,
			exponent: 2,
			checkStr: "123",
		},
		{
			name:     "high precision no trailing zeros",
			value:    12345678,
			exponent: 8,
			checkStr: "0.12345678",
		},
		{
			name:     "single decimal place",
			value:    1,
			exponent: 1,
			checkStr: "0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CreateDecimalFromFixed(tc.value, tc.exponent)
			assert.Equal(t, tc.checkStr, result.String())
		})
	}
}

func Benchmark_ConvertDecimalToFixed(b *testing.B) {
	d := shopspring_decimal.NewFromFloat(12345.6789)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertDecimalToFixed(d, 8)
	}
}

func Benchmark_CreateDecimalFromFixed(b *testing.B) {
	value := uint64(123456789)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CreateDecimalFromFixed(value, 8)
	}
}

func Test_Inverse(t *testing.T) {
	tests := []struct {
		name     string
		input    shopspring_decimal.Decimal
		expected shopspring_decimal.Decimal
	}{
		{
			name:     "inverse of 2",
			input:    shopspring_decimal.NewFromInt(2),
			expected: shopspring_decimal.NewFromFloat(0.5),
		},
		{
			name:     "inverse of 0.5",
			input:    shopspring_decimal.NewFromFloat(0.5),
			expected: shopspring_decimal.NewFromInt(2),
		},
		{
			name:     "inverse of 1",
			input:    shopspring_decimal.NewFromInt(1),
			expected: shopspring_decimal.NewFromInt(1),
		},
		{
			name:     "inverse of 10",
			input:    shopspring_decimal.NewFromInt(10),
			expected: shopspring_decimal.NewFromFloat(0.1),
		},
		{
			name:     "inverse of 12.5",
			input:    shopspring_decimal.NewFromFloat(12.5),
			expected: shopspring_decimal.NewFromFloat(0.08),
		},
		{
			name:     "inverse of 0.1",
			input:    shopspring_decimal.NewFromFloat(0.1),
			expected: shopspring_decimal.NewFromInt(10),
		},
		{
			name:     "inverse of 1000",
			input:    shopspring_decimal.NewFromInt(1000),
			expected: shopspring_decimal.NewFromFloat(0.001),
		},
		{
			name:     "inverse of -0.25",
			input:    shopspring_decimal.NewFromFloat(-0.25),
			expected: shopspring_decimal.NewFromInt(-4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Inverse(tt.input)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Unvalidated", func(t *testing.T) {
			result := InverseUnvalidated(tt.input)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}

	for _, tt := range tests {
		t.Run(tt.name+"_OrZero", func(t *testing.T) {
			result := InverseOrZero(tt.input)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}

	// Dividion by zero cases
	t.Run("inverse of zero returns error", func(t *testing.T) {
		_, err := Inverse(shopspring_decimal.Zero)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot compute inverse of zero")
	})

	t.Run("inverse unvalidated of zero panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				// We expect a panic here due to division by zero
				assert.Contains(t, fmt.Sprint(r), "division by 0")
			} else {
				t.Errorf("expected panic due to division by zero, but did not panic")
			}
		}()
		InverseUnvalidated(shopspring_decimal.Zero)
	})

	t.Run("inverse or zero of zero returns zero", func(t *testing.T) {
		result := InverseOrZero(shopspring_decimal.Zero)
		assert.True(t, result.Equal(shopspring_decimal.Zero),
			"expected 0, got '%s'", result.String())
	})
}

func Benchmark_Inverse(b *testing.B) {
	value := shopspring_decimal.NewFromFloat(12345.6789)
	for b.Loop() {
		_, _ = Inverse(value)
	}
}

func Benchmark_InverseUnvalidated(b *testing.B) {
	value := shopspring_decimal.NewFromFloat(12345.6789)
	for b.Loop() {
		_ = InverseUnvalidated(value)
	}
}

func Benchmark_InverseOrZero(b *testing.B) {
	value := shopspring_decimal.NewFromFloat(12345.6789)
	for b.Loop() {
		_ = InverseOrZero(value)
	}
}

func Test_MulDiv(t *testing.T) {
	tests := []struct {
		name     string
		a        shopspring_decimal.Decimal
		b        shopspring_decimal.Decimal
		c        shopspring_decimal.Decimal
		expected shopspring_decimal.Decimal
	}{
		{
			name:     "normal case",
			a:        shopspring_decimal.NewFromInt(10),
			b:        shopspring_decimal.NewFromInt(20),
			c:        shopspring_decimal.NewFromInt(5),
			expected: shopspring_decimal.NewFromInt(40), // (10 * 20) / 5 = 40
		},
		{
			name:     "a and b are zero",
			a:        shopspring_decimal.Zero,
			b:        shopspring_decimal.Zero,
			c:        shopspring_decimal.NewFromInt(5),
			expected: shopspring_decimal.Zero, // (0 * 0) / 5 = 0
		},
		{
			name:     "large numbers",
			a:        shopspring_decimal.NewFromInt(1_000_000),
			b:        shopspring_decimal.NewFromInt(2_000_000),
			c:        shopspring_decimal.NewFromInt(500_000),
			expected: shopspring_decimal.NewFromInt(4_000_000), // (1M * 2M) / 500K = 4M
		},
		{
			name:     "small numbers",
			a:        shopspring_decimal.NewFromFloat(0.1),
			b:        shopspring_decimal.NewFromFloat(0.2),
			c:        shopspring_decimal.NewFromFloat(0.5),
			expected: shopspring_decimal.NewFromFloat(0.04), // (0.1 * 0.2) / 0.5 = 0.04
		},
		{
			name:     "negative numbers",
			a:        shopspring_decimal.NewFromInt(-10),
			b:        shopspring_decimal.NewFromInt(-20),
			c:        shopspring_decimal.NewFromInt(-50),
			expected: shopspring_decimal.NewFromInt(-4), // (-10 * -20) / -50 = -4
		},
		{
			name:     "large floats",
			a:        shopspring_decimal.NewFromFloat(1.23456789),
			b:        shopspring_decimal.NewFromFloat(2.34567891),
			c:        shopspring_decimal.NewFromFloat(3.45678912),
			expected: shopspring_decimal.NewFromFloat(0.8377427034184254), // (1.23456789 * 2.34567891) / 3.45678912 = 0.8377427034184254
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MulDiv(tt.a, tt.b, tt.c)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Unvalidated", func(t *testing.T) {
			result := MulDivUnvalidated(tt.a, tt.b, tt.c)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}

	for _, tt := range tests {
		t.Run(tt.name+"_OrZero", func(t *testing.T) {
			result := MulDivOrZero(tt.a, tt.b, tt.c)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}

	// Division by zero cases

	t.Run("division by zero returns zero", func(t *testing.T) {
		result, err := MulDiv(shopspring_decimal.NewFromInt(10), shopspring_decimal.NewFromInt(20), shopspring_decimal.Zero)
		assert.Error(t, err)
		assert.True(t, result.Equal(shopspring_decimal.Zero))
	})

	t.Run("division by zero unvalidated panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				// We expect a panic here due to division by zero
				assert.Contains(t, fmt.Sprint(r), "division by 0")
			} else {
				t.Errorf("expected panic due to division by zero, but did not panic")
			}
		}()
		MulDivUnvalidated(shopspring_decimal.NewFromInt(10), shopspring_decimal.NewFromInt(20), shopspring_decimal.Zero)
	})

	t.Run("division by zero or zero returns zero", func(t *testing.T) {
		result := MulDivOrZero(shopspring_decimal.NewFromInt(10), shopspring_decimal.NewFromInt(20), shopspring_decimal.Zero)
		assert.True(t, result.Equal(shopspring_decimal.Zero))
	})
}

func Benchmark_MulDiv(b *testing.B) {
	a := shopspring_decimal.NewFromFloat(12345.6789)
	bb := shopspring_decimal.NewFromFloat(98765.4321)
	c := shopspring_decimal.NewFromFloat(100.0)
	for b.Loop() {
		_, _ = MulDiv(a, bb, c)
	}
}

func Benchmark_MulDivUnvalidated(b *testing.B) {
	a := shopspring_decimal.NewFromFloat(12345.6789)
	bb := shopspring_decimal.NewFromFloat(98765.4321)
	c := shopspring_decimal.NewFromFloat(100.0)
	for b.Loop() {
		_ = MulDivUnvalidated(a, bb, c)
	}
}

func Benchmark_MulDivOrZero(b *testing.B) {
	a := shopspring_decimal.NewFromFloat(12345.6789)
	bb := shopspring_decimal.NewFromFloat(98765.4321)
	c := shopspring_decimal.NewFromFloat(100.0)
	for b.Loop() {
		_ = MulDivOrZero(a, bb, c)
	}
}

func Test_Mean2(t *testing.T) {
	tests := []struct {
		name     string
		value0   shopspring_decimal.Decimal
		value1   shopspring_decimal.Decimal
		expected shopspring_decimal.Decimal
	}{
		{
			name:     "both 0",
			value0:   shopspring_decimal.NewFromInt(0),
			value1:   shopspring_decimal.NewFromInt(0),
			expected: shopspring_decimal.NewFromInt(0),
		},
		{
			name:     "both same (+ve)",
			value0:   shopspring_decimal.NewFromInt(123_456_789),
			value1:   shopspring_decimal.NewFromInt(123_456_789),
			expected: shopspring_decimal.NewFromInt(123_456_789),
		},
		{
			name:     "both same (-ve)",
			value0:   shopspring_decimal.NewFromInt(-123_456_789),
			value1:   shopspring_decimal.NewFromInt(-123_456_789),
			expected: shopspring_decimal.NewFromInt(-123_456_789),
		},
		{
			name:     "spread of +10",
			value0:   shopspring_decimal.NewFromInt(45000),
			value1:   shopspring_decimal.NewFromInt(45010),
			expected: shopspring_decimal.NewFromInt(45005),
		},
		{
			name:     "spread of -10",
			value0:   shopspring_decimal.NewFromInt(-45000),
			value1:   shopspring_decimal.NewFromInt(-45010),
			expected: shopspring_decimal.NewFromInt(-45005),
		},
		{
			name:     "spread of +17",
			value0:   shopspring_decimal.NewFromInt(45000),
			value1:   shopspring_decimal.NewFromInt(45017),
			expected: shopspring_decimal.New(450085, -1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Mean2(tt.value0, tt.value1)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}
}

func Test_Median(t *testing.T) {
	tests := []struct {
		name     string
		values   []shopspring_decimal.Decimal
		expected shopspring_decimal.Decimal
	}{
		{
			name:     "empty slice",
			values:   []shopspring_decimal.Decimal{},
			expected: shopspring_decimal.Zero,
		},
		{
			name:     "single value",
			values:   []shopspring_decimal.Decimal{shopspring_decimal.NewFromInt(45000)},
			expected: shopspring_decimal.NewFromInt(45000),
		},
		{
			name: "three values - normal market",
			values: []shopspring_decimal.Decimal{
				shopspring_decimal.NewFromInt(45000), // best bid
				shopspring_decimal.NewFromInt(45010), // best ask
				shopspring_decimal.NewFromInt(45005), // last trade
			},
			expected: shopspring_decimal.NewFromInt(45005),
		},
		{
			name: "three values - unsorted",
			values: []shopspring_decimal.Decimal{
				shopspring_decimal.NewFromInt(45010),
				shopspring_decimal.NewFromInt(45000),
				shopspring_decimal.NewFromInt(45005),
			},
			expected: shopspring_decimal.NewFromInt(45005),
		},
		{
			name: "two values - one-sided book",
			values: []shopspring_decimal.Decimal{
				shopspring_decimal.NewFromInt(45000),
				shopspring_decimal.NewFromInt(45005),
			},
			expected: shopspring_decimal.NewFromFloat(45002.5),
		},
		{
			name: "four values - even number",
			values: []shopspring_decimal.Decimal{
				shopspring_decimal.NewFromInt(10),
				shopspring_decimal.NewFromInt(20),
				shopspring_decimal.NewFromInt(30),
				shopspring_decimal.NewFromInt(40),
			},
			expected: shopspring_decimal.NewFromInt(25), // (20 + 30) / 2
		},
		{
			name: "five values - odd number",
			values: []shopspring_decimal.Decimal{
				shopspring_decimal.NewFromInt(10),
				shopspring_decimal.NewFromInt(20),
				shopspring_decimal.NewFromInt(30),
				shopspring_decimal.NewFromInt(40),
				shopspring_decimal.NewFromInt(50),
			},
			expected: shopspring_decimal.NewFromInt(30),
		},
		{
			name: "decimal values",
			values: []shopspring_decimal.Decimal{
				shopspring_decimal.RequireFromString("45000.50"),
				shopspring_decimal.RequireFromString("45010.75"),
				shopspring_decimal.RequireFromString("45005.25"),
			},
			expected: shopspring_decimal.RequireFromString("45005.25"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Median(tt.values...)
			assert.True(t, result.Equal(tt.expected),
				"expected %s, got '%s'", tt.expected.String(), result.String())
		})
	}
}
