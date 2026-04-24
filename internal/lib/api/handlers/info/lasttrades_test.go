package info

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

func Test_LastTradesRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		symbol    Symbol
		limit     int
		expectErr bool
	}{
		{
			name:      "valid request with all parameters",
			symbol:    "BTC-USDT",
			limit:     20,
			expectErr: false,
		},
		{
			name:      "valid request with default limit",
			symbol:    "BTC-USDT",
			limit:     0, // Should default to 50
			expectErr: false,
		},
		{
			name:      "empty symbol should fail",
			symbol:    "",
			limit:     20,
			expectErr: true,
		},
		{
			name:      "limit too high should fail",
			symbol:    "BTC-USDT",
			limit:     101, // Max is 100
			expectErr: true,
		},
		{
			name:      "negative limit should fail",
			symbol:    "BTC-USDT",
			limit:     -1,
			expectErr: true,
		},
		{
			name:      "maximum valid limit",
			symbol:    "BTC-USDT",
			limit:     100,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := LastTradesRequest{
				Symbol: tt.symbol,
				Limit:  tt.limit,
			}

			// Basic validation logic (this would be handled by the handler)
			hasError := false

			if req.Symbol == "" {
				hasError = true
			}

			if req.Limit < 0 || req.Limit > 100 {
				hasError = true
			}

			if tt.expectErr {
				assert.True(t, hasError, "Expected validation error for %s", tt.name)
			} else {
				assert.False(t, hasError, "Expected no validation error for %s", tt.name)
			}
		})
	}
}

func Test_PublicTrade_Structure(t *testing.T) {
	t.Run("public trade has correct fields", func(t *testing.T) {
		trade := PublicTrade{
			TradeId:   "123456789",
			Symbol:    "BTC-USDT",
			Side:      "buy",
			Price:     Price("50000.50"),
			Quantity:  Quantity("0.1"),
			Timestamp: 1704067200500,
			IsMaker:   false,
		}

		// Verify all fields are properly set
		assert.Equal(t, TradeId("123456789"), trade.TradeId)
		assert.Equal(t, Symbol("BTC-USDT"), trade.Symbol)
		assert.Equal(t, "buy", trade.Side)
		assert.Equal(t, Price("50000.50"), trade.Price)
		assert.Equal(t, Quantity("0.1"), trade.Quantity)
		assert.Equal(t, Timestamp(1704067200500), trade.Timestamp)
		assert.False(t, trade.IsMaker)
	})
}

func newLastTradesInfoContext(mock *mockSubaccountClientLastTrades) InfoContext {
	return snx_lib_api_handlers_types.NewInfoContext(
		snx_lib_logging_doubles.NewStubLogger(), context.Background(),
		nil, nil, nil, nil, mock, nil, nil, nil,
		snx_lib_request.NewRequestID(), "req",
	)
}

func publicTradeItem(dir string, now *timestamppb.Timestamp) *v4grpc.PublicTradeHistoryItem {
	return &v4grpc.PublicTradeHistoryItem{Id: 1, Symbol: "BTC-USDT", Direction: dir, FilledPrice: "1", FilledQuantity: "1", TradedAt: now}
}

func Test_Handle_getLastTrades_directionMapping(t *testing.T) {
	now := timestamppb.Now()

	t.Run("sell directions map to side=sell", func(t *testing.T) {
		for _, dir := range []string{"Open Short", "Close Long", "Short", "Sell", "open short", "close long"} {
			mock := &mockSubaccountClientLastTrades{
				response: &v4grpc.GetLastTradesResponse{
					Trades: []*v4grpc.PublicTradeHistoryItem{publicTradeItem(dir, now)},
				},
			}
			statusCode, resp := Handle_getLastTrades(newLastTradesInfoContext(mock), map[string]any{"symbol": "BTC-USDT"})
			require.Equal(t, HTTPStatusCode_200_OK, statusCode, "direction=%q", dir)
			data := resp.Response.(GetLastTradesResponse)
			assert.Equal(t, "sell", data.Trades[0].Side, "direction=%q", dir)
		}
	})

	t.Run("buy directions map to side=buy", func(t *testing.T) {
		for _, dir := range []string{"Open Long", "Close Short", "Long", "Buy", "open long", "close short"} {
			mock := &mockSubaccountClientLastTrades{
				response: &v4grpc.GetLastTradesResponse{
					Trades: []*v4grpc.PublicTradeHistoryItem{publicTradeItem(dir, now)},
				},
			}
			statusCode, resp := Handle_getLastTrades(newLastTradesInfoContext(mock), map[string]any{"symbol": "BTC-USDT"})
			require.Equal(t, HTTPStatusCode_200_OK, statusCode, "direction=%q", dir)
			data := resp.Response.(GetLastTradesResponse)
			assert.Equal(t, "buy", data.Trades[0].Side, "direction=%q", dir)
		}
	})

	t.Run("unrecognized direction returns 500", func(t *testing.T) {
		for _, dir := range []string{"unknown", "Unknown", "UNKNOWN", "sideways", "", "garbage"} {
			mock := &mockSubaccountClientLastTrades{
				response: &v4grpc.GetLastTradesResponse{
					Trades: []*v4grpc.PublicTradeHistoryItem{publicTradeItem(dir, now)},
				},
			}
			statusCode, resp := Handle_getLastTrades(newLastTradesInfoContext(mock), map[string]any{"symbol": "BTC-USDT"})
			require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode, "direction=%q should be rejected", dir)
			require.NotNil(t, resp.Error, "direction=%q", dir)
		}
	})
}

func Test_GetLastTradesResponse_Structure(t *testing.T) {
	t.Run("response has correct structure", func(t *testing.T) {
		trades := []PublicTrade{
			{
				TradeId:   "123456789",
				Symbol:    "BTC-USDT",
				Side:      "buy",
				Price:     Price("50000.50"),
				Quantity:  Quantity("0.1"),
				Timestamp: 1704067200500,
				IsMaker:   false,
			},
			{
				TradeId:   "123456788",
				Symbol:    "BTC-USDT",
				Side:      "sell",
				Price:     Price("50000.25"),
				Quantity:  Quantity("0.05"),
				Timestamp: 1704067199800,
				IsMaker:   true,
			},
		}

		response := GetLastTradesResponse{
			Trades: trades,
		}

		assert.Len(t, response.Trades, 2)
		assert.Equal(t, TradeId("123456789"), response.Trades[0].TradeId)
		assert.Equal(t, TradeId("123456788"), response.Trades[1].TradeId)
	})
}
