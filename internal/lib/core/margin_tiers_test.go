package core

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

func Test_GetMaintenanceMarginRate(t *testing.T) {
	tests := []struct {
		name        string
		maxLeverage uint8
		expected    string
	}{
		{
			name:        "100x leverage",
			maxLeverage: 100,
			expected:    "0.005", // 1/(2*100) = 0.005
		},
		{
			name:        "50x leverage",
			maxLeverage: 50,
			expected:    "0.01", // 1/(2*50) = 0.01
		},
		{
			name:        "25x leverage",
			maxLeverage: 25,
			expected:    "0.02", // 1/(2*25) = 0.02
		},
		{
			name:        "10x leverage",
			maxLeverage: 10,
			expected:    "0.05", // 1/(2*10) = 0.05
		},
		{
			name:        "2x leverage",
			maxLeverage: 2,
			expected:    "0.25", // 1/(2*2) = 0.25
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier := &postgrestypes.MaintenanceMarginTier{
				MaxLeverage: tt.maxLeverage,
			}
			result := tier.GetMaintenanceMarginRate()
			expected, err := shopspring_decimal.NewFromString(tt.expected)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.True(t, result.Equal(expected), "Expected %s, got '%s'", expected.String(), result.String())
		})
	}
}

func Test_GetInitialMarginRate(t *testing.T) {
	tests := []struct {
		name        string
		maxLeverage uint8
		expected    string
	}{
		{
			name:        "100x leverage",
			maxLeverage: 100,
			expected:    "0.01", // 2 * 0.005 = 0.01
		},
		{
			name:        "50x leverage",
			maxLeverage: 50,
			expected:    "0.02", // 2 * 0.01 = 0.02
		},
		{
			name:        "25x leverage",
			maxLeverage: 25,
			expected:    "0.04", // 2 * 0.02 = 0.04
		},
		{
			name:        "10x leverage",
			maxLeverage: 10,
			expected:    "0.1", // 2 * 0.05 = 0.1
		},
		{
			name:        "2x leverage",
			maxLeverage: 2,
			expected:    "0.5", // 2 * 0.25 = 0.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier := &postgrestypes.MaintenanceMarginTier{
				MaxLeverage: tt.maxLeverage,
			}
			result := tier.GetInitialMarginRate()
			expected, err := shopspring_decimal.NewFromString(tt.expected)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.True(t, result.Equal(expected), "Expected %s, got '%s'", expected.String(), result.String())
		})
	}
}

func Test_ValidateMaintenanceMarginTiers_rejects_non_positive_max_leverage(t *testing.T) {
	tiers := []postgrestypes.MaintenanceMarginTier{
		{
			MinPositionSize: shopspring_decimal.Zero,
			MaxPositionSize: nil,
			MaxLeverage:     0,
		},
	}
	err := ValidateMaintenanceMarginTiers(tiers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max leverage must be greater than 0")
}

func Test_GetMaintenanceDeduction(t *testing.T) {
	// Create sample tiers similar to BTC market
	maxPos1 := shopspring_decimal.NewFromInt(1000000)
	maxPos2 := shopspring_decimal.NewFromInt(50000000)
	maxPos3 := shopspring_decimal.NewFromInt(150000000)
	maxPos4 := shopspring_decimal.NewFromInt(300000000)

	tiers := []postgrestypes.MaintenanceMarginTier{
		{
			MinPositionSize:        shopspring_decimal.Zero,
			MaxPositionSize:        &maxPos1,
			MaxLeverage:            100,
			MaintenanceMarginRatio: shopspring_decimal.NewFromFloat(0.005), // 0.5%
		},
		{
			MinPositionSize:        shopspring_decimal.NewFromInt(1000001),
			MaxPositionSize:        &maxPos2,
			MaxLeverage:            50,
			MaintenanceMarginRatio: shopspring_decimal.NewFromFloat(0.01), // 1%
		},
		{
			MinPositionSize:        shopspring_decimal.NewFromInt(50000001),
			MaxPositionSize:        &maxPos3,
			MaxLeverage:            25,
			MaintenanceMarginRatio: shopspring_decimal.NewFromFloat(0.02), // 2%
		},
		{
			MinPositionSize:        shopspring_decimal.NewFromInt(150000001),
			MaxPositionSize:        &maxPos4,
			MaxLeverage:            10,
			MaintenanceMarginRatio: shopspring_decimal.NewFromFloat(0.05), // 5%
		},
		{
			MinPositionSize:        shopspring_decimal.NewFromInt(300000001),
			MaxPositionSize:        nil,
			MaxLeverage:            2,
			MaintenanceMarginRatio: shopspring_decimal.NewFromFloat(0.25), // 25%
		},
	}

	tests := []struct {
		name         string
		tierIndex    int
		expectedCalc string
		expected     string
	}{
		{
			name:         "Tier 0 - no deduction",
			tierIndex:    0,
			expectedCalc: "0",
			expected:     "0",
		},
		{
			name:         "Tier 1 deduction",
			tierIndex:    1,
			expectedCalc: "1,000,001 * (0.01 - 0.005) = 1,000,001 * 0.005",
			expected:     "5000.005",
		},
		{
			name:         "Tier 2 deduction",
			tierIndex:    2,
			expectedCalc: "5,000.005 + 50,000,001 * (0.02 - 0.01) = 5,000.005 + 50,000,001 * 0.01",
			expected:     "505000.015",
		},
		{
			name:         "Tier 3 deduction",
			tierIndex:    3,
			expectedCalc: "505,000.015 + 150,000,001 * (0.05 - 0.02) = 505,000.015 + 150,000,001 * 0.03",
			expected:     "5005000.045",
		},
		{
			name:         "Tier 4 deduction",
			tierIndex:    4,
			expectedCalc: "5,005,000.045 + 300,000,001 * (0.25 - 0.05) = 5,005,000.045 + 300,000,001 * 0.20",
			expected:     "65005000.245",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMaintenanceDeduction(tiers, tt.tierIndex)
			expected, err := shopspring_decimal.NewFromString(tt.expected)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.True(t, result.Equal(expected), "Expected %s, got '%s'", expected.String(), result.String())
			t.Logf("Tier %d deduction calculation: %s = %s", tt.tierIndex, tt.expectedCalc, result.String())
		})
	}
}

