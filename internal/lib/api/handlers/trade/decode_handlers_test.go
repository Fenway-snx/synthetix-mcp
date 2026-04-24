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
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

func newDecodeTradeContext(
	subAccountId snx_lib_core.SubAccountId,
) TradeContext {
	mock := snx_lib_authtest.NewMockSubaccountServiceClient()
	return snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil, nil,
		mock,
		nil, nil, nil,
		snx_lib_request.NewRequestID(),
		"test-req",
		"0x1234567890123456789012345678901234567890",
		subAccountId,
	)
}

func newDecodeTradeContextWithClient(
	subAccountId snx_lib_core.SubAccountId,
	client v4grpc.SubaccountServiceClient,
) TradeContext {
	return snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil, nil,
		client,
		nil, nil, nil,
		snx_lib_request.NewRequestID(),
		"test-req",
		"0x1234567890123456789012345678901234567890",
		subAccountId,
	)
}

// --- gRPC mocks ---

type failingOpenOrdersMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
}

func (m *failingOpenOrdersMock) GetOpenOrders(ctx context.Context, req *v4grpc.GetOpenOrdersRequest, opts ...grpc.CallOption) (*v4grpc.GetOpenOrdersResponse, error) {
	return nil, errors.New("grpc unavailable")
}

// twapOpenOrdersMock returns a single TWAP order with the given execution state JSON.
type twapOpenOrdersMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	twapStateJSON string
}

func (m *twapOpenOrdersMock) GetOpenOrders(ctx context.Context, req *v4grpc.GetOpenOrdersRequest, opts ...grpc.CallOption) (*v4grpc.GetOpenOrdersResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()
	return &v4grpc.GetOpenOrdersResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Orders: []*v4grpc.OpenOrderItem{
			{
				OrderId:            &v4grpc.OrderId{VenueId: 1},
				Symbol:             "BTC-USD",
				Side:               "buy",
				Type:               "twap",
				Quantity:           "10",
				RemainingQuantity:  "8",
				TwapExecutionState: &m.twapStateJSON,
			},
		},
	}, nil
}

type failingTradeHistoryMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
}

func (m *failingTradeHistoryMock) GetTradeHistory(ctx context.Context, req *v4grpc.GetTradeHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetTradeHistoryResponse, error) {
	return nil, errors.New("grpc unavailable")
}

// tradeHistoryWithOrderTypeMock returns one trade with a representative order_type string.
type tradeHistoryWithOrderTypeMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
}

func (m *tradeHistoryWithOrderTypeMock) GetTradeHistory(ctx context.Context, req *v4grpc.GetTradeHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetTradeHistoryResponse, error) {
	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	return &v4grpc.GetTradeHistoryResponse{
		TimestampMs: timestampMs,
		TimestampUs: timestampUs,
		Trades: []*v4grpc.TradeHistoryItem{
			{
				Id:             1,
				Symbol:         "BTC-USD",
				OrderType:      "Limit",
				Direction:      "Open Long",
				FilledPrice:    "100",
				FilledQuantity: "1",
				FillType:       snx_lib_core.FillTypeMaker,
				TradedAt:       timestamppb.New(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		TotalCount: 1,
		HasMore:    false,
	}, nil
}

type failingUpdateLeverageMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
}

func (m *failingUpdateLeverageMock) UpdateSubAccountMarketLeverage(ctx context.Context, req *v4grpc.UpdateSubAccountMarketLeverageRequest, opts ...grpc.CallOption) (*v4grpc.UpdateSubAccountMarketLeverageResponse, error) {
	return nil, errors.New("grpc unavailable")
}

func callUpdateLeverageWithValidatedPayload(
	ctx TradeContext,
	params map[string]any,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	payload, err := snx_lib_api_validation.DecodeUpdateLeverageAction(params)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}

	validated := &ValidatedUpdateLeverageAction{Payload: payload}

	return Handle_updateLeverage(ctx.WithAction("updateLeverage", validated), params)
}

func callCreateSubaccountWithValidatedPayload(
	ctx TradeContext,
	params map[string]any,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	payload, err := snx_lib_api_validation.DecodeCreateSubaccountAction(params)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}

	validated := &snx_lib_api_validation.ValidatedCreateSubaccountAction{Payload: payload}

	return Handle_createSubaccount(ctx.WithAction("createSubaccount", validated), params)
}

// --- Tests ---

