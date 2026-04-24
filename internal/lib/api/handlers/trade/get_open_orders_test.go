package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

func Test_OpenOrderRequest_JSON_MARSHALLING(t *testing.T) {

	t.Run("marshal", func(t *testing.T) {

		v := OpenOrderRequest{}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"symbol":"","side":"","type":"","status":"","limit":0,"offset":0}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)
	})

	t.Run("marshal - non-empty", func(t *testing.T) {

		v := OpenOrderRequest{
			Symbol: "BTC-USDT",
			Side:   "buy",
			Type:   "market",
			Status: "FILLED",
			Limit:  50,
			Offset: 0,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"symbol":"BTC-USDT","side":"buy","type":"market","status":"FILLED","limit":50,"offset":0}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)
	})
}

func Test_TWAPDetails_JSON_MARSHALLING(t *testing.T) {

	t.Run("marshal includes startedAtMs and totalDurationMs", func(t *testing.T) {
		v := TWAPDetails{
			AveragePrice:    "100.50",
			IntervalMs:      30000,
			TotalTrades:     48,
			TradesFilled:    4,
			TotalFees:       "1.25",
			StartedAtMs:     1775631898551,
			TotalDurationMs: 1440000,
		}

		bytes, err := json.Marshal(v)
		require.Nil(t, err)

		expected := `{"averagePrice":"100.50","intervalMs":30000,"totalTrades":48,"tradesFilled":4,"totalFees":"1.25","startedAtMs":1775631898551,"totalDurationMs":1440000}`
		assert.Equal(t, expected, string(bytes))

		var v2 TWAPDetails
		require.Nil(t, json.Unmarshal(bytes, &v2))
		assert.Equal(t, v, v2)
	})

	t.Run("zero values are present in JSON", func(t *testing.T) {
		v := TWAPDetails{}

		bytes, err := json.Marshal(v)
		require.Nil(t, err)

		expected := `{"averagePrice":"","intervalMs":0,"totalTrades":0,"tradesFilled":0,"totalFees":"","startedAtMs":0,"totalDurationMs":0}`
		assert.Equal(t, expected, string(bytes))
	})
}

func Test_OpenOrdersResponseItem_JSON_MARSHALLING(t *testing.T) {

	t.Run("marshal - empty", func(t *testing.T) {

		v := GetOpenOrdersResponseItem{}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		assert.Equal(t, `{"order":{"venueId":""},"orderId":"","symbol":"","side":"","type":"","quantity":"","price":"","triggerPrice":"","triggerPriceType":"","timeInForce":"","reduceOnly":false,"postOnly":false,"createdTime":0,"updatedTime":0,"filledQuantity":"","closePosition":false}`, string(bytes))

		// while we can marshal an empty symbol into an empty string, we do
		// not provide symmetrical behaviour - so the unmarshal will fail

		var v2 GetOpenOrdersResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Equal(t, "venue order id cannot be empty", err.Error())
		} else {

			assert.Fail(t, "should fail due to being unable to unmarshal empty symbol")
		}
	})

	t.Run("marshal - non-empty", func(t *testing.T) {

		v := GetOpenOrdersResponseItem{
			OrderId: OrderId{
				VenueId:  "123",
				ClientId: "cli-123",
			},
			DEPRECATED_VenueOrderId:      "123",
			Symbol:                       "ETH-USDT",
			Side:                         "sell",
			Type:                         "market",
			Quantity:                     Quantity("1234.5678"),
			Price:                        Price("999.999"),
			TriggerPrice:                 Price("999.888"),
			TriggerPriceType:             TriggerPriceType_mark,
			TimeInForce:                  "GTC",
			ReduceOnly:                   true,
			PostOnly:                     true,
			CreatedTime:                  1762568009123,
			UpdatedTime:                  1762568009123,
			FilledQuantity:               Quantity("1515"),
			TakeProfitOrderId:            nil,
			DEPRECATED_TakeProfitOrderId: "",
			StopLossOrderId: &OrderId{
				VenueId:  "132",
				ClientId: "",
			},
			DEPRECATED_StopLossOrderId: "132",
			ClosePosition:              true,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"order":{"venueId":"123","clientId":"cli-123"},"orderId":"123","symbol":"ETH-USDT","side":"sell","type":"market","quantity":"1234.5678","price":"999.999","triggerPrice":"999.888","triggerPriceType":"mark","timeInForce":"GTC","reduceOnly":true,"postOnly":true,"createdTime":1762568009123,"updatedTime":1762568009123,"filledQuantity":"1515","stopLossOrder":{"venueId":"132"},"stopLossOrderId":"132","closePosition":true}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)

		var v2 GetOpenOrdersResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			assert.Equal(t, v, v2)
		}

	})
}

type openOrdersMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	resp *v4grpc.GetOpenOrdersResponse
}

func (m *openOrdersMock) GetOpenOrders(_ context.Context, _ *v4grpc.GetOpenOrdersRequest, _ ...grpc.CallOption) (*v4grpc.GetOpenOrdersResponse, error) {
	return m.resp, nil
}

func Test_getOpenOrders_trigger_price_type(t *testing.T) {
	mock := &openOrdersMock{
		MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		resp: &v4grpc.GetOpenOrdersResponse{
			Orders: []*v4grpc.OpenOrderItem{
				{
					OrderId:           &v4grpc.OrderId{VenueId: 1, ClientId: "cli-1"},
					Symbol:            "BTC-USD",
					Side:              "buy",
					Type:              "limit",
					Quantity:          "1",
					RemainingQuantity: "1",
					Price:             "50000",
					TimeInForce:       v4grpc.TimeInForce_GTC,
					TriggerPrice:      "47500",
					TriggerPriceType:  "mark_price",
				},
				{
					OrderId:           &v4grpc.OrderId{VenueId: 2, ClientId: "cli-2"},
					Symbol:            "ETH-USD",
					Side:              "sell",
					Type:              "limit",
					Quantity:          "2",
					RemainingQuantity: "2",
					Price:             "3000",
					TimeInForce:       v4grpc.TimeInForce_GTC,
					TriggerPrice:      "3100",
					TriggerPriceType:  "last_price",
				},
			},
		},
	}
	ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

	statusCode, resp := Handle_getOpenOrders(ctx, map[string]any{})

	require.Equal(t, HTTPStatusCode_200_OK, statusCode)
	require.NotNil(t, resp)

	bytes, err := json.Marshal(resp.Response)
	require.Nil(t, err)

	var items []map[string]any
	require.Nil(t, json.Unmarshal(bytes, &items))
	require.Len(t, items, 2)

	assert.Equal(t, "mark_price", items[0]["triggerPriceType"])
	assert.Equal(t, "last_price", items[1]["triggerPriceType"])
}
