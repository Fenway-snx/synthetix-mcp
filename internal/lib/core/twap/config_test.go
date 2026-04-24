package twap_test

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	snx_lib_core_twap "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/twap"
)

func Test_Config_BasicStructure(t *testing.T) {
	t.Run("creates valid TWAP config", func(t *testing.T) {
		cfg := &snx_lib_core_twap.Config{
			ChunkIntervalMs: 10000,
			ChunkQuantity:   shopspring_decimal.NewFromInt(10),
			ChunksTotal:     10,
			TotalQuantity:   shopspring_decimal.NewFromInt(100),
		}

		assert.NotNil(t, cfg)
		assert.True(t, cfg.TotalQuantity.Equal(shopspring_decimal.NewFromInt(100)))
		assert.True(t, cfg.ChunkQuantity.Equal(shopspring_decimal.NewFromInt(10)))
		assert.Equal(t, int64(10000), cfg.ChunkIntervalMs)
		assert.Equal(t, 10, cfg.ChunksTotal)
		assert.True(t, cfg.PriceLimit.IsZero())
	})
}

func Test_Config_DecimalPrecision(t *testing.T) {
	t.Run("handles decimal quantities correctly", func(t *testing.T) {
		totalQty, _ := shopspring_decimal.NewFromString("100.123456")
		chunkQty, _ := shopspring_decimal.NewFromString("10.012345")

		cfg := &snx_lib_core_twap.Config{
			ChunkIntervalMs: 10000,
			ChunkQuantity:   chunkQty,
			ChunksTotal:     10,
			TotalQuantity:   totalQty,
		}

		assert.True(t, cfg.TotalQuantity.Equal(totalQty))
		assert.True(t, cfg.ChunkQuantity.Equal(chunkQty))
	})

	t.Run("preserves decimal precision", func(t *testing.T) {
		qty := shopspring_decimal.NewFromFloat(0.00000001)
		cfg := &snx_lib_core_twap.Config{
			ChunkQuantity: qty,
			ChunksTotal:   1,
			TotalQuantity: qty,
		}

		assert.True(t, cfg.TotalQuantity.Equal(qty))
		assert.False(t, cfg.TotalQuantity.IsZero())
	})
}

func Test_Config_PriceLimit(t *testing.T) {
	t.Run("zero price limit means no limit", func(t *testing.T) {
		cfg := &snx_lib_core_twap.Config{
			ChunkIntervalMs: 10000,
			ChunkQuantity:   shopspring_decimal.NewFromInt(10),
			ChunksTotal:     10,
			PriceLimit:      shopspring_decimal.Zero,
			TotalQuantity:   shopspring_decimal.NewFromInt(100),
		}

		assert.True(t, cfg.PriceLimit.IsZero())
		assert.False(t, cfg.PriceLimit.IsPositive())
	})

	t.Run("positive price limit enables limit order chunks", func(t *testing.T) {
		cfg := &snx_lib_core_twap.Config{
			ChunkIntervalMs: 10000,
			ChunkQuantity:   shopspring_decimal.NewFromInt(10),
			ChunksTotal:     10,
			PriceLimit:      shopspring_decimal.NewFromFloat(49999.99),
			TotalQuantity:   shopspring_decimal.NewFromInt(100),
		}

		assert.True(t, cfg.PriceLimit.IsPositive())
	})
}
