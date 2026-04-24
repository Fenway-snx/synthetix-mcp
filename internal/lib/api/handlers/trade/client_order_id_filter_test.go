package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type clientOrderIDFilterSubaccountMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	openOrdersReq   *v4grpc.GetOpenOrdersRequest
	orderHistoryReq *v4grpc.GetOrderHistoryRequest
}

func (m *clientOrderIDFilterSubaccountMock) GetOpenOrders(ctx context.Context, req *v4grpc.GetOpenOrdersRequest, opts ...grpc.CallOption) (*v4grpc.GetOpenOrdersResponse, error) {
	m.openOrdersReq = req
	return &v4grpc.GetOpenOrdersResponse{}, nil
}

func (m *clientOrderIDFilterSubaccountMock) GetOrderHistory(ctx context.Context, req *v4grpc.GetOrderHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetOrderHistoryResponse, error) {
	m.orderHistoryReq = req
	return &v4grpc.GetOrderHistoryResponse{}, nil
}

func Test_QueryHandlers_RejectWhitespacePaddedClientOrderIdFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		handle func(TradeContext, HandlerParams) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any])
		params HandlerParams
	}{
		{
			name:   "getOpenOrders",
			handle: Handle_getOpenOrders,
			params: HandlerParams{"clientOrderId": "  padded-cloid  "},
		},
		{
			name:   "getOrderHistory",
			handle: Handle_getOrderHistory,
			params: HandlerParams{"clientOrderId": "  padded-cloid  "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &clientOrderIDFilterSubaccountMock{
				MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			}
			ctx := createTestTradeContextWithClient("req-1", 100, nil, mock)

			statusCode, resp := tt.handle(ctx, tt.params)
			require.NotNil(t, resp)

			assert.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
			assert.Equal(t, "error", resp.Status)
			require.NotNil(t, resp.Error)
			assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
			assert.Contains(t, resp.Error.Message, "leading or trailing whitespace")
		})
	}
}

func Test_QueryHandlers_ForwardCanonicalClientOrderIdFilters(t *testing.T) {
	t.Parallel()

	t.Run("getOpenOrders", func(t *testing.T) {
		t.Parallel()

		mock := &clientOrderIDFilterSubaccountMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("req-1", 100, nil, mock)

		statusCode, resp := Handle_getOpenOrders(ctx, HandlerParams{"clientOrderId": "alpha-1"})
		require.NotNil(t, resp)

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.openOrdersReq)
		assert.Equal(t, "alpha-1", mock.openOrdersReq.ClientOrderId)
	})

	t.Run("getOrderHistory", func(t *testing.T) {
		t.Parallel()

		mock := &clientOrderIDFilterSubaccountMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("req-2", 100, nil, mock)

		statusCode, resp := Handle_getOrderHistory(ctx, HandlerParams{"clientOrderId": "alpha-1"})
		require.NotNil(t, resp)

		assert.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.orderHistoryReq)
		assert.Equal(t, "alpha-1", mock.orderHistoryReq.ClientOrderId)
	})
}
