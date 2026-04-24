package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type capturingTradesForPositionMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	captured *v4grpc.GetTradesForPositionRequest
	resp     *v4grpc.GetTradesForPositionResponse
	err      error
}

func (m *capturingTradesForPositionMock) GetTradesForPosition(_ context.Context, req *v4grpc.GetTradesForPositionRequest, _ ...grpc.CallOption) (*v4grpc.GetTradesForPositionResponse, error) {
	m.captured = req
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil {
		return m.resp, nil
	}
	return &v4grpc.GetTradesForPositionResponse{}, nil
}

func Test_getTradesForPosition(t *testing.T) {
	t.Run("success - returns trades for position", func(t *testing.T) {
		now := timestamppb.Now()
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTradesForPositionResponse{
				Trades: []*v4grpc.TradeHistoryItem{
					{
						Id:             1001,
						SubAccountId:   100,
						Symbol:         "ETH-USDT",
						OrderType:      "Limit",
						Direction:      "Open Long",
						FilledPrice:    "3000",
						FilledQuantity: "1.5",
						FillType:       "Taker",
						TradedAt:       now,
						Fee:            "2.5",
						FeeRate:        "0.001",
						ClosedPnl:      "0",
						MarkPrice:      "3010",
						EntryPrice:     "3000",
						PostOnly:       false,
						ReduceOnly:     false,
						TradeId:        5001,
					},
				},
				HasMore: false,
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		require.NotNil(t, mock.captured)
		assert.Equal(t, int64(100), mock.captured.SubAccountId)
		assert.Equal(t, uint64(42), mock.captured.PositionId)

		data, ok := resp.Response.(GetTradesForPositionResponse)
		require.True(t, ok)
		require.Len(t, data.Trades, 1)
		assert.Equal(t, TradeId("1001"), data.Trades[0].TradeId)
		assert.Equal(t, "ETH-USDT", string(data.Trades[0].Symbol))
		assert.Equal(t, OrderType("Limit"), data.Trades[0].OrderType)
		assert.Equal(t, "buy", data.Trades[0].Side)
		assert.Equal(t, Price("3000"), data.Trades[0].Price)
		assert.Equal(t, Quantity("1.5"), data.Trades[0].Quantity)
		assert.False(t, data.HasMore)
	})

	t.Run("success - hasMore true with trades", func(t *testing.T) {
		now := timestamppb.Now()
		// Simulate a page of 2 trades with more available beyond this page
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTradesForPositionResponse{
				Trades: []*v4grpc.TradeHistoryItem{
					{
						Id:             1001,
						SubAccountId:   100,
						Symbol:         "ETH-USDT",
						Direction:      "Open Long",
						FilledPrice:    "3000",
						FilledQuantity: "1.0",
						FillType:       "Taker",
						TradedAt:       now,
						TradeId:        5001,
					},
					{
						Id:             1002,
						SubAccountId:   100,
						Symbol:         "ETH-USDT",
						Direction:      "Open Long",
						FilledPrice:    "3100",
						FilledQuantity: "0.5",
						FillType:       "Taker",
						TradedAt:       now,
						TradeId:        5002,
					},
				},
				HasMore: true,
				Offset:  0,
				Limit:   2,
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
			"limit":      2,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		data, ok := resp.Response.(GetTradesForPositionResponse)
		require.True(t, ok)
		require.Len(t, data.Trades, 2)
		assert.True(t, data.HasMore)
	})

	t.Run("success - second page returns remaining trades with hasMore false", func(t *testing.T) {
		now := timestamppb.Now()
		// Simulate offset=2 into a set of 3 total trades: returns 1 trade, no more pages
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTradesForPositionResponse{
				Trades: []*v4grpc.TradeHistoryItem{
					{
						Id:             1003,
						SubAccountId:   100,
						Symbol:         "ETH-USDT",
						Direction:      "Close Long",
						FilledPrice:    "3200",
						FilledQuantity: "0.5",
						FillType:       "Taker",
						TradedAt:       now,
						TradeId:        5003,
					},
				},
				HasMore: false,
				Offset:  2,
				Limit:   2,
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
			"limit":      2,
			"offset":     2,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		assert.Equal(t, int32(2), mock.captured.GetOffset())
		assert.Equal(t, int32(2), mock.captured.GetLimit())

		data, ok := resp.Response.(GetTradesForPositionResponse)
		require.True(t, ok)
		require.Len(t, data.Trades, 1)
		assert.Equal(t, TradeId("1003"), data.Trades[0].TradeId)
		assert.Equal(t, "sell", data.Trades[0].Side)
		assert.False(t, data.HasMore)
	})

	t.Run("success - empty result", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTradesForPositionResponse{
				Trades: []*v4grpc.TradeHistoryItem{},
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		data, ok := resp.Response.(GetTradesForPositionResponse)
		require.True(t, ok)
		assert.Empty(t, data.Trades)
		assert.False(t, data.HasMore)
	})

	t.Run("success - pagination params forwarded to gRPC", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTradesForPositionResponse{
				Trades: []*v4grpc.TradeHistoryItem{},
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, _ := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
			"limit":      50,
			"offset":     10,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		assert.Equal(t, int32(10), mock.captured.GetOffset())
		assert.Equal(t, int32(50), mock.captured.GetLimit())
	})

	t.Run("success - default limit applied when not specified", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTradesForPositionResponse{
				Trades: []*v4grpc.TradeHistoryItem{},
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, _ := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		assert.Equal(t, int32(100), mock.captured.GetLimit())
		assert.Equal(t, int32(0), mock.captured.GetOffset())
	})

	t.Run("validation error - negative offset", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
			"offset":     -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.Nil(t, mock.captured, "gRPC should not be called with invalid offset")
	})

	t.Run("validation error - negative limit", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
			"limit":      -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.Nil(t, mock.captured, "gRPC should not be called with invalid limit")
	})

	t.Run("validation error - limit exceeds 1000", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
			"limit":      1001,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.Nil(t, mock.captured, "gRPC should not be called when limit exceeds max")
	})

	t.Run("validation error - missing subaccountId", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 0, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "subaccountId is required")
		assert.Nil(t, mock.captured, "gRPC should not be called when subaccountId is missing")
	})

	t.Run("validation error - missing positionId", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "positionId is required")
		assert.Nil(t, mock.captured, "gRPC should not be called when positionId is missing")
	})

	t.Run("validation error - non-numeric positionId", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "abc",
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "positionId must be a valid numeric value")
	})

	t.Run("error - grpc InvalidArgument returns 400", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         status.Error(codes.InvalidArgument, "invalid position id"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("error - grpc internal failure returns 500", func(t *testing.T) {
		mock := &capturingTradesForPositionMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         errors.New("subaccount unavailable"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{
			"positionId": "42",
		})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "Failed to retrieve trades for position", resp.Error.Message)
	})

	t.Run("direction mapping - sell directions map to side=sell", func(t *testing.T) {
		now := timestamppb.Now()
		for _, dir := range []string{"Open Short", "Close Long", "Short", "Sell", "open short", "close long"} {
			mock := &capturingTradesForPositionMock{
				MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
				resp: &v4grpc.GetTradesForPositionResponse{
					Trades: []*v4grpc.TradeHistoryItem{
						{Id: 1, Symbol: "BTC-USDT", Direction: dir, FilledPrice: "1", FilledQuantity: "1", TradedAt: now, TradeId: 1},
					},
				},
			}
			ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)
			statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{"positionId": "1"})
			require.Equal(t, HTTPStatusCode_200_OK, statusCode, "direction=%q", dir)
			data := resp.Response.(GetTradesForPositionResponse)
			assert.Equal(t, "sell", data.Trades[0].Side, "direction=%q", dir)
		}
	})

	t.Run("direction mapping - buy directions map to side=buy", func(t *testing.T) {
		now := timestamppb.Now()
		for _, dir := range []string{"Open Long", "Close Short", "Long", "Buy", "open long", "close short"} {
			mock := &capturingTradesForPositionMock{
				MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
				resp: &v4grpc.GetTradesForPositionResponse{
					Trades: []*v4grpc.TradeHistoryItem{
						{Id: 1, Symbol: "BTC-USDT", Direction: dir, FilledPrice: "1", FilledQuantity: "1", TradedAt: now, TradeId: 1},
					},
				},
			}
			ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)
			statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{"positionId": "1"})
			require.Equal(t, HTTPStatusCode_200_OK, statusCode, "direction=%q", dir)
			data := resp.Response.(GetTradesForPositionResponse)
			assert.Equal(t, "buy", data.Trades[0].Side, "direction=%q", dir)
		}
	})

	t.Run("direction mapping - unrecognized direction returns 500", func(t *testing.T) {
		now := timestamppb.Now()
		for _, dir := range []string{"unknown", "Unknown", "UNKNOWN", "sideways", "", "garbage"} {
			mock := &capturingTradesForPositionMock{
				MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
				resp: &v4grpc.GetTradesForPositionResponse{
					Trades: []*v4grpc.TradeHistoryItem{
						{Id: 1, Symbol: "BTC-USDT", Direction: dir, FilledPrice: "1", FilledQuantity: "1", TradedAt: now, TradeId: 1},
					},
				},
			}
			ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)
			statusCode, resp := Handle_getTradesForPosition(ctx, map[string]any{"positionId": "1"})
			require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode, "direction=%q should be rejected", dir)
			require.NotNil(t, resp.Error, "direction=%q", dir)
		}
	})
}
