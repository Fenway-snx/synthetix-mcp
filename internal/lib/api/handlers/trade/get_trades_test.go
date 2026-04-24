package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type capturingGetTradesMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	resp *v4grpc.GetTradeHistoryResponse
	err  error
}

func (m *capturingGetTradesMock) GetTradeHistory(_ context.Context, _ *v4grpc.GetTradeHistoryRequest, _ ...grpc.CallOption) (*v4grpc.GetTradeHistoryResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil {
		return m.resp, nil
	}
	return &v4grpc.GetTradeHistoryResponse{}, nil
}

func Test_getTrades_directionMapping(t *testing.T) {
	now := timestamppb.Now()

	t.Run("sell directions map to side=sell", func(t *testing.T) {
		for _, dir := range []string{"Open Short", "Close Long", "Short", "Sell", "open short", "close long"} {
			mock := &capturingGetTradesMock{
				MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
				resp: &v4grpc.GetTradeHistoryResponse{
					Trades: []*v4grpc.TradeHistoryItem{
						{Id: 1, Symbol: "BTC-USDT", Direction: dir, FilledPrice: "1", FilledQuantity: "1", TradedAt: now, TradeId: 1},
					},
				},
			}
			ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)
			statusCode, resp := Handle_getTrades(ctx, map[string]any{})
			require.Equal(t, HTTPStatusCode_200_OK, statusCode, "direction=%q", dir)
			data := resp.Response.(GetTradesResponse)
			assert.Equal(t, "sell", data.Trades[0].Side, "direction=%q", dir)
		}
	})

	t.Run("buy directions map to side=buy", func(t *testing.T) {
		for _, dir := range []string{"Open Long", "Close Short", "Long", "Buy", "open long", "close short"} {
			mock := &capturingGetTradesMock{
				MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
				resp: &v4grpc.GetTradeHistoryResponse{
					Trades: []*v4grpc.TradeHistoryItem{
						{Id: 1, Symbol: "BTC-USDT", Direction: dir, FilledPrice: "1", FilledQuantity: "1", TradedAt: now, TradeId: 1},
					},
				},
			}
			ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)
			statusCode, resp := Handle_getTrades(ctx, map[string]any{})
			require.Equal(t, HTTPStatusCode_200_OK, statusCode, "direction=%q", dir)
			data := resp.Response.(GetTradesResponse)
			assert.Equal(t, "buy", data.Trades[0].Side, "direction=%q", dir)
		}
	})

	t.Run("unrecognized direction returns 500", func(t *testing.T) {
		for _, dir := range []string{"unknown", "Unknown", "UNKNOWN", "sideways", "", "garbage"} {
			mock := &capturingGetTradesMock{
				MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
				resp: &v4grpc.GetTradeHistoryResponse{
					Trades: []*v4grpc.TradeHistoryItem{
						{Id: 1, Symbol: "BTC-USDT", Direction: dir, FilledPrice: "1", FilledQuantity: "1", TradedAt: now, TradeId: 1},
					},
				},
			}
			ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)
			statusCode, resp := Handle_getTrades(ctx, map[string]any{})
			require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode, "direction=%q should be rejected", dir)
			require.NotNil(t, resp.Error, "direction=%q", dir)
		}
	})
}
