package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

func newActionTradeContext(
	subAccountId snx_lib_core.SubAccountId,
	tradingClient v4grpc.TradingServiceClient,
	payload any,
) TradeContext {
	mock := snx_lib_authtest.NewMockSubaccountServiceClient()
	ctx := snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil,
		tradingClient,
		mock,
		nil, nil, nil,
		snx_lib_request.NewRequestID(),
		"test-req",
		"0x1234567890123456789012345678901234567890",
		subAccountId,
	)
	return ctx.WithAction("test", payload)
}

// ---------------------------------------------------------------------------
// placeOrders mocks
// ---------------------------------------------------------------------------

type mockTradingPlaceOrders struct {
	v4grpc.TradingServiceClient
	batchResp    *v4grpc.PlaceOrderResponse
	batchErr     error
	compoundResp *v4grpc.PlaceOrderResponse
	compoundErr  error
	positionResp *v4grpc.PlaceOrderResponse
	positionErr  error
}

func (m *mockTradingPlaceOrders) PlaceBatchOrder(_ context.Context, _ *v4grpc.PlaceOrderRequest, _ ...grpc.CallOption) (*v4grpc.PlaceOrderResponse, error) {
	return m.batchResp, m.batchErr
}

func (m *mockTradingPlaceOrders) PlaceCompoundOrder(_ context.Context, _ *v4grpc.PlaceOrderRequest, _ ...grpc.CallOption) (*v4grpc.PlaceOrderResponse, error) {
	return m.compoundResp, m.compoundErr
}

func (m *mockTradingPlaceOrders) PlacePositionTPAndSl(_ context.Context, _ *v4grpc.PlaceOrderRequest, _ ...grpc.CallOption) (*v4grpc.PlaceOrderResponse, error) {
	return m.positionResp, m.positionErr
}

// ---------------------------------------------------------------------------
// cancelOrders mock
// ---------------------------------------------------------------------------

type mockTradingCancelOrders struct {
	v4grpc.TradingServiceClient
	resp *v4grpc.CancelOrderResponse
	err  error
}

func (m *mockTradingCancelOrders) CancelOrder(_ context.Context, _ *v4grpc.CancelOrderRequest, _ ...grpc.CallOption) (*v4grpc.CancelOrderResponse, error) {
	return m.resp, m.err
}

// ---------------------------------------------------------------------------
// cancelAllOrders mock
// ---------------------------------------------------------------------------

type mockTradingCancelAllOrders struct {
	v4grpc.TradingServiceClient
	resp *v4grpc.CancelAllOrdersResponse
	err  error
}

func (m *mockTradingCancelAllOrders) CancelAllOrders(_ context.Context, _ *v4grpc.CancelAllOrdersRequest, _ ...grpc.CallOption) (*v4grpc.CancelAllOrdersResponse, error) {
	return m.resp, m.err
}

// ---------------------------------------------------------------------------
// scheduleCancel mock
// ---------------------------------------------------------------------------

type mockTradingScheduleCancel struct {
	v4grpc.TradingServiceClient
	resp *v4grpc.ScheduleCancelResponse
	err  error
}

func (m *mockTradingScheduleCancel) ScheduleCancel(_ context.Context, _ *v4grpc.ScheduleCancelRequest, _ ...grpc.CallOption) (*v4grpc.ScheduleCancelResponse, error) {
	return m.resp, m.err
}

// ---------------------------------------------------------------------------
// modifyOrder mock
// ---------------------------------------------------------------------------

type mockTradingModifyOrder struct {
	v4grpc.TradingServiceClient
	resp *v4grpc.ModifyOrderResponse
	err  error
}

func (m *mockTradingModifyOrder) ModifyOrder(_ context.Context, _ *v4grpc.ModifyOrderRequest, _ ...grpc.CallOption) (*v4grpc.ModifyOrderResponse, error) {
	return m.resp, m.err
}

// ---------------------------------------------------------------------------
// withdrawCollateral mock
// ---------------------------------------------------------------------------

type mockTradingWithdraw struct {
	v4grpc.TradingServiceClient
	resp *v4grpc.WithdrawCollateralResponse
	err  error
}

func (m *mockTradingWithdraw) WithdrawCollateral(_ context.Context, _ *v4grpc.WithdrawCollateralRequest, _ ...grpc.CallOption) (*v4grpc.WithdrawCollateralResponse, error) {
	return m.resp, m.err
}

// ===========================================================================
// Handle_placeOrders
// ===========================================================================

