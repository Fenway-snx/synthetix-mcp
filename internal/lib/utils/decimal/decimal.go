package decimal

import (
	"errors"
	"fmt"
	"math"
	"sort"

	shopspring_decimal "github.com/shopspring/decimal"
)

// NOTE: because we have inconsistencies in the types used to represent
// exponents, several of the function defined in this file are generic
// around several integer types, as explained below:
//
//  1. for simplicity, we wish to express exponents as `int64`;
//  2. the decimal library we use - **shopspring.decimal** - requires its
//     exponents to be `int32`;
//  3. our database _currently_ records exponents as `uint32`;
//  4. our problem-domain should only have exponents in the range 0-255;

var (
	errCannotComputeInverseOfZero             = errors.New("cannot compute inverse of zero")
	errDivisionByZeroInMulDiv                 = errors.New("division by zero in MulDiv")
	errExponentOutOfRange                     = errors.New("exponent out of range")
	errNegativeValueCannotBeConvertedToUint64 = errors.New("negative value cannot be converted to uint64")
	errPrecisionExceeded                      = errors.New("precision exceeded")
	errValueTooLargeForUint64                 = errors.New("value too large for uint64")
)

const (
	validExponentMinimumValue = -256
	validExponentMaximumValue = 255
)

// Database decimal(20,8) constraints
// Max value: 999,999,999,999.99999999
// Min value: -999,999,999,999.99999999
var (
	MaxDatabaseDecimal = shopspring_decimal.RequireFromString("999999999999.99999999")
	MinDatabaseDecimal = shopspring_decimal.RequireFromString("-999999999999.99999999")
)

// CapForDatabase caps decimal values to fit within the database decimal(20,8) constraint.
// Returns the original value if within bounds, otherwise returns the capped value and true.
func CapForDatabase(value shopspring_decimal.Decimal) (shopspring_decimal.Decimal, bool) {
	if value.GreaterThan(MaxDatabaseDecimal) {
		return MaxDatabaseDecimal, true
	}
	if value.LessThan(MinDatabaseDecimal) {
		return MinDatabaseDecimal, true
	}
	return value, false
}

func _validateExponentValue[E uint8 | uint32 | int32 | int64 | int](exponent any) (int32, int64, error) {

	var exponent32 int32
	var exponent64 int64

	switch i := exponent.(type) {
	case uint8:

		exponent64 = int64(i)
	case uint32:

		exponent64 = int64(i)
	case int32:

		exponent64 = int64(i)
	case int64:

		exponent64 = i
	case int:

		exponent64 = int64(i)
	default:

		return math.MinInt32, math.MinInt64, fmt.Errorf("VIOLATION: exponent has unexpected type '%[1]T' (value=%[1]v)", exponent)
	}

	if exponent64 < validExponentMinimumValue || exponent64 > validExponentMaximumValue {
		return math.MinInt32, math.MinInt64, errExponentOutOfRange
	} else {
		exponent32 = int32(exponent64)

		return exponent32, exponent64, nil
	}
}

// Converts a shopspring.Decimal to uint64 with the specified exponent
// This is used when sending values to the matching service
// Example: Decimal{45000.50} with exponent 2 → 4500050
//
// IMPORTANT: This function uses truncation, not rounding. If the input has more
// decimal places than the exponent allows, the extra digits are truncated.
// This is intentional because:
// 1. Input validation should ensure values have at most 'exponent' decimal places
// 2. After validation, multiplying by 10^exponent gives an exact integer
// 3. Truncation preserves the exact validated value without unexpected changes
func ConvertDecimalToFixed[E int | int32 | int64 | uint8 | uint32](d shopspring_decimal.Decimal, exponent E) (uint64, error) {
	if exponent32, _, err := _validateExponentValue[E](exponent); err != nil {
		return 0, err
	} else {
		// Multiply by 10^exponent
		multiplier := shopspring_decimal.New(1, exponent32)
		result := d.Mul(multiplier)

		// Truncate to integer (no rounding)
		// This preserves the exact value after shifting the decimal point
		result = result.Truncate(0)

		// Check if result fits in uint64
		if result.Sign() < 0 {
			return 0, errNegativeValueCannotBeConvertedToUint64
		}

		// Convert to big.Int for safe conversion
		bigInt := result.BigInt()
		if !bigInt.IsUint64() {
			return 0, errValueTooLargeForUint64
		}

		return bigInt.Uint64(), nil
	}
}

