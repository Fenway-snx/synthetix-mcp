package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

func Test_GetOrderHistoryResponseItem_JSON_MARSHALLING(t *testing.T) {

	t.Run("marshal - empty", func(t *testing.T) {

		v := GetOrderHistoryResponseItem{}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"order":{"venueId":""},"orderId":"","symbol":"","side":"","type":"","quantity":"","price":"","status":"","timeInForce":"","createdTime":0,"updateTime":0,"filledQuantity":"","filledPrice":"","triggeredByLiquidation":false,"reduceOnly":false,"postOnly":false}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)
	})

	t.Run("marshal - non-empty", func(t *testing.T) {

		v := GetOrderHistoryResponseItem{
			OrderId: OrderId{
				VenueId:  "456",
				ClientId: "cli-456",
			},
			DEPRECATED_VenueOrderId: "456",
			Symbol:                  "ETH-USD",
			Side:                    "BUY",
			Type:                    "LIMIT",
			Quantity:                Quantity("10.5"),
			Price:                   Price("3200.00"),
			Status:                  "FILLED",
			TimeInForce:             "GTC",
			CreatedTime:             1735680000000,
			UpdateTime:              1735689600000,
			FilledQuantity:          Quantity("10.5"),
			FilledPrice:             Price("3199.50"),
			TriggeredByLiquidation:  false,
			ReduceOnly:              true,
			PostOnly:                false,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"order":{"venueId":"456","clientId":"cli-456"},"orderId":"456","symbol":"ETH-USD","side":"BUY","type":"LIMIT","quantity":"10.5","price":"3200.00","status":"FILLED","timeInForce":"GTC","createdTime":1735680000000,"updateTime":1735689600000,"filledQuantity":"10.5","filledPrice":"3199.50","triggeredByLiquidation":false,"reduceOnly":true,"postOnly":false}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)

		var v2 GetOrderHistoryResponseItem
		err = json.Unmarshal(bytes, &v2)

		require.Nil(t, err)
		assert.Equal(t, v, v2)
	})

	t.Run("marshal - updateTime differs from createdTime", func(t *testing.T) {

		v := GetOrderHistoryResponseItem{
			OrderId: OrderId{
				VenueId:  "789",
				ClientId: "cli-789",
			},
			DEPRECATED_VenueOrderId: "789",
			Symbol:                  "BTC-USD",
			Side:                    "SELL",
			Type:                    "MARKET",
			Quantity:                Quantity("0.5"),
			Price:                   Price("42000.00"),
			Status:                  "PARTIALLY_FILLED",
			TimeInForce:             "IOC",
			CreatedTime:             1735680000000,
			UpdateTime:              1735690000000,
			FilledQuantity:          Quantity("0.3"),
			FilledPrice:             Price("41999.00"),
			TriggeredByLiquidation:  true,
			ReduceOnly:              false,
			PostOnly:                true,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		var parsed map[string]any
		err = json.Unmarshal(bytes, &parsed)

		require.Nil(t, err)
		assert.Equal(t, float64(1735680000000), parsed["createdTime"])
		assert.Equal(t, float64(1735690000000), parsed["updateTime"])
		assert.NotEqual(t, parsed["createdTime"], parsed["updateTime"])
	})
}

type orderHistoryTimeoutMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	resp   *v4grpc.GetOrderHistoryResponse
	err    error
	ctxErr error
}

func (m *orderHistoryTimeoutMock) GetOrderHistory(ctx context.Context, _ *v4grpc.GetOrderHistoryRequest, _ ...grpc.CallOption) (*v4grpc.GetOrderHistoryResponse, error) {
	m.ctxErr = ctx.Err()
	if m.resp != nil {
		return m.resp, nil
	}
	return nil, m.err
}

func createTestTradeContextWithContext(
	ctx context.Context,
	requestId string,
	subAccountId snx_lib_core.SubAccountId,
	validatedPayload any,
	client v4grpc.SubaccountServiceClient,
) TradeContext {
	return snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		ctx,
		nil, nil, nil, nil,
		client,
		nil, nil, nil,
		snx_lib_request.NewRequestID(),
		snx_lib_api_types.ClientRequestId(requestId),
		snx_lib_api_types.WalletAddress("0x1234567890123456789012345678901234567890"),
		subAccountId,
	).WithAction("getOrderHistory", validatedPayload)
}