func Test_MaintenanceDeductionWithCalculatedRatios(t *testing.T) {
	// Test deduction calculation with rates calculated from max leverage
	maxPos1 := shopspring_decimal.NewFromInt(1000000)
	maxPos2 := shopspring_decimal.NewFromInt(50000000)

	tiers := []postgrestypes.MaintenanceMarginTier{
		{
			MinPositionSize: shopspring_decimal.Zero,
			MaxPositionSize: &maxPos1,
			MaxLeverage:     100,
		},
		{
			MinPositionSize: shopspring_decimal.NewFromInt(1000001),
			MaxPositionSize: &maxPos2,
			MaxLeverage:     50,
		},
		{
			MinPositionSize: shopspring_decimal.NewFromInt(50000001),
			MaxPositionSize: nil,
			MaxLeverage:     25,
		},
	}

	// Calculate rates from max leverage
	for i := range tiers {
		tiers[i].MaintenanceMarginRatio = tiers[i].GetMaintenanceMarginRate()
		tiers[i].InitialMarginRatio = tiers[i].GetInitialMarginRate()
	}

	// Test deduction calculations
	t.Run("Tier 0 - no deduction", func(t *testing.T) {
		deduction := GetMaintenanceDeduction(tiers, 0)
		assert.True(t, deduction.Equal(shopspring_decimal.Zero), "Expected 0, got '%s'", deduction.String())
	})

	t.Run("Tier 1 - with calculated rates", func(t *testing.T) {
		deduction := GetMaintenanceDeduction(tiers, 1)
		// 1,000,001 * (1/(2*50) - 1/(2*100)) = 1,000,001 * (0.01 - 0.005) = 5,000.005
		expected := shopspring_decimal.NewFromFloat(5000.005)
		assert.True(t, deduction.Equal(expected), "Expected %s, got '%s'", expected.String(), deduction.String())
	})

	t.Run("Tier 2 - with calculated rates", func(t *testing.T) {
		deduction := GetMaintenanceDeduction(tiers, 2)
		// 5,000.005 + 50,000,001 * (1/(2*25) - 1/(2*50)) = 5,000.005 + 50,000,001 * (0.02 - 0.01)
		expected := shopspring_decimal.NewFromFloat(505000.015)
		assert.True(t, deduction.Equal(expected), "Expected %s, got '%s'", expected.String(), deduction.String())
	})
}

