package info

import (
	"context"
	"errors"
	"net/http"
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

func Test_Handle_getMarkets(t *testing.T) {
	t.Run("successful markets retrieval with margin tiers", func(t *testing.T) {
		snx_lib_api_handlers_utils.ResetMarketsCache()
		logger := snx_lib_logging_doubles.NewStubLogger()
		mockMarketConfig := new(MockMarketConfigClient)

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  logger,
			},
			MarketConfigClient: mockMarketConfig,
			ClientRequestId:    "test-request-123",
		}

		// Create test market with margin tiers
		testMarket := &v4grpc.Market{
			Symbol:                     "BTC-USDT",
			Description:                "Bitcoin",
			BaseAsset:                  "BTC",
			QuoteAsset:                 "USDT",
			IsOpen:                     true,
			PriceExponent:              0,
			QuantityExponent:           5,
			TickSize:                   "1.0",
			MinTradeAmount:             "0.001",
			ContractSize:               "1",
			MaxMarketOrderAmount:       "20.00",
			MaxLimitOrderAmount:        "100.00",
			LimitOrderPriceCapRatio:    "0.05",
			LimitOrderPriceFloorRatio:  "0.05",
			MarketOrderPriceCapRatio:   "0.05",
			MarketOrderPriceFloorRatio: "0.05",
			LiquidationClearanceFee:    "0.0125",
			MinNotionalValue:           "10",
			MaintenanceMarginTiers: []*v4grpc.MaintenanceMarginTier{
				{
					MinPositionSize:           "0",
					MaxPositionSize:           "1000000",
					MaxLeverage:               100,
					MaintenanceMarginRatio:    "0.005",
					InitialMarginRatio:        "0.01",
					MaintenanceDeductionValue: "0",
				},
				{
					MinPositionSize:           "1000001",
					MaxPositionSize:           "50000000",
					MaxLeverage:               50,
					MaintenanceMarginRatio:    "0.01",
					InitialMarginRatio:        "0.02",
					MaintenanceDeductionValue: "5000",
				},
				{
					MinPositionSize:           "50000001",
					MaxPositionSize:           "", // Unlimited
					MaxLeverage:               25,
					MaintenanceMarginRatio:    "0.02",
					InitialMarginRatio:        "0.04",
					MaintenanceDeductionValue: "505000",
				},
			},
		}

		mockMarketConfig.On("GetAllMarkets", mock.Anything, mock.Anything, mock.Anything).
			Return(&v4grpc.GetAllMarketsResponse{
				Markets: []*v4grpc.Market{testMarket},
			}, nil)

		req := map[string]any{
			"action": "getMarkets",
		}

		status, response := Handle_getMarkets(ctx, req)

		assert.Equal(t, http.StatusOK, int(status))
		assert.NotNil(t, response)
		assert.Equal(t, "ok", response.Status)
		assert.Equal(t, ClientRequestId("test-request-123"), response.ClientRequestId)

		// Verify the response data contains markets
		markets, ok := response.Response.([]MarketResponse)
		assert.True(t, ok, "Response should be of type []MarketResponse")
		assert.Len(t, markets, 1)

		// Verify market details
		market := markets[0]
		assert.Equal(t, Symbol("BTC-USDT"), market.Symbol)
		assert.Equal(t, "Bitcoin", market.Description)
		assert.Equal(t, Asset("BTC"), market.BaseAsset)
		assert.Equal(t, Asset("USDT"), market.QuoteAsset)
		assert.True(t, market.IsOpen)

		// Verify maintenance margin tiers are included
		assert.Len(t, market.MaintenanceMarginTiers, 3, "Should have 3 margin tiers")

		// Verify first tier
		tier1 := market.MaintenanceMarginTiers[0]
		assert.Equal(t, "0", tier1.MinPositionSize)
		assert.Equal(t, "1000000", tier1.MaxPositionSize)
		assert.Equal(t, uint32(100), tier1.MaxLeverage)
		assert.True(t, tier1.MaintenanceMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.005)))
		assert.True(t, tier1.InitialMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.01)))
		assert.True(t, tier1.MaintenanceDeductionValue.Equal(shopspring_decimal.NewFromInt(0)))

		// Verify second tier
		tier2 := market.MaintenanceMarginTiers[1]
		assert.Equal(t, "1000001", tier2.MinPositionSize)
		assert.Equal(t, "50000000", tier2.MaxPositionSize)
		assert.Equal(t, uint32(50), tier2.MaxLeverage)
		assert.True(t, tier2.MaintenanceMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.01)))
		assert.True(t, tier2.InitialMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.02)))
		assert.True(t, tier2.MaintenanceDeductionValue.Equal(shopspring_decimal.NewFromInt(5000)))

		// Verify third tier (unlimited max position)
		tier3 := market.MaintenanceMarginTiers[2]
		assert.Equal(t, "50000001", tier3.MinPositionSize)
		assert.Equal(t, "", tier3.MaxPositionSize, "Should be empty string for unlimited")
		assert.Equal(t, uint32(25), tier3.MaxLeverage)
		assert.True(t, tier3.MaintenanceMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.02)))
		assert.True(t, tier3.InitialMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.04)))
		assert.True(t, tier3.MaintenanceDeductionValue.Equal(shopspring_decimal.NewFromInt(505000)))

		mockMarketConfig.AssertExpectations(t)
	})

	t.Run("market config service error", func(t *testing.T) {
		snx_lib_api_handlers_utils.ResetMarketsCache()
		logger := snx_lib_logging_doubles.NewStubLogger()
		mockMarketConfig := new(MockMarketConfigClient)

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  logger,
			},
			MarketConfigClient: mockMarketConfig,
			ClientRequestId:    "test-request-456",
		}

		mockMarketConfig.On("GetAllMarkets", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("market config service unavailable"))

		req := map[string]any{
			"action": "getMarkets",
		}

		status, response := Handle_getMarkets(ctx, req)

		assert.Equal(t, http.StatusInternalServerError, int(status))
		assert.NotNil(t, response)
		assert.Equal(t, "error", response.Status)
		assert.Equal(t, ClientRequestId("test-request-456"), response.ClientRequestId)
		assert.NotNil(t, response.Error)
		assert.Contains(t, response.Error.Message, "Could not pull market configuration")

		mockMarketConfig.AssertExpectations(t)
	})

	t.Run("empty markets list", func(t *testing.T) {
		snx_lib_api_handlers_utils.ResetMarketsCache()
		logger := snx_lib_logging_doubles.NewStubLogger()
		mockMarketConfig := new(MockMarketConfigClient)

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  logger,
			},
			MarketConfigClient: mockMarketConfig,
			ClientRequestId:    "test-request-789",
		}

		mockMarketConfig.On("GetAllMarkets", mock.Anything, mock.Anything, mock.Anything).
			Return(&v4grpc.GetAllMarketsResponse{
				Markets: []*v4grpc.Market{},
			}, nil)

		req := map[string]any{
			"action": "getMarkets",
		}

		status, response := Handle_getMarkets(ctx, req)

		assert.Equal(t, http.StatusOK, int(status))
		assert.NotNil(t, response)
		assert.Equal(t, "ok", response.Status)

		markets, ok := response.Response.([]MarketResponse)
		assert.True(t, ok)
		assert.Len(t, markets, 0)

		mockMarketConfig.AssertExpectations(t)
	})
}

