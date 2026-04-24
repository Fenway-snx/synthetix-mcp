package core_test

import (
	"encoding/json"
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

func Test_TWAPExecutionState_WireRoundTrip(t *testing.T) {
	state := &snx_lib_core.TWAPExecutionState{
		OrderId: snx_lib_core.OrderId{
			VenueId:  snx_lib_core.VenueOrderId(42),
			ClientId: snx_lib_core.ClientOrderId("c1"),
		},
		SubAccountID:      7,
		ChunkIntervalMs:   1000,
		ChunkQuantity:     shopspring_decimal.NewFromInt(1),
		ChunksTotal:       5,
		Symbol:            "ETH-USD",
		TotalQuantity:     shopspring_decimal.NewFromInt(5),
		FilledNotional:    shopspring_decimal.Zero,
		QuantityFilled:    shopspring_decimal.Zero,
		QuantitySubmitted: shopspring_decimal.Zero,
		State:             snx_lib_core.OrderStatePlaced,
		TotalFees:         shopspring_decimal.Zero,
	}

	ev := snx_lib_core.OpenOrderEvent{
		SubAccountId:       7,
		TWAPExecutionState: state,
	}

	outer, err := json.Marshal(&ev)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(outer, &raw))
	twapRaw, ok := raw["twap_execution_state"]
	require.True(t, ok)
	// The TWAP state should be a native JSON object (not a quoted string).
	assert.True(t, len(twapRaw) > 0 && twapRaw[0] == '{', "expected JSON object, got: %s", string(twapRaw))

	var decoded snx_lib_core.OpenOrderEvent
	require.NoError(t, json.Unmarshal(outer, &decoded))
	require.True(t, snx_lib_core.TWAPExecutionStateHasPayload(decoded.TWAPExecutionState))
	assert.Equal(t, state.OrderId.VenueId, decoded.TWAPExecutionState.OrderId.VenueId)
	assert.True(t, state.ChunkQuantity.Equal(decoded.TWAPExecutionState.ChunkQuantity))
}

func Test_TWAPExecutionState_AveragePrice(t *testing.T) {
	t.Run("returns zero when nothing filled", func(t *testing.T) {
		xs := &snx_lib_core.TWAPExecutionState{
			FilledNotional: shopspring_decimal.Zero,
			QuantityFilled: shopspring_decimal.Zero,
		}
		assert.True(t, xs.AveragePrice(2).IsZero())
	})

	t.Run("computes weighted average correctly", func(t *testing.T) {
		xs := &snx_lib_core.TWAPExecutionState{
			FilledNotional: shopspring_decimal.NewFromInt(1_050_000),
			QuantityFilled: shopspring_decimal.NewFromInt(20),
		}
		avg := xs.AveragePrice(2)
		assert.True(t, avg.Equal(shopspring_decimal.NewFromInt(52500)),
			"expected 1050000/20 = 52500, got %s", avg)
	})

	t.Run("applies banker rounding on non-terminating division", func(t *testing.T) {
		xs := &snx_lib_core.TWAPExecutionState{
			FilledNotional: shopspring_decimal.NewFromInt(100),
			QuantityFilled: shopspring_decimal.NewFromInt(3),
		}
		avg := xs.AveragePrice(4)
		expected := shopspring_decimal.RequireFromString("33.3333")
		assert.True(t, avg.Equal(expected),
			"expected 100/3 rounded to 4 decimals = 33.3333, got %s", avg)
	})

	t.Run("rounds midpoint to even digit", func(t *testing.T) {
		xs := &snx_lib_core.TWAPExecutionState{
			FilledNotional: shopspring_decimal.RequireFromString("105.005"),
			QuantityFilled: shopspring_decimal.NewFromInt(1),
		}
		avg := xs.AveragePrice(2)
		expected := shopspring_decimal.RequireFromString("105.00")
		assert.True(t, avg.Equal(expected),
			"expected 105.005 banker-rounded to 2dp = 105.00, got %s", avg)
	})
}