func Test_CalculateMaintenanceMargin(t *testing.T) {
	// Create sample tiers
	maxPos1 := shopspring_decimal.NewFromInt(1000000)
	maxPos2 := shopspring_decimal.NewFromInt(50000000)

	tiers := []postgrestypes.MaintenanceMarginTier{
		{
			MinPositionSize:           shopspring_decimal.Zero,
			MaxPositionSize:           &maxPos1,
			MaxLeverage:               100,
			MaintenanceMarginRatio:    shopspring_decimal.NewFromFloat(0.005),
			MaintenanceDeductionValue: shopspring_decimal.Zero,
		},
		{
			MinPositionSize:           shopspring_decimal.NewFromInt(1000001),
			MaxPositionSize:           &maxPos2,
			MaxLeverage:               50,
			MaintenanceMarginRatio:    shopspring_decimal.NewFromFloat(0.01),
			MaintenanceDeductionValue: shopspring_decimal.NewFromFloat(5000.005),
		},
		{
			MinPositionSize:           shopspring_decimal.NewFromInt(50000001),
			MaxPositionSize:           nil,
			MaxLeverage:               25,
			MaintenanceMarginRatio:    shopspring_decimal.NewFromFloat(0.02),
			MaintenanceDeductionValue: shopspring_decimal.NewFromFloat(505000.015),
		},
	}

	tests := []struct {
		name           string
		notionalValue  shopspring_decimal.Decimal
		expectedMargin string
		expectedCalc   string
	}{
		{
			name:           "Small position in tier 1",
			notionalValue:  shopspring_decimal.NewFromInt(500000),
			expectedMargin: "2500",
			expectedCalc:   "500,000 * 0.005 - 0 = 2,500",
		},
		{
			name:           "Position in tier 2",
			notionalValue:  shopspring_decimal.NewFromInt(25000000),
			expectedMargin: "244999.995",
			expectedCalc:   "25,000,000 * 0.01 - 5,000.005 = 250,000 - 5,000.005 = 244,999.995",
		},
		{
			name:           "Large position in tier 3",
			notionalValue:  shopspring_decimal.NewFromInt(100000000),
			expectedMargin: "1494999.985",
			expectedCalc:   "100,000,000 * 0.02 - 505,000.015 = 2,000,000 - 505,000.015 = 1,494,999.985",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateMaintenanceMargin(tiers, tt.notionalValue)
			expected, err := shopspring_decimal.NewFromString(tt.expectedMargin)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.True(t, result.Equal(expected), "Expected %s, got '%s'", expected.String(), result.String())
			t.Logf("Calculation: %s", tt.expectedCalc)
		})
	}
}

func Test_FindApplicableTier(t *testing.T) {
	// Create sample tiers
	maxPos1 := shopspring_decimal.NewFromInt(1000000)
	maxPos2 := shopspring_decimal.NewFromInt(50000000)

	tiers := []postgrestypes.MaintenanceMarginTier{
		{
			MinPositionSize: shopspring_decimal.Zero,
			MaxPositionSize: &maxPos1,
			MaxLeverage:     100,
		},
		{
			MinPositionSize: shopspring_decimal.NewFromInt(1000001),
			MaxPositionSize: &maxPos2,
			MaxLeverage:     50,
		},
		{
			MinPositionSize: shopspring_decimal.NewFromInt(50000001),
			MaxPositionSize: nil,
			MaxLeverage:     25,
		},
	}

	tests := []struct {
		name              string
		notionalValue     shopspring_decimal.Decimal
		expectedTierIndex int
		expectedNil       bool
	}{
		{
			name:              "Position in tier 1",
			notionalValue:     shopspring_decimal.NewFromInt(500000),
			expectedTierIndex: 0,
		},
		{
			name:              "Position at tier 1 boundary",
			notionalValue:     shopspring_decimal.NewFromInt(1000000),
			expectedTierIndex: 0,
		},
		{
			name:              "Position in tier 2",
			notionalValue:     shopspring_decimal.NewFromInt(25000000),
			expectedTierIndex: 1,
		},
		{
			name:              "Position in tier 3",
			notionalValue:     shopspring_decimal.NewFromInt(100000000),
			expectedTierIndex: 2,
		},
		{
			name:              "Very large position in last tier",
			notionalValue:     shopspring_decimal.NewFromInt(1000000000),
			expectedTierIndex: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindApplicableTier(tiers, tt.notionalValue)
			if tt.expectedNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tiers[tt.expectedTierIndex].MaxLeverage, result.MaxLeverage)
			}
		})
	}
}