// Creates a shopspring.Decimal from uint64 with the specified exponent
// This is used when receiving values from the matching service
// Example: 4500050 with exponent 2 → Decimal{45000.50}
func CreateDecimalFromFixed[E int | int32 | int64 | uint8 | uint32](value uint64, exponent E) shopspring_decimal.Decimal {
	if exponent32, _, err := _validateExponentValue[E](exponent); err != nil {
		return shopspring_decimal.Zero //, err
	} else {
		// Use NewFromString to handle uint64 values that exceed int64 max
		// This avoids overflow when value > math.MaxInt64
		d, err := shopspring_decimal.NewFromString(fmt.Sprintf("%d", value))
		if err != nil {
			// This should never happen with a valid uint64, but handle it gracefully
			// Fall back to zero to avoid panic
			return shopspring_decimal.Zero
		}
		return d.Shift(-exponent32)
	}
}

// Converts a string representation to uint64 with the specified exponent
// This is used when reading human-readable values from the database
// Example: "45000.50" with exponent 2 → 4500050
func ConvertStringToFixed[E int | int32 | int64 | uint8 | uint32](s string, exponent E) (uint64, error) {
	// Parse string to decimal
	d, err := shopspring_decimal.NewFromString(s)
	if err != nil {
		return 0, fmt.Errorf("failed to parse string to decimal: %w", err)
	}

	// Use existing ConvertDecimalToFixed
	return ConvertDecimalToFixed[E](d, exponent)
}

// Returns the inverse (reciprocal) of the given decimal value
// If the input value is zero, returns an error to avoid division by zero
func Inverse(value shopspring_decimal.Decimal) (shopspring_decimal.Decimal, error) {
	if value.IsZero() {
		return value, errCannotComputeInverseOfZero
	} else {
		return shopspring_decimal.NewFromInt(1).Div(value), nil
	}
}

// Returns the inverse (reciprocal) of the given decimal value
// If the input value is zero, it panics
func InverseUnvalidated(value shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	return shopspring_decimal.NewFromInt(1).Div(value)
}

// Returns the inverse (reciprocal) of the given decimal value
// If the input value is zero, returns zero instead of an error
func InverseOrZero(value shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	if inv, err := Inverse(value); err != nil {
		return shopspring_decimal.Zero
	} else {
		return inv
	}
}

func Mean2(value0, value1 shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	return value0.Add(value1).Div(shopspring_decimal.NewFromInt(2))
}

// Median calculates the median of a slice of decimals
// Returns zero if the slice is empty
// For even number of elements, returns the average of the two middle values
func Median(values ...shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	if len(values) == 0 {
		return shopspring_decimal.Zero
	}

	if len(values) == 1 {
		return values[0]
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]shopspring_decimal.Decimal, len(values))
	copy(sorted, values)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LessThan(sorted[j])
	})

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		// Even number of elements: return average of two middle values
		return Mean2(sorted[mid-1], sorted[mid])
	}
	// Odd number of elements: return middle value
	return sorted[mid]
}

// Multiplies a and b, then divides by c
// Returns an error if division by zero occurs
func MulDiv(a, b, c shopspring_decimal.Decimal) (shopspring_decimal.Decimal, error) {
	if c.IsZero() {
		return c, errDivisionByZeroInMulDiv
	} else {
		return a.Mul(b).Div(c), nil
	}
}

// Multiplies a and b, then divides by c
// If an error occurs (e.g., division by zero), returns zero
func MulDivOrZero(a, b, c shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	if result, err := MulDiv(a, b, c); err != nil {
		return shopspring_decimal.Zero
	} else {
		return result
	}
}

// Multiplies a and b, then divides by c. Only use non-zero values for c.
// Panics if division by zero occurs
func MulDivUnvalidated(a, b, c shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	return a.Mul(b).Div(c)
}

// Parses a decimal string and enforces exponent rules:
// - Accept when decimals <= maxPlaces
// - Accept when excess decimals beyond maxPlaces are zeros (numerically equal after truncation)
// - Reject when excess decimals contain non-zero digits (returns errPrecisionExceeded)
// Returns parsed value and error. Empty string returns (zero, nil).
func ParseAndValidateDecimal[E int | int32 | int64 | uint8 | uint32](valStr string, maxPlaces E) (shopspring_decimal.Decimal, error) {
	if maxPlaces32, _, err := _validateExponentValue[E](maxPlaces); err != nil {
		return shopspring_decimal.Zero, err
	} else {
		if val, err := shopspring_decimal.NewFromString(valStr); err != nil {
			return shopspring_decimal.Zero, fmt.Errorf("invalid decimal: %w", err)
		} else {
			if val.Truncate(maxPlaces32).Cmp(val) != 0 {
				return shopspring_decimal.Zero, fmt.Errorf("%w: has more than %d significant decimal places", errPrecisionExceeded, maxPlaces)
			} else {
				return val, nil
			}
		}
	}
}