func Test_ConvertProtoToMarketResponse(t *testing.T) {
	t.Run("successful conversion with margin tiers", func(t *testing.T) {
		protoMarket := &v4grpc.Market{
			Symbol:                     "ETH-USDT",
			Description:                "Ethereum",
			BaseAsset:                  "ETH",
			QuoteAsset:                 "USDT",
			IsOpen:                     true,
			PriceExponent:              1,
			QuantityExponent:           4,
			TickSize:                   "0.1",
			MinTradeAmount:             "0.001",
			ContractSize:               "1",
			MaxMarketOrderAmount:       "150.00",
			MaxLimitOrderAmount:        "1500.00",
			LimitOrderPriceCapRatio:    "0.05",
			LimitOrderPriceFloorRatio:  "0.05",
			MarketOrderPriceCapRatio:   "0.05",
			MarketOrderPriceFloorRatio: "0.05",
			LiquidationClearanceFee:    "0.0125",
			MinNotionalValue:           "5",
			MaintenanceMarginTiers: []*v4grpc.MaintenanceMarginTier{
				{
					MinPositionSize:        "0",
					MaxPositionSize:        "500000",
					MaxLeverage:            100,
					MaintenanceMarginRatio: "0.005",
					InitialMarginRatio:     "0.01",
				},
			},
		}

		result, err := snx_lib_api_handlers_utils.ConvertProtoToMarketResponse(protoMarket)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Symbol("ETH-USDT"), result.Symbol)
		assert.Equal(t, "Ethereum", result.Description)
		assert.Len(t, result.MaintenanceMarginTiers, 1)

		tier := result.MaintenanceMarginTiers[0]
		assert.Equal(t, "0", tier.MinPositionSize)
		assert.Equal(t, "500000", tier.MaxPositionSize)
		assert.Equal(t, uint32(100), tier.MaxLeverage)
		assert.True(t, tier.MaintenanceMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.005)))
		assert.True(t, tier.InitialMarginRequirement.Equal(shopspring_decimal.NewFromFloat(0.01)))
		assert.True(t, tier.MaintenanceDeductionValue.Equal(shopspring_decimal.Zero), "Should default to zero when omitted")
	})

	t.Run("invalid tick size", func(t *testing.T) {
		protoMarket := &v4grpc.Market{
			Symbol:         "BTC-USDT",
			TickSize:       "invalid",
			MinTradeAmount: "0.001",
		}

		_, err := snx_lib_api_handlers_utils.ConvertProtoToMarketResponse(protoMarket)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tick_size")
	})

	t.Run("invalid maintenance margin ratio", func(t *testing.T) {
		protoMarket := &v4grpc.Market{
			Symbol:                     "BTC-USDT",
			TickSize:                   "1.0",
			MinTradeAmount:             "0.001",
			ContractSize:               "1",
			MaxMarketOrderAmount:       "100",
			MaxLimitOrderAmount:        "1000",
			LimitOrderPriceCapRatio:    "0.05",
			LimitOrderPriceFloorRatio:  "0.05",
			MarketOrderPriceCapRatio:   "0.05",
			MarketOrderPriceFloorRatio: "0.05",
			LiquidationClearanceFee:    "0.0125",
			MinNotionalValue:           "10",
			MaintenanceMarginTiers: []*v4grpc.MaintenanceMarginTier{
				{
					MinPositionSize:        "0",
					MaxPositionSize:        "1000000",
					MaxLeverage:            100,
					MaintenanceMarginRatio: "invalid",
					InitialMarginRatio:     "0.01",
				},
			},
		}

		_, err := snx_lib_api_handlers_utils.ConvertProtoToMarketResponse(protoMarket)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid maintenance_margin_ratio")
	})
}