func Test_Handle_placeOrders(t *testing.T) {
	successResp := &v4grpc.PlaceOrderResponse{
		Orders: []*v4grpc.PlaceOrderResponseItem{
			{
				IsSuccess: true,
				Status:    v4grpc.OrderStatus_ACCEPTED,
				OrderId:   &v4grpc.OrderId{VenueId: 12345},
			},
		},
	}

	t.Run("MissingPayload", func(t *testing.T) {
		mock := &mockTradingPlaceOrders{}
		ctx := newActionTradeContext(100, mock, nil)

		code, resp := Handle_placeOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("InvalidPayloadType", func(t *testing.T) {
		mock := &mockTradingPlaceOrders{}
		ctx := newActionTradeContext(100, mock, "wrong-type")

		code, resp := Handle_placeOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidBatchOrder", func(t *testing.T) {
		mock := &mockTradingPlaceOrders{batchResp: successResp}
		validated := &ValidatedPlaceOrdersAction{
			Payload: &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Grouping: GroupingValues_na,
				Orders: []snx_lib_api_json.PlaceOrderRequest{
					{Symbol: "BTC-USD", Side: "buy", OrderType: "market", Quantity: Quantity("1.0")},
				},
			},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_placeOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(OrderDataResponse)
		require.True(t, ok)
		require.Len(t, data.Statuses, 1)
		require.NotNil(t, data.Statuses[0].Resting)
	})

	t.Run("ValidCompoundOrder", func(t *testing.T) {
		mock := &mockTradingPlaceOrders{
			compoundResp: &v4grpc.PlaceOrderResponse{
				Orders: []*v4grpc.PlaceOrderResponseItem{
					{IsSuccess: true, Status: v4grpc.OrderStatus_FILLED, OrderId: &v4grpc.OrderId{VenueId: 1}},
					{IsSuccess: true, Status: v4grpc.OrderStatus_ACCEPTED, OrderId: &v4grpc.OrderId{VenueId: 2}},
				},
			},
		}
		validated := &ValidatedPlaceOrdersAction{
			Payload: &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Grouping: GroupingValues_normalTpsl,
				Orders: []snx_lib_api_json.PlaceOrderRequest{
					{Symbol: "ETH-USD", Side: "buy", OrderType: "market", Quantity: Quantity("10")},
					{Symbol: "ETH-USD", Side: "sell", OrderType: "triggerTp", TriggerPrice: Price("5000"), Quantity: Quantity("10")},
				},
			},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_placeOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("ValidPositionTPSL", func(t *testing.T) {
		mock := &mockTradingPlaceOrders{
			positionResp: &v4grpc.PlaceOrderResponse{
				Orders: []*v4grpc.PlaceOrderResponseItem{
					{IsSuccess: true, Status: v4grpc.OrderStatus_ACCEPTED, OrderId: &v4grpc.OrderId{VenueId: 99}},
				},
			},
		}
		validated := &ValidatedPlaceOrdersAction{
			Payload: &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Grouping: GroupingValues_positionsTpsl,
				Orders: []snx_lib_api_json.PlaceOrderRequest{
					{Symbol: "BTC-USD", Side: "sell", OrderType: "triggerTp", TriggerPrice: Price("70000"), Quantity: Quantity("0.5")},
				},
			},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_placeOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("GRPCErrorOnPlaceBatchOrder", func(t *testing.T) {
		mock := &mockTradingPlaceOrders{batchErr: errors.New("connection refused")}
		validated := &ValidatedPlaceOrdersAction{
			Payload: &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Grouping: GroupingValues_na,
				Orders: []snx_lib_api_json.PlaceOrderRequest{
					{Symbol: "BTC-USD", Side: "buy", OrderType: "market", Quantity: Quantity("1.0")},
				},
			},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_placeOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})
}

// ===========================================================================
// Handle_cancelOrders
// ===========================================================================

func Test_Handle_cancelOrders(t *testing.T) {
	t.Run("MissingPayload", func(t *testing.T) {
		mock := &mockTradingCancelOrders{}
		ctx := newActionTradeContext(100, mock, nil)

		code, resp := Handle_cancelOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidCancel", func(t *testing.T) {
		mock := &mockTradingCancelOrders{
			resp: &v4grpc.CancelOrderResponse{
				Orders: []*v4grpc.CancelOrderResponseItem{
					{OrderId: &v4grpc.OrderId{VenueId: 111}, ErrorMessage: ""},
					{OrderId: &v4grpc.OrderId{VenueId: 222}, ErrorMessage: ""},
				},
			},
		}
		validated := &ValidatedCancelOrdersAction{
			VenueOrderIds: []snx_lib_api_types.VenueOrderId{"111", "222"},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_cancelOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(OrderDataResponse)
		require.True(t, ok)
		require.Len(t, data.Statuses, 2)
		require.NotNil(t, data.Statuses[0].Canceled)
		require.NotNil(t, data.Statuses[1].Canceled)
	})

	t.Run("GRPCError", func(t *testing.T) {
		mock := &mockTradingCancelOrders{err: errors.New("rpc failed")}
		validated := &ValidatedCancelOrdersAction{
			VenueOrderIds: []snx_lib_api_types.VenueOrderId{"111"},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_cancelOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})
}

// ===========================================================================
// Handle_cancelAllOrders
// ===========================================================================

func Test_Handle_cancelAllOrders(t *testing.T) {
	t.Run("MissingPayload", func(t *testing.T) {
		mock := &mockTradingCancelAllOrders{}
		ctx := newActionTradeContext(100, mock, nil)

		code, resp := Handle_cancelAllOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidCancelAll", func(t *testing.T) {
		mock := &mockTradingCancelAllOrders{
			resp: &v4grpc.CancelAllOrdersResponse{
				Orders: []*v4grpc.CancelOrderResponseItem{
					{OrderId: &v4grpc.OrderId{VenueId: 10}, Symbol: "BTC-USD"},
					{OrderId: &v4grpc.OrderId{VenueId: 20}, Symbol: "ETH-USD"},
				},
			},
		}
		validated := &ValidatedCancelAllOrdersAction{
			Symbols: []Symbol{"BTC-USD", "ETH-USD"},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_cancelAllOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.([]CancelAllOrdersResponseItem)
		require.True(t, ok)
		require.Len(t, data, 2)
		assert.Equal(t, "BTC-USD", string(*data[0].Symbol))
		assert.Equal(t, "ETH-USD", string(*data[1].Symbol))
	})

	t.Run("GRPCError", func(t *testing.T) {
		mock := &mockTradingCancelAllOrders{err: errors.New("rpc failed")}
		validated := &ValidatedCancelAllOrdersAction{
			Symbols: []Symbol{"BTC-USD"},
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_cancelAllOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})
}

// ===========================================================================
// Handle_scheduleCancel
// ===========================================================================

func Test_Handle_scheduleCancel(t *testing.T) {
	t.Run("MissingPayload", func(t *testing.T) {
		mock := &mockTradingScheduleCancel{}
		ctx := newActionTradeContext(100, mock, nil)

		code, resp := Handle_scheduleCancel(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidSchedule", func(t *testing.T) {
		triggerTimeMs := int64(1_700_000_000_000)
		mock := &mockTradingScheduleCancel{
			resp: &v4grpc.ScheduleCancelResponse{
				IsActive:       true,
				Message:        "dead-man-switch armed",
				TimeoutSeconds: 60,
				TriggerTime:    timestamppb.New(time.UnixMilli(triggerTimeMs)),
			},
		}
		validated := &ValidatedScheduleCancelAction{
			TimeoutSeconds: 60,
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_scheduleCancel(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(ScheduleCancelResponse)
		require.True(t, ok)
		assert.True(t, data.IsActive)
		assert.Equal(t, "dead-man-switch armed", data.Message)
		assert.Equal(t, int64(60), data.TimeoutSeconds)
		require.NotNil(t, data.TriggerTime)
		assert.Equal(t, Timestamp(triggerTimeMs), *data.TriggerTime)
	})

	t.Run("ValidClear", func(t *testing.T) {
		mock := &mockTradingScheduleCancel{
			resp: &v4grpc.ScheduleCancelResponse{
				IsActive:       false,
				Message:        "dead-man-switch disabled",
				TimeoutSeconds: 0,
			},
		}
		validated := &ValidatedScheduleCancelAction{
			TimeoutSeconds: 0,
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_scheduleCancel(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(ScheduleCancelResponse)
		require.True(t, ok)
		assert.False(t, data.IsActive)
		assert.Equal(t, "dead-man-switch disabled", data.Message)
		assert.Equal(t, int64(0), data.TimeoutSeconds)
		assert.Nil(t, data.TriggerTime)
	})

	t.Run("GRPCError", func(t *testing.T) {
		mock := &mockTradingScheduleCancel{err: errors.New("rpc failed")}
		validated := &ValidatedScheduleCancelAction{
			TimeoutSeconds: 60,
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_scheduleCancel(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})
}

// ===========================================================================
// Handle_modifyOrder
// ===========================================================================

func Test_Handle_modifyOrder(t *testing.T) {
	t.Run("MissingPayload", func(t *testing.T) {
		mock := &mockTradingModifyOrder{}
		ctx := newActionTradeContext(100, mock, nil)

		code, resp := Handle_modifyOrder(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidModifyWithPriceChange", func(t *testing.T) {
		mock := &mockTradingModifyOrder{
			resp: &v4grpc.ModifyOrderResponse{
				OrderId:  &v4grpc.OrderId{VenueId: 123},
				Status:   v4grpc.OrderStatus_ACCEPTED,
				Price:    snx_lib_utils_test.MakePointerOf("50000"),
				Quantity: snx_lib_utils_test.MakePointerOf("1.0"),
			},
		}
		validated := &ValidatedModifyOrderAction{
			Payload: &snx_lib_api_validation.ModifyOrderActionPayload{
				Action:       "modifyOrder",
				VenueOrderId: "123",
				Price:        snx_lib_utils_test.MakePointerOf(Price("50000")),
				Quantity:     snx_lib_utils_test.MakePointerOf(Quantity("1.0")),
			},
			VenueOrderId: "123",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_modifyOrder(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(ModifyOrderResponse)
		require.True(t, ok)
		assert.Equal(t, "accepted", data.Status)
		assert.Equal(t, Price("50000"), data.Price)
		assert.Equal(t, Quantity("1.0"), data.Quantity)
	})

	t.Run("GRPCError", func(t *testing.T) {
		mock := &mockTradingModifyOrder{err: errors.New("rpc failed")}
		validated := &ValidatedModifyOrderAction{
			Payload: &snx_lib_api_validation.ModifyOrderActionPayload{
				Action:       "modifyOrder",
				VenueOrderId: "123",
				Price:        snx_lib_utils_test.MakePointerOf(Price("50000")),
			},
			VenueOrderId: "123",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_modifyOrder(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})
}

// ===========================================================================
// Handle_withdrawCollateral
// ===========================================================================

func newWithdrawTradeContext(
	subAccountId snx_lib_core.SubAccountId,
	tradingClient v4grpc.TradingServiceClient,
) TradeContext {
	mock := snx_lib_authtest.NewMockSubaccountServiceClient()
	return snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil,
		tradingClient,
		mock,
		nil, nil, nil,
		snx_lib_request.NewRequestID(),
		"test-req",
		"0x1234567890123456789012345678901234567890",
		subAccountId,
	)
}

func callWithdrawCollateralWithValidatedPayload(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	payload, err := snx_lib_api_validation.DecodeWithdrawCollateralAction(params)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}

	validated, err := snx_lib_api_validation.NewValidatedWithdrawCollateralAction(payload)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	return Handle_withdrawCollateral(ctx.WithAction("withdrawCollateral", validated), params)
}

func Test_Handle_withdrawCollateral(t *testing.T) {
	validParams := HandlerParams{
		"symbol":      "USDC",
		"amount":      "100.5",
		"destination": "0x1234567890123456789012345678901234567890",
	}

	t.Run("Valid", func(t *testing.T) {
		mock := &mockTradingWithdraw{
			resp: &v4grpc.WithdrawCollateralResponse{
				Symbol:      "USDC",
				Amount:      "100.5",
				Destination: "0x1234567890123456789012345678901234567890",
			},
		}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, validParams)

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(WithdrawCollateralResponse)
		require.True(t, ok)
		assert.Equal(t, "USDC", string(data.Symbol))
		assert.Equal(t, "100.5", data.Amount)
		assert.Equal(t, WalletAddress("0x1234567890123456789012345678901234567890"), data.Destination)
	})

	t.Run("MissingSymbol", func(t *testing.T) {
		mock := &mockTradingWithdraw{}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, HandlerParams{
			"amount":      "100",
			"destination": "0x1234567890123456789012345678901234567890",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("InvalidAmountNotANumber", func(t *testing.T) {
		mock := &mockTradingWithdraw{}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, HandlerParams{
			"symbol":      "USDC",
			"amount":      "not-a-number",
			"destination": "0x1234567890123456789012345678901234567890",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("NegativeAmount", func(t *testing.T) {
		mock := &mockTradingWithdraw{}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, HandlerParams{
			"symbol":      "USDC",
			"amount":      "-50",
			"destination": "0x1234567890123456789012345678901234567890",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("InvalidDestinationAddress", func(t *testing.T) {
		mock := &mockTradingWithdraw{}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, HandlerParams{
			"symbol":      "USDC",
			"amount":      "100",
			"destination": "not-an-address",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCError/InvalidArgument", func(t *testing.T) {
		mock := &mockTradingWithdraw{err: status.Error(codes.InvalidArgument, "bad input")}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, validParams)

		assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCError/NotFound", func(t *testing.T) {
		mock := &mockTradingWithdraw{err: status.Error(codes.NotFound, "not found")}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, validParams)

		assert.Equal(t, HTTPStatusCode_404_NotFound, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCError/PermissionDenied", func(t *testing.T) {
		mock := &mockTradingWithdraw{err: status.Error(codes.PermissionDenied, "denied")}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, validParams)

		assert.Equal(t, HTTPStatusCode_403_Forbidden, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCError/Internal", func(t *testing.T) {
		mock := &mockTradingWithdraw{err: status.Error(codes.Internal, "db failure")}
		ctx := newWithdrawTradeContext(100, mock)

		code, resp := callWithdrawCollateralWithValidatedPayload(ctx, validParams)

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})
}
