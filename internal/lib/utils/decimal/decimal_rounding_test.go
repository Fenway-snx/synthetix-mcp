package decimal

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_ConvertDecimalToFixed_ROUNDING_ISSUE(t *testing.T) {
	// This test demonstrates why rounding might be problematic

	// Example: User has 1.23456789 BTC and we use exponent 8
	// Expected: 123456789 (just shifting decimal 8 places)
	// But what if they have more precision?

	tests := []struct {
		name        string
		input       string // Using string to control exact precision
		exponent    int64
		expected    uint64
		description string
	}{
		{
			name:        "exact precision matches exponent",
			input:       "1.23456789",
			exponent:    3,
			expected:    1234,
			description: "Should convert exactly",
		},
		{
			name:        "more precision than exponent",
			input:       "1.234567894", // Extra digit: 4
			exponent:    8,
			expected:    123456789, // Should truncate, not round
			description: "With rounding, this would become 123456789 (4 < 5, rounds down)",
		},
		{
			name:        "more precision with rounding up case",
			input:       "1.234567895", // Extra digit: 5
			exponent:    8,
			expected:    123456789, // Should truncate to 123456789
			description: "With rounding, this would incorrectly become 123456790",
		},
		{
			name:        "another rounding up case",
			input:       "1.234567899", // Extra digit: 9
			exponent:    8,
			expected:    123456789, // Should truncate to 123456789
			description: "With rounding, this would incorrectly become 123456790",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input, err := shopspring_decimal.NewFromString(tc.input)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			result, err := ConvertDecimalToFixed(input, tc.exponent)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// This will fail for the rounding cases if rounding is applied
			if result != tc.expected {
				t.Errorf("%s: expected %d, got %d. %s",
					tc.name, tc.expected, result, tc.description,
				)
			}
		})
	}
}

func Test_ConvertDecimalToFixed_VALIDATION_LOGIC(t *testing.T) {
	// Test to show the relationship with validation logic
	// If user provides "1.234567895" for a market with exponent 8,
	// validation should reject it (too many decimals)
	// But if they provide "1.23456789", it should be accepted

	// The conversion function should NOT round because:
	// 1. If validation passed, the value has at most 'exponent' decimal places
	// 2. Multiplying by 10^exponent will give an exact integer
	// 3. No rounding needed

	marketExponent := int64(8)

	// Valid input - should convert exactly
	validInput, _ := shopspring_decimal.NewFromString("1.23456789")
	result, err := ConvertDecimalToFixed(validInput, marketExponent)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, uint64(123456789), result)

	// What happens with truncate vs round?
	// Let's create our own version without rounding
	withoutRounding := func(d shopspring_decimal.Decimal, exp int64) uint64 {
		multiplier := shopspring_decimal.New(1, int32(exp))
		result := d.Mul(multiplier)
		// Just truncate to integer (remove decimals)
		return result.BigInt().Uint64()
	}

	// Test case that shows the difference
	problematicInput, _ := shopspring_decimal.NewFromString("1.234567895")

	// Current implementation with rounding
	withRound, _ := ConvertDecimalToFixed(problematicInput, marketExponent)

	// Without rounding
	withoutRound := withoutRounding(problematicInput, marketExponent)

	t.Logf("Input: %s", problematicInput)
	t.Logf("With rounding: %d", withRound)
	t.Logf("Without rounding: %d", withoutRound)

	// These should now be the same since we use truncation
	assert.Equal(t, withRound, withoutRound,
		"Truncation should give the same result as our manual implementation",
	)
}
