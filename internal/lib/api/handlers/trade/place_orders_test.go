package trade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

func Test_ConvertJSONOrderToGRPC_TriggerPriceType(t *testing.T) {

	t.Run("take profit with trigger price type mark", func(t *testing.T) {
		order := snx_lib_api_json.PlaceOrderRequest{
			Side:             "sell",
			OrderType:        API_WKS_triggerTp,
			Quantity:         Quantity("1"),
			TriggerPrice:     Price("55000"),
			TriggerPriceType: "mark",
			IsTriggerMarket:  true,
		}

		grpcOrder, err := convertJSONOrderToGRPC(order, "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, v4grpc.OrderType_TAKE_PROFIT_MARKET, grpcOrder.Type)
		require.NotNil(t, grpcOrder.TriggerPrice)
		assert.Equal(t, "55000", *grpcOrder.TriggerPrice)
		assert.Equal(t, "mark", grpcOrder.TriggerPriceType)
	})

	t.Run("take profit with trigger price type last", func(t *testing.T) {
		order := snx_lib_api_json.PlaceOrderRequest{
			Side:             "sell",
			OrderType:        API_WKS_triggerTp,
			Quantity:         Quantity("1"),
			TriggerPrice:     Price("55000"),
			TriggerPriceType: "last",
			IsTriggerMarket:  true,
		}

		grpcOrder, err := convertJSONOrderToGRPC(order, "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, "last", grpcOrder.TriggerPriceType)
	})

	t.Run("take profit with empty trigger price type defaults to empty", func(t *testing.T) {
		order := snx_lib_api_json.PlaceOrderRequest{
			Side:            "sell",
			OrderType:       API_WKS_triggerTp,
			Quantity:        Quantity("1"),
			TriggerPrice:    Price("55000"),
			IsTriggerMarket: true,
		}

		grpcOrder, err := convertJSONOrderToGRPC(order, "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, "", grpcOrder.TriggerPriceType)
	})

	t.Run("stop loss with trigger price type last", func(t *testing.T) {
		order := snx_lib_api_json.PlaceOrderRequest{
			Side:             "sell",
			OrderType:        API_WKS_triggerSl,
			Quantity:         Quantity("1"),
			TriggerPrice:     Price("45000"),
			TriggerPriceType: "last",
			IsTriggerMarket:  true,
		}

		grpcOrder, err := convertJSONOrderToGRPC(order, "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, v4grpc.OrderType_STOP_MARKET, grpcOrder.Type)
		assert.Equal(t, "last", grpcOrder.TriggerPriceType)
	})

	t.Run("stop loss with empty trigger price type defaults to empty", func(t *testing.T) {
		order := snx_lib_api_json.PlaceOrderRequest{
			Side:            "sell",
			OrderType:       API_WKS_triggerSl,
			Quantity:        Quantity("1"),
			TriggerPrice:    Price("45000"),
			IsTriggerMarket: true,
		}

		grpcOrder, err := convertJSONOrderToGRPC(order, "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, "", grpcOrder.TriggerPriceType)
	})

	t.Run("stop limit with trigger price type and price", func(t *testing.T) {
		order := snx_lib_api_json.PlaceOrderRequest{
			Side:             "sell",
			OrderType:        API_WKS_triggerSl,
			Quantity:         Quantity("1"),
			Price:            Price("44000"),
			TriggerPrice:     Price("45000"),
			TriggerPriceType: "last",
		}

		grpcOrder, err := convertJSONOrderToGRPC(order, "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, v4grpc.OrderType_STOP, grpcOrder.Type)
		require.NotNil(t, grpcOrder.Price)
		assert.Equal(t, "44000", *grpcOrder.Price)
		assert.Equal(t, "last", grpcOrder.TriggerPriceType)
	})
}

func Test_ConvertJSONOrderToGRPC_InvalidOrderType(t *testing.T) {
	order := snx_lib_api_json.PlaceOrderRequest{
		Side:      "buy",
		OrderType: "invalidType",
		Quantity:  Quantity("1"),
	}

	_, err := convertJSONOrderToGRPC(order, "BTC-USDT")
	require.Error(t, err)
	assert.ErrorIs(t, err, errInvalidOrderType)
	assert.Contains(t, err.Error(), "invalidType")
}

func Test_ConvertJSONOrderToGRPC_TWAPInterval(t *testing.T) {
	t.Run("passes TWAP interval seconds through to grpc", func(t *testing.T) {
		order := snx_lib_api_json.PlaceOrderRequest{
			Side:            "buy",
			OrderType:       "twap",
			Quantity:        Quantity("1"),
			DurationSeconds: 600,
			IntervalSeconds: 120,
		}

		grpcOrder, err := convertJSONOrderToGRPC(order, "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		require.NotNil(t, grpcOrder.TwapParams)
		assert.Equal(t, int64(600), grpcOrder.TwapParams.DurationSeconds)
		assert.Equal(t, int64(120), grpcOrder.TwapParams.IntervalSeconds)
	})
}
