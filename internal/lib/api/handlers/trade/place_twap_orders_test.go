package trade

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// ---------------------------------------------------------------------------
// TWAP-specific mock
// ---------------------------------------------------------------------------

type mockTradingTWAP struct {
	v4grpc.TradingServiceClient
	resp *v4grpc.PlaceOrderResponse
	err  error
}

func (m *mockTradingTWAP) PlaceTWAPOrder(_ context.Context, _ *v4grpc.PlaceOrderRequest, _ ...grpc.CallOption) (*v4grpc.PlaceOrderResponse, error) {
	return m.resp, m.err
}

func newTWAPPayload() *ValidatedPlaceOrdersAction {
	return &ValidatedPlaceOrdersAction{
		Payload: &PlaceOrdersActionPayload{
			Action:   "placeOrders",
			Grouping: GroupingValues_twap,
			Orders: []snx_lib_api_json.PlaceOrderRequest{
				{
					Symbol:          "BTC-USD",
					Side:            "buy",
					OrderType:       "twap",
					Quantity:        Quantity("1.0"),
					DurationSeconds: 600,
				},
			},
		},
	}
}

// ===========================================================================
// handleTWAPOrder — gRPC error classification tests
// ===========================================================================

func Test_handleTWAPOrder_GRPCError_InvalidArgument(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "validation failure — min notional",
			message: "TWAP order validation failed: TWAP validation failed: TWAP total order size must be at least $10,000 USD equivalent: notional_value=9993.58 USD, minimum=10000 USD",
		},
		{
			name:    "validation failure — duration too short",
			message: "TWAP order validation failed: TWAP validation failed: TWAP duration must be at least 5 minutes: duration=60s, minimum=300s",
		},
		{
			name:    "validation failure — missing twap_params",
			message: "TWAP order validation failed: TWAP order requires twap_params",
		},
		{
			name:    "validation failure — wrong order type",
			message: "PlaceTWAPOrder requires order type TWAP, got LIMIT",
		},
		{
			name:    "validation failure — multiple orders",
			message: "TWAP request must contain exactly one order",
		},
		{
			name:    "validation failure — missing request_id",
			message: "request id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTradingTWAP{
				err: status.Error(codes.InvalidArgument, tt.message),
			}
			ctx := newActionTradeContext(100, mock, newTWAPPayload())

			code, resp := Handle_placeOrders(ctx, HandlerParams{})

			assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
			require.NotNil(t, resp)
			assert.Equal(t, "error", resp.Status)
			require.NotNil(t, resp.Error)
			assert.Equal(t, snx_lib_api_json.ErrorCodeValidationError, resp.Error.Code)
			assert.Equal(t, tt.message, resp.Error.Message)
		})
	}
}

func Test_handleTWAPOrder_GRPCError_Internal(t *testing.T) {
	mock := &mockTradingTWAP{
		err: status.Error(codes.Internal, "failed to send TWAP order to actor: actor mailbox full"),
	}
	ctx := newActionTradeContext(100, mock, newTWAPPayload())

	code, resp := Handle_placeOrders(ctx, HandlerParams{})

	assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
	require.NotNil(t, resp)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
}

func Test_handleTWAPOrder_GRPCError_DeadlineExceeded(t *testing.T) {
	mock := &mockTradingTWAP{
		err: status.Error(codes.DeadlineExceeded, "timeout waiting for TWAP order response"),
	}
	ctx := newActionTradeContext(100, mock, newTWAPPayload())

	code, resp := Handle_placeOrders(ctx, HandlerParams{})

	// NewSystemErrorResponse detects DeadlineExceeded and returns timeout response.
	// The dispatch layer uses HasTimeoutError() to map this to 503.
	_ = code
	require.NotNil(t, resp)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeRequestTimeout, resp.Error.Code)
	assert.True(t, resp.HasTimeoutError())
}

func Test_handleTWAPOrder_NonGRPCError(t *testing.T) {
	mock := &mockTradingTWAP{
		err: errors.New("connection refused"),
	}
	ctx := newActionTradeContext(100, mock, newTWAPPayload())

	code, resp := Handle_placeOrders(ctx, HandlerParams{})

	assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
	require.NotNil(t, resp)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
}

func Test_handleTWAPOrder_Success(t *testing.T) {
	mock := &mockTradingTWAP{
		resp: &v4grpc.PlaceOrderResponse{
			Orders: []*v4grpc.PlaceOrderResponseItem{
				{
					IsSuccess: true,
					Status:    v4grpc.OrderStatus_ACCEPTED,
					OrderId:   &v4grpc.OrderId{VenueId: 12345},
				},
			},
		},
	}
	ctx := newActionTradeContext(100, mock, newTWAPPayload())

	code, resp := Handle_placeOrders(ctx, HandlerParams{})

	assert.Equal(t, HTTPStatusCode_200_OK, code)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)

	data, ok := resp.Response.(OrderDataResponse)
	require.True(t, ok)
	require.Len(t, data.Statuses, 1)
	require.NotNil(t, data.Statuses[0].Resting)
}

func Test_handleTWAPOrder_ActorRejection(t *testing.T) {
	// Actor-level rejections come back in the structured response body (not gRPC error)
	mock := &mockTradingTWAP{
		resp: &v4grpc.PlaceOrderResponse{
			Orders: []*v4grpc.PlaceOrderResponseItem{
				{
					IsSuccess: false,
					Message:   "insufficient margin",
					OrderId:   &v4grpc.OrderId{VenueId: 12345},
				},
			},
		},
	}
	ctx := newActionTradeContext(100, mock, newTWAPPayload())

	code, resp := Handle_placeOrders(ctx, HandlerParams{})

	// Actor rejections return HTTP 200 with per-order error in the response body
	assert.Equal(t, HTTPStatusCode_200_OK, code)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)

	data, ok := resp.Response.(OrderDataResponse)
	require.True(t, ok)
	require.Len(t, data.Statuses, 1)
	require.NotNil(t, data.Statuses[0].Error)
}

func Test_handleTWAPOrder_MultipleOrdersRejected(t *testing.T) {
	// The REST handler itself rejects multiple orders before calling gRPC
	mock := &mockTradingTWAP{}
	payload := &ValidatedPlaceOrdersAction{
		Payload: &PlaceOrdersActionPayload{
			Action:   "placeOrders",
			Grouping: GroupingValues_twap,
			Orders: []snx_lib_api_json.PlaceOrderRequest{
				{Symbol: "BTC-USD", Side: "buy", OrderType: "twap", Quantity: Quantity("1.0"), DurationSeconds: 600},
				{Symbol: "ETH-USD", Side: "sell", OrderType: "twap", Quantity: Quantity("5.0"), DurationSeconds: 600},
			},
		},
	}
	ctx := newActionTradeContext(100, mock, payload)

	code, resp := Handle_placeOrders(ctx, HandlerParams{})

	assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
	require.NotNil(t, resp)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "exactly one order")
}