func Test_Decode_Handle_getOpenOrders(t *testing.T) {
	t.Run("valid minimal - empty params defaults limit to 50", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOpenOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("all filters with float64 values", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOpenOrders(ctx, HandlerParams{
			"symbol": "BTC-USD",
			"side":   "buy",
			"limit":  25.0,
			"offset": 10.0,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("limit and offset as float64 coerced to int32", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOpenOrders(ctx, HandlerParams{
			"limit":  100.0,
			"offset": 5.0,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("gRPC error returns 500", func(t *testing.T) {
		mock := &failingOpenOrdersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := newDecodeTradeContextWithClient(100, mock)
		statusCode, _ := Handle_getOpenOrders(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
	})

	t.Run("TWAP order includes startedAtMs and totalDurationMs", func(t *testing.T) {
		twapJSON := `{
			"order_id":{"void":1},
			"sub_account_id":100,
			"chunk_interval_ms":30000,
			"chunk_quantity":"0.5",
			"chunks_total":48,
			"chunks_filled":4,
			"started_at_ms":1775631898551,
			"total_fees":"1.25",
			"filled_notional":"200",
			"quantity_filled":"2",
			"symbol":"BTC-USD",
			"side":1
		}`
		mock := &twapOpenOrdersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			twapStateJSON:               twapJSON,
		}
		ctx := newDecodeTradeContextWithClient(100, mock)
		statusCode, resp := Handle_getOpenOrders(ctx, HandlerParams{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.Equal(t, "ok", resp.Status)

		// Extract twapDetails from the response
		orders, ok := resp.Response.(GetOpenOrdersResponse)
		require.True(t, ok)
		require.Len(t, orders, 1)
		require.NotNil(t, orders[0].TwapDetails)

		td := orders[0].TwapDetails
		assert.Equal(t, int64(1775631898551), td.StartedAtMs)
		assert.Equal(t, int64(48*30000), td.TotalDurationMs)
		assert.Equal(t, int64(30000), td.IntervalMs)
		assert.Equal(t, 48, td.TotalTrades)
		assert.Equal(t, 4, td.TradesFilled)
		assert.Equal(t, "1.25", td.TotalFees)
		assert.Equal(t, Price("100"), td.AveragePrice) // 200/2 = 100
	})
}

func Test_Decode_Handle_getOrderHistory(t *testing.T) {
	now := snx_lib_utils_time.Now()
	recentMs := float64(now.Add(-1 * time.Hour).UnixMilli())
	nowMs := float64(now.UnixMilli())

	t.Run("valid request with deprecated fromTime/toTime", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"fromTime": recentMs,
			"toTime":   nowMs,
			"limit":    50.0,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("valid request with startTime/endTime", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"startTime": recentMs,
			"endTime":   nowMs,
			"limit":     50.0,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("startTime and fromTime with same value is accepted", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"startTime": recentMs,
			"fromTime":  recentMs,
			"endTime":   nowMs,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("conflicting startTime and fromTime returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"startTime": recentMs,
			"fromTime":  nowMs,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("conflicting endTime and toTime returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"endTime": recentMs,
			"toTime":  nowMs,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("fromTime > toTime returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"fromTime": nowMs,
			"toTime":   recentMs,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("startTime > endTime returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"startTime": nowMs,
			"endTime":   recentMs,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("range > 7 days returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		eightDaysAgoMs := float64(now.Add(-8 * 24 * time.Hour).UnixMilli())
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"fromTime": eightDaysAgoMs,
			"toTime":   nowMs,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("statusFilter as []any coerced to []string", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{
			"status": []any{"FILLED", "CANCELLED"},
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})
}

func Test_Decode_Handle_getPositions(t *testing.T) {
	now := snx_lib_utils_time.Now()
	recentMs := float64(now.Add(-1 * time.Hour).UnixMilli())
	nowMs := float64(now.UnixMilli())

	t.Run("valid minimal - empty params", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPositions(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("limit and offset as float64", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPositions(ctx, map[string]any{
			"limit":  20.0,
			"offset": 5.0,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("deprecated fromTime/toTime as float64", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPositions(ctx, map[string]any{
			"fromTime": recentMs,
			"toTime":   nowMs,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("startTime/endTime as float64", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPositions(ctx, map[string]any{
			"startTime": recentMs,
			"endTime":   nowMs,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("conflicting startTime and fromTime returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPositions(ctx, map[string]any{
			"startTime": recentMs,
			"fromTime":  nowMs,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})
}

func Test_Decode_Handle_getTrades(t *testing.T) {
	t.Run("valid minimal - empty params defaults limit to 100", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getTrades(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("all fields with float64", func(t *testing.T) {
		now := snx_lib_utils_time.Now()
		startMs := float64(now.Add(-1 * time.Hour).UnixMilli())
		endMs := float64(now.UnixMilli())

		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getTrades(ctx, map[string]any{
			"symbol":    "ETH-USD",
			"limit":     50.0,
			"offset":    10.0,
			"startTime": startMs,
			"endTime":   endMs,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("negative limit returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getTrades(ctx, map[string]any{
			"limit": -1.0,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("limit > 1000 returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getTrades(ctx, map[string]any{
			"limit": 1001.0,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("gRPC error returns 500", func(t *testing.T) {
		mock := &failingTradeHistoryMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := newDecodeTradeContextWithClient(100, mock)
		statusCode, _ := Handle_getTrades(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
	})

	t.Run("GetTradesResponseItem OrderType matches gRPC order_type", func(t *testing.T) {
		mock := &tradeHistoryWithOrderTypeMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := newDecodeTradeContextWithClient(100, mock)
		statusCode, resp := Handle_getTrades(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp.Response)
		got, ok := resp.Response.(GetTradesResponse)
		require.True(t, ok)
		require.Len(t, got.Trades, 1)
		var wantOrderType snx_lib_api_types.OrderType = "Limit"
		assert.Equal(t, wantOrderType, got.Trades[0].OrderType)
		assert.Equal(t, Symbol("BTC-USD"), got.Trades[0].Symbol)
		assert.Equal(t, "buy", got.Trades[0].Side)
	})
}

func Test_Decode_Handle_getBalanceUpdates(t *testing.T) {
	t.Run("valid with float64 limit and offset", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{
			"limit":  50.0,
			"offset": 0.0,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("SelectedAccountId == 0 returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(0)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("valid actionFilter DEPOSIT", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{
			"actionFilter": "DEPOSIT",
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("valid startTime and endTime as float64 milliseconds", func(t *testing.T) {
		now := snx_lib_utils_time.Now()
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{
			"startTime": float64(now.Add(-2 * time.Hour).UnixMilli()),
			"endTime":   float64(now.UnixMilli()),
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})
}

func Test_Decode_Handle_getFundingPayments(t *testing.T) {
	t.Run("valid minimal with SelectedAccountId > 0", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("SelectedAccountId == 0 returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(0)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("limit as float64 coercion", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{
			"limit": 25.0,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("timestamps as float64", func(t *testing.T) {
		now := snx_lib_utils_time.Now()
		startMs := float64(now.Add(-1 * time.Hour).UnixMilli())
		endMs := float64(now.UnixMilli())

		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{
			"startTime": startMs,
			"endTime":   endMs,
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("startTime >= endTime returns 400", func(t *testing.T) {
		now := snx_lib_utils_time.Now()
		nowMs := float64(now.UnixMilli())
		earlierMs := float64(now.Add(-1 * time.Hour).UnixMilli())

		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{
			"startTime": nowMs,
			"endTime":   earlierMs,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("negative limit returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{
			"limit": -5.0,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})
}

func Test_Decode_Handle_updateLeverage(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := callUpdateLeverageWithValidatedPayload(ctx, map[string]any{
			"symbol":   "BTC-USD",
			"leverage": "10",
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("missing symbol returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := callUpdateLeverageWithValidatedPayload(ctx, map[string]any{
			"leverage": "10",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("invalid leverage returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := callUpdateLeverageWithValidatedPayload(ctx, map[string]any{
			"symbol":   "BTC-USD",
			"leverage": "abc",
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("gRPC error returns 500", func(t *testing.T) {
		mock := &failingUpdateLeverageMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := newDecodeTradeContextWithClient(100, mock)
		statusCode, _ := callUpdateLeverageWithValidatedPayload(ctx, map[string]any{
			"symbol":   "BTC-USD",
			"leverage": "10",
		})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
	})
}

func Test_Decode_Handle_createSubaccount(t *testing.T) {
	t.Run("valid with name", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := callCreateSubaccountWithValidatedPayload(ctx, map[string]any{
			"name": "My Sub",
		})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("empty name is valid - name is optional", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := callCreateSubaccountWithValidatedPayload(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})
}

func Test_Decode_Handle_getPerformanceHistory(t *testing.T) {
	t.Run("no params defaults period to day", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPerformanceHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(PerformanceHistoryResponse)
		require.True(t, ok)
		assert.Equal(t, "day", data.Period)
	})

	t.Run("period=week", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPerformanceHistory(ctx, map[string]any{
			"period": "week",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(PerformanceHistoryResponse)
		require.True(t, ok)
		assert.Equal(t, "week", data.Period)
	})

	t.Run("period as non-string falls back to day", func(t *testing.T) {
		ctx := newDecodeTradeContext(100)
		statusCode, resp := Handle_getPerformanceHistory(ctx, map[string]any{
			"period": 123,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(PerformanceHistoryResponse)
		require.True(t, ok)
		assert.Equal(t, "day", data.Period)
	})

	t.Run("SelectedAccountId == 0 returns 400", func(t *testing.T) {
		ctx := newDecodeTradeContext(0)
		statusCode, resp := Handle_getPerformanceHistory(ctx, map[string]any{})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})
}
