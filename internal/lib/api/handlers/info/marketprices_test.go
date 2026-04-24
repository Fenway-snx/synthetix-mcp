package info

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"encoding/json"
	"errors"
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_repository "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/repository"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type mockMarketDataClientGetAllMarketData struct {
	v4grpc.MarketDataServiceClient
	response *v4grpc.GetAllMarketDataResponse
	err      error
}

var _ v4grpc.MarketDataServiceClient = (*mockMarketDataClientGetAllMarketData)(nil)

func (m *mockMarketDataClientGetAllMarketData) GetAllMarketData(ctx context.Context, in *v4grpc.GetAllMarketDataRequest, opts ...grpc.CallOption) (*v4grpc.GetAllMarketDataResponse, error) {
	return m.response, m.err
}

func Test_fetchMarketPricesData(t *testing.T) {
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

		data, err := fetchMarketPricesData(ctx, nil)

		assert.Nil(t, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not pull market configuration")

		mockMarketConfig.AssertExpectations(t)
	})

	t.Run("skips symbol when bulk market data omits it (stale cache race)", func(t *testing.T) {
		snx_lib_api_handlers_utils.ResetMarketsCache()

		mockLogger := snx_lib_logging_doubles.NewStubLogger()
		mockMarketConfig := new(MockMarketConfigClient)
		mockMarketData := &mockMarketDataClientGetAllMarketData{
			response: &v4grpc.GetAllMarketDataResponse{
				Markets: []*v4grpc.MarketDataResponse{},
			},
		}

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context:          context.Background(),
				Logger:           mockLogger,
				SubaccountClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			},
			MarketConfigClient: mockMarketConfig,
			MarketDataClient:   mockMarketData,
			ClientRequestId:    "test-request-789",
		}

		mockMarketConfig.On("GetAllMarkets", mock.Anything, mock.Anything, mock.Anything).
			Return(&v4grpc.GetAllMarketsResponse{
				Markets: []*v4grpc.Market{
					CreateTestMarket("BTC-USD"),
				},
			}, nil)

		data, err := fetchMarketPricesData(ctx, nil)

		assert.NoError(t, err, "should not error when market data is missing for a symbol")
		assert.NotNil(t, data)
		assert.Empty(t, data, "deactivated symbol should be omitted from results")

		mockMarketConfig.AssertExpectations(t)
	})
}

func Test_MarketPriceTypes(t *testing.T) {
	t.Run("MarketPriceResponse structure validation", func(t *testing.T) {
		mp := MarketPriceResponse{
			Symbol:     Symbol("BTC-USD"),
			BestBid:    Price("45000.00"),
			BestAsk:    Price("45001.00"),
			MarkPrice:  Price("45000.50"),
			IndexPrice: Price("45000.75"),
			LastPrice:  Price("45000.25"),
		}

		assert.Equal(t, Symbol("BTC-USD"), mp.Symbol)
		assert.Equal(t, Price("45000.00"), mp.BestBid)
		assert.Equal(t, Price("45001.00"), mp.BestAsk)
		assert.Equal(t, Price("45000.50"), mp.MarkPrice)
		assert.Equal(t, Price("45000.75"), mp.IndexPrice)
		assert.Equal(t, Price("45000.25"), mp.LastPrice)
	})

	t.Run("MarketPricesResponse type is slice of MarketPriceResponse", func(t *testing.T) {
		prices := []MarketPriceResponse{
			{Symbol: Symbol("BTC-USD"), MarkPrice: Price("45000")},
			{Symbol: Symbol("ETH-USD"), MarkPrice: Price("3000")},
		}

		assert.Len(t, prices, 2)
		assert.Equal(t, Symbol("BTC-USD"), prices[0].Symbol)
		assert.Equal(t, Symbol("ETH-USD"), prices[1].Symbol)
	})

	t.Run("default values for missing price data", func(t *testing.T) {
		mp := MarketPriceResponse{
			Symbol: Symbol("NEW-USD"),
		}

		assert.Equal(t, Symbol("NEW-USD"), mp.Symbol)
		assert.Equal(t, Price_None, mp.BestBid)
		assert.Equal(t, Price_None, mp.BestAsk)
		assert.Equal(t, Price_None, mp.MarkPrice)
		assert.Equal(t, Price_None, mp.IndexPrice)
		assert.Equal(t, Price_None, mp.LastPrice)
	})

	t.Run("funding rate marshals as string", func(t *testing.T) {
		mp := MarketPriceResponse{
			Symbol:      Symbol("BTC-USD"),
			FundingRate: "0.00123",
		}

		payload, err := json.Marshal(mp)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Contains(t, string(payload), `"fundingRate":"0.00123"`)
	})
}

func Test_loadPriceFromCacheOrFallbackWithFetcher(t *testing.T) {
	t.Run("falls back to repository when cache is unavailable", func(t *testing.T) {
		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  snx_lib_logging_doubles.NewStubLogger(),
			},
		}

		price, err := loadPriceFromCacheOrFallbackWithFetcher(
			ctx,
			nil,
			snx_lib_core.MarketName("BTC-USD"),
			snx_lib_core.PriceType_mark,
			func(context.Context, string, snx_lib_core.PriceType, int) ([]snx_lib_db_repository.PriceData, error) {
				return []snx_lib_db_repository.PriceData{
					{Price: shopspring_decimal.RequireFromString("123.45")},
				}, nil
			},
		)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Price("123.45"), price)
	})

	t.Run("returns empty when no cache or repository is available", func(t *testing.T) {
		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  snx_lib_logging_doubles.NewStubLogger(),
			},
		}

		price, err := loadPriceFromCacheOrFallbackWithFetcher(
			ctx,
			nil,
			snx_lib_core.MarketName("BTC-USD"),
			snx_lib_core.PriceType_mark,
			nil,
		)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Price_None, price)
	})

	t.Run("wraps repository fallback errors", func(t *testing.T) {
		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  snx_lib_logging_doubles.NewStubLogger(),
			},
		}

		price, err := loadPriceFromCacheOrFallbackWithFetcher(
			ctx,
			nil,
			snx_lib_core.MarketName("BTC-USD"),
			snx_lib_core.PriceType_mark,
			func(context.Context, string, snx_lib_core.PriceType, int) ([]snx_lib_db_repository.PriceData, error) {
				return nil, errors.New("redis unavailable")
			},
		)

		assert.Equal(t, Price_None, price)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not get mark price for BTC-USD")
	})
}