func Test_GetOrderHistoryResponseItem_JSON_MARSHALLING_WithTriggerFields(t *testing.T) {
	t.Run("marshal - conditional order with trigger fields", func(t *testing.T) {
		v := GetOrderHistoryResponseItem{
			OrderId: OrderId{
				VenueId:  "789",
				ClientId: "cli-789",
			},
			DEPRECATED_VenueOrderId: "789",
			Symbol:                  "BTC-USD",
			Side:                    "BUY",
			Type:                    "STOP_MARKET",
			Quantity:                Quantity("1"),
			Price:                   Price("Market"),
			Status:                  "PLACED",
			TimeInForce:             "GTC",
			CreatedTime:             1735680000000,
			UpdateTime:              1735689600000,
			FilledQuantity:          Quantity("0"),
			FilledPrice:             Price("0"),
			TriggerPrice:            Price("47500"),
			TriggerPriceType:        "mark_price",
		}

		bytes, err := json.Marshal(v)
		require.Nil(t, err)

		var parsed map[string]any
		err = json.Unmarshal(bytes, &parsed)
		require.Nil(t, err)

		assert.Equal(t, "47500", parsed["triggerPrice"])
		assert.Equal(t, "mark_price", parsed["triggerPriceType"])
	})

	t.Run("marshal - non-conditional order omits trigger fields", func(t *testing.T) {
		v := GetOrderHistoryResponseItem{
			OrderId: OrderId{
				VenueId:  "790",
				ClientId: "cli-790",
			},
			DEPRECATED_VenueOrderId: "790",
			Symbol:                  "BTC-USD",
			Side:                    "BUY",
			Type:                    "LIMIT",
			Quantity:                Quantity("1"),
			Price:                   Price("50000"),
			Status:                  "PLACED",
			TimeInForce:             "GTC",
			CreatedTime:             1735680000000,
			UpdateTime:              1735689600000,
			FilledQuantity:          Quantity("0"),
			FilledPrice:             Price("0"),
		}

		bytes, err := json.Marshal(v)
		require.Nil(t, err)

		var parsed map[string]any
		err = json.Unmarshal(bytes, &parsed)
		require.Nil(t, err)

		_, hasTriggerPrice := parsed["triggerPrice"]
		_, hasTriggerPriceType := parsed["triggerPriceType"]
		assert.False(t, hasTriggerPrice)
		assert.False(t, hasTriggerPriceType)
	})
}

func Test_getOrderHistory_trigger_fields(t *testing.T) {
	triggerPrice := "47500"
	triggerPriceType := "mark_price"

	mock := &orderHistoryTimeoutMock{
		MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		resp: &v4grpc.GetOrderHistoryResponse{
			Orders: []*v4grpc.OrderHistoryItem{
				{
					OrderId:          &v4grpc.OrderId{VenueId: 1, ClientId: "cli-1"},
					Symbol:           "BTC-USD",
					Side:             "Long",
					Type:             "Stop Market",
					Quantity:         "1",
					Price:            "Market",
					Status:           "Placed",
					FilledQuantity:   "0",
					FilledPrice:      "0",
					TriggerPrice:     &triggerPrice,
					TriggerPriceType: &triggerPriceType,
				},
			},
		},
	}
	ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

	statusCode, resp := Handle_getOrderHistory(ctx, map[string]any{})

	require.Equal(t, HTTPStatusCode_200_OK, statusCode)
	require.NotNil(t, resp)

	bytes, err := json.Marshal(resp.Response)
	require.Nil(t, err)

	var items []map[string]any
	err = json.Unmarshal(bytes, &items)
	require.Nil(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, "47500", items[0]["triggerPrice"])
	assert.Equal(t, "mark_price", items[0]["triggerPriceType"])
}

func Test_getOrderHistory_timeout(t *testing.T) {
	t.Run("error - context deadline exceeded returns timeout error", func(t *testing.T) {
		mock := &orderHistoryTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         context.DeadlineExceeded,
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getOrderHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeRequestTimeout, resp.Error.Code)
		assert.Equal(t, "Request timed out", resp.Error.Message)
		assert.True(t, resp.Error.Retryable)
		assert.True(t, resp.HasTimeoutError())
	})

	t.Run("error - grpc DeadlineExceeded returns timeout error", func(t *testing.T) {
		mock := &orderHistoryTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         status.Error(codes.DeadlineExceeded, "deadline exceeded"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getOrderHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeRequestTimeout, resp.Error.Code)
		assert.Equal(t, "Request timed out", resp.Error.Message)
		assert.True(t, resp.Error.Retryable)
		assert.True(t, resp.HasTimeoutError())
	})

	t.Run("error - expired context propagates to grpc call", func(t *testing.T) {
		expiredCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		time.Sleep(5 * time.Millisecond)

		mock := &orderHistoryTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         expiredCtx.Err(),
		}
		ctx := createTestTradeContextWithContext(expiredCtx, "test-req", 100, nil, mock)

		statusCode, resp := Handle_getOrderHistory(ctx, map[string]any{})

		assert.ErrorIs(t, mock.ctxErr, context.DeadlineExceeded)
		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.True(t, resp.HasTimeoutError())
	})
}
