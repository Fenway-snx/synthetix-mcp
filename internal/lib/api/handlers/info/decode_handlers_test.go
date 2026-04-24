package info

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockMarketDataClientCandles struct {
	v4grpc.MarketDataServiceClient
	response *v4grpc.GetCandlesticksResponse
	err      error
	lastReq  *v4grpc.GetCandlesticksRequest
}

func (m *mockMarketDataClientCandles) GetCandlesticks(ctx context.Context, in *v4grpc.GetCandlesticksRequest, opts ...grpc.CallOption) (*v4grpc.GetCandlesticksResponse, error) {
	m.lastReq = in
	return m.response, m.err
}

type mockMarketDataClientOrderbook struct {
	v4grpc.MarketDataServiceClient
	response *v4grpc.OrderbookDepthResponse
	err      error
	lastReq  *v4grpc.OrderbookDepthRequest
}

func (m *mockMarketDataClientOrderbook) GetOrderbookDepth(ctx context.Context, in *v4grpc.OrderbookDepthRequest, opts ...grpc.CallOption) (*v4grpc.OrderbookDepthResponse, error) {
	m.lastReq = in
	return m.response, m.err
}

type mockSubaccountClientLastTrades struct {
	v4grpc.SubaccountServiceClient
	response *v4grpc.GetLastTradesResponse
	err      error
	lastReq  *v4grpc.GetLastTradesRequest
}

func (m *mockSubaccountClientLastTrades) GetLastTrades(ctx context.Context, in *v4grpc.GetLastTradesRequest, opts ...grpc.CallOption) (*v4grpc.GetLastTradesResponse, error) {
	m.lastReq = in
	return m.response, m.err
}

type mockSubaccountClientFundingHistory struct {
	v4grpc.SubaccountServiceClient
	response *v4grpc.GetFundingRateHistoryResponse
	err      error
	lastReq  *v4grpc.GetFundingRateHistoryRequest
}

func (m *mockSubaccountClientFundingHistory) GetFundingRateHistory(ctx context.Context, in *v4grpc.GetFundingRateHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetFundingRateHistoryResponse, error) {
	m.lastReq = in
	return m.response, m.err
}

type mockSubaccountClientFundingRate struct {
	v4grpc.SubaccountServiceClient
	response *v4grpc.GetLatestFundingRatesResponse
	err      error
	lastReq  *v4grpc.GetLatestFundingRatesRequest
}

func (m *mockSubaccountClientFundingRate) GetLatestFundingRates(ctx context.Context, in *v4grpc.GetLatestFundingRatesRequest, opts ...grpc.CallOption) (*v4grpc.GetLatestFundingRatesResponse, error) {
	m.lastReq = in
	return m.response, m.err
}

type mockMarketConfigClientGetMarkets struct {
	v4grpc.MarketConfigServiceClient
	response *v4grpc.GetAllMarketsResponse
	err      error
	lastReq  *v4grpc.GetAllMarketsRequest
}

func (m *mockMarketConfigClientGetMarkets) GetAllMarkets(ctx context.Context, in *v4grpc.GetAllMarketsRequest, opts ...grpc.CallOption) (*v4grpc.GetAllMarketsResponse, error) {
	m.lastReq = in
	return m.response, m.err
}

// ---------------------------------------------------------------------------
// Handle_getCandles
// ---------------------------------------------------------------------------

