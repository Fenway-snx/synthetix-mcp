package types

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_MaintenanceMarginTier_STRUCTURE(t *testing.T) {
	t.Run("MaintenanceMarginTier structure validation", func(t *testing.T) {
		tier := MaintenanceMarginTier{
			MinPositionSize:              "0",
			MaxPositionSize:              "1000000",
			MaxLeverage:                  100,
			InitialMarginRequirement:     shopspring_decimal.NewFromFloat(0.01),
			MaintenanceMarginRequirement: shopspring_decimal.NewFromFloat(0.005),
		}

		assert.Equal(t, "0", tier.MinPositionSize)
		assert.Equal(t, "1000000", tier.MaxPositionSize)
		assert.Equal(t, uint32(100), tier.MaxLeverage)
		assert.True(t, tier.InitialMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.01)))
		assert.True(t, tier.MaintenanceMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.005)))
	})

	t.Run("unlimited max position size", func(t *testing.T) {
		tier := MaintenanceMarginTier{
			MinPositionSize:              "1000000",
			MaxPositionSize:              "", // Empty string for unlimited
			MaxLeverage:                  10,
			InitialMarginRequirement:     shopspring_decimal.NewFromFloat(0.1),
			MaintenanceMarginRequirement: shopspring_decimal.NewFromFloat(0.05),
		}

		assert.Equal(t, "", tier.MaxPositionSize, "Empty string indicates unlimited position size")
	})
}