func Test_Decode_Handle_getCandles(t *testing.T) {
	t.Parallel()

	emptyCandleResp := &v4grpc.GetCandlesticksResponse{
		Candlesticks: []*v4grpc.Candlestick{},
	}

	t.Run("valid request with all fields as float64", func(t *testing.T) {
		t.Parallel()

		mock := &mockMarketDataClientCandles{response: emptyCandleResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, mock, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getCandles(ctx, map[string]any{
			"symbol":    "BTC-USD",
			"interval":  "1h",
			"limit":     float64(50),
			"startTime": float64(1000),
			"endTime":   float64(2000),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, "BTC-USD", mock.lastReq.Symbol)
		assert.Equal(t, "1h", mock.lastReq.Timeframe)
		assert.Equal(t, int32(50), mock.lastReq.Limit)
	})

	t.Run("missing symbol", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, _ := Handle_getCandles(ctx, map[string]any{
			"interval": "1h",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("missing interval", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, _ := Handle_getCandles(ctx, map[string]any{
			"symbol": "BTC-USD",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("invalid interval", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, _ := Handle_getCandles(ctx, map[string]any{
			"symbol":   "BTC-USD",
			"interval": "3h",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("negative limit", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, _ := Handle_getCandles(ctx, map[string]any{
			"symbol":   "BTC-USD",
			"interval": "1h",
			"limit":    float64(-5),
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("startTime >= endTime", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, _ := Handle_getCandles(ctx, map[string]any{
			"symbol":    "BTC-USD",
			"interval":  "1h",
			"startTime": float64(5000),
			"endTime":   float64(5000),
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("limit as float64 coercion", func(t *testing.T) {
		t.Parallel()

		mock := &mockMarketDataClientCandles{response: emptyCandleResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, mock, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getCandles(ctx, map[string]any{
			"symbol":   "ETH-USD",
			"interval": "5m",
			"limit":    float64(50),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, int32(50), mock.lastReq.Limit)
	})

	t.Run("timestamps as float64 coercion", func(t *testing.T) {
		t.Parallel()

		mock := &mockMarketDataClientCandles{response: emptyCandleResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, mock, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getCandles(ctx, map[string]any{
			"symbol":    "ETH-USD",
			"interval":  "1d",
			"startTime": float64(1700000000000),
			"endTime":   float64(1700086400000),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
	})

	t.Run("grpc error", func(t *testing.T) {
		t.Parallel()

		mock := &mockMarketDataClientCandles{err: errors.New("unavailable")}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, mock, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getCandles(ctx, map[string]any{
			"symbol":   "BTC-USD",
			"interval": "1h",
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
	})
}

// ---------------------------------------------------------------------------
// Handle_getFundingRate — decode-specific additions
// ---------------------------------------------------------------------------

func Test_Decode_Handle_getFundingRate(t *testing.T) {
	t.Parallel()

	t.Run("symbol missing from params (empty map)", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getFundingRate(ctx, map[string]any{})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("symbol as non-string type", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getFundingRate(ctx, map[string]any{
			"symbol": float64(123),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})
}

// ---------------------------------------------------------------------------
// Handle_getFundingRateHistory — decode-specific additions
// ---------------------------------------------------------------------------

func Test_Decode_Handle_getFundingRateHistory(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startTimeMs := now.Add(-24 * time.Hour).UnixMilli()
	endTimeMs := now.UnixMilli()

	t.Run("all timestamps as float64 coercion", func(t *testing.T) {
		t.Parallel()

		fundingTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

		mock := &mockSubaccountClientFundingHistory{
			response: &v4grpc.GetFundingRateHistoryResponse{
				Symbol: "BTC-USDT",
				FundingRates: []*v4grpc.FundingRateHistoryItem{
					{
						FundingRate: "0.0001",
						FundingTime: timestamppb.New(fundingTime),
						AppliedAt:   timestamppb.New(fundingTime.Add(time.Minute)),
					},
				},
			},
		}

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, mock, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": float64(startTimeMs),
			"endTime":   float64(endTimeMs),
			"limit":     float64(100),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, "BTC-USDT", mock.lastReq.Symbol)
		assert.Equal(t, int32(100), mock.lastReq.Limit)
	})

	t.Run("limit as float64 coercion", func(t *testing.T) {
		t.Parallel()

		mock := &mockSubaccountClientFundingHistory{
			response: &v4grpc.GetFundingRateHistoryResponse{
				Symbol:       "ETH-USDT",
				FundingRates: []*v4grpc.FundingRateHistoryItem{},
			},
		}

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, mock, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "ETH-USDT",
			"startTime": float64(startTimeMs),
			"endTime":   float64(endTimeMs),
			"limit":     float64(100),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, int32(100), mock.lastReq.Limit)
	})
}

// ---------------------------------------------------------------------------
// Handle_getLastTrades
// ---------------------------------------------------------------------------

func Test_Decode_Handle_getLastTrades(t *testing.T) {
	t.Parallel()

	emptyTradesResp := &v4grpc.GetLastTradesResponse{
		Trades: []*v4grpc.PublicTradeHistoryItem{},
	}

	t.Run("valid minimal (symbol only, limit defaults to 50)", func(t *testing.T) {
		t.Parallel()

		mock := &mockSubaccountClientLastTrades{response: emptyTradesResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, mock, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getLastTrades(ctx, map[string]any{
			"symbol": "BTC-USDT",
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)
		require.NotNil(t, mock.lastReq)
		assert.Equal(t, "BTC-USDT", mock.lastReq.Symbol)
		assert.Equal(t, int32(50), *mock.lastReq.Limit)
	})

	t.Run("valid with limit as float64", func(t *testing.T) {
		t.Parallel()

		mock := &mockSubaccountClientLastTrades{response: emptyTradesResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, mock, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getLastTrades(ctx, map[string]any{
			"symbol": "ETH-USDT",
			"limit":  float64(20),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, int32(20), *mock.lastReq.Limit)
	})

	t.Run("limit > 100", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getLastTrades(ctx, map[string]any{
			"symbol": "BTC-USDT",
			"limit":  float64(200),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("missing symbol", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getLastTrades(ctx, map[string]any{
			"limit": float64(10),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("grpc error", func(t *testing.T) {
		t.Parallel()

		mock := &mockSubaccountClientLastTrades{err: errors.New("unavailable")}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, mock, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getLastTrades(ctx, map[string]any{
			"symbol": "BTC-USDT",
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
	})
}

// ---------------------------------------------------------------------------
// Handle_getOrderbook
// ---------------------------------------------------------------------------

func Test_Decode_Handle_getOrderbook(t *testing.T) {
	t.Parallel()

	emptyOrderbookResp := &v4grpc.OrderbookDepthResponse{
		Bids: []*v4grpc.PriceLevel{},
		Asks: []*v4grpc.PriceLevel{},
	}

	t.Run("valid with default limit", func(t *testing.T) {
		t.Parallel()

		mock := &mockMarketDataClientOrderbook{response: emptyOrderbookResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, mock, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getOrderbook(ctx, map[string]any{
			"symbol": "BTC-USD",
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, int32(500), mock.lastReq.Limit)
	})

	t.Run("valid limits", func(t *testing.T) {
		t.Parallel()

		for _, limit := range []int{5, 10, 20, 50, 100, 500, 1000} {
			t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
				mock := &mockMarketDataClientOrderbook{response: emptyOrderbookResp}
				ctx := snx_lib_api_handlers_types.NewInfoContext(
					snx_lib_logging_doubles.NewStubLogger(), context.Background(),
					nil, nil, nil, nil, nil, mock, nil, nil,
					snx_lib_request.NewRequestID(), "req",
				)

				status, resp := Handle_getOrderbook(ctx, map[string]any{
					"symbol": "BTC-USD",
					"limit":  float64(limit),
				})

				require.NotNil(t, resp)
				assert.Equal(t, HTTPStatusCode_200_OK, status)
				assert.Equal(t, int32(limit), mock.lastReq.Limit)
			})
		}
	})

	t.Run("invalid limit (15)", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getOrderbook(ctx, map[string]any{
			"symbol": "BTC-USD",
			"limit":  float64(15),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
	})

	t.Run("limit as float64 coercion", func(t *testing.T) {
		t.Parallel()

		mock := &mockMarketDataClientOrderbook{response: emptyOrderbookResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, mock, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getOrderbook(ctx, map[string]any{
			"symbol": "BTC-USD",
			"limit":  float64(100),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, int32(100), mock.lastReq.Limit)
	})

	t.Run("grpc error", func(t *testing.T) {
		t.Parallel()

		mock := &mockMarketDataClientOrderbook{err: errors.New("unavailable")}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, mock, nil, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getOrderbook(ctx, map[string]any{
			"symbol": "BTC-USD",
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
	})
}

// ---------------------------------------------------------------------------
// Handle_getMarkets — decode-specific (direct params access, not mapstructure)
// ---------------------------------------------------------------------------

func Test_Decode_Handle_getMarkets(t *testing.T) {
	emptyMarketsResp := &v4grpc.GetAllMarketsResponse{
		Markets: []*v4grpc.Market{},
	}

	t.Run("no params - activeOnly defaults to false", func(t *testing.T) {
		snx_lib_api_handlers_utils.ResetMarketsCache()

		mock := &mockMarketConfigClientGetMarkets{response: emptyMarketsResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, mock, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getMarkets(ctx, map[string]any{})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)
		assert.False(t, mock.lastReq.ActiveOnly)
	})

	t.Run("activeOnly=true", func(t *testing.T) {
		snx_lib_api_handlers_utils.ResetMarketsCache()

		mock := &mockMarketConfigClientGetMarkets{response: emptyMarketsResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, mock, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getMarkets(ctx, map[string]any{
			"activeOnly": true,
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.True(t, mock.lastReq.ActiveOnly)
	})

	t.Run("activeOnly as string stays false (current behavior)", func(t *testing.T) {
		snx_lib_api_handlers_utils.ResetMarketsCache()

		mock := &mockMarketConfigClientGetMarkets{response: emptyMarketsResp}
		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(), context.Background(),
			nil, nil, nil, nil, nil, nil, mock, nil,
			snx_lib_request.NewRequestID(), "req",
		)

		status, resp := Handle_getMarkets(ctx, map[string]any{
			"activeOnly": "true",
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.False(t, mock.lastReq.ActiveOnly)
	})
}
