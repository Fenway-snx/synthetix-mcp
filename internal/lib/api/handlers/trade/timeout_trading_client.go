package trade

import (
	"context"
	"time"

	"google.golang.org/grpc"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type timeoutTradingClient struct {
	timeoutDuration time.Duration
	v4grpc.TradingServiceClient
}

// NewTimeoutTradingClient wraps client so each unary RPC uses requestTimeout.
// requestTimeout must be positive; otherwise returns nil.
func NewTimeoutTradingClient(client v4grpc.TradingServiceClient, requestTimeout time.Duration) v4grpc.TradingServiceClient {
	if client == nil || requestTimeout <= 0 {
		return nil
	}

	return &timeoutTradingClient{
		timeoutDuration:      requestTimeout,
		TradingServiceClient: client,
	}
}

func (c *timeoutTradingClient) CancelAllOrders(
	ctx context.Context,
	req *v4grpc.CancelAllOrdersRequest,
	opts ...grpc.CallOption,
) (*v4grpc.CancelAllOrdersResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.CancelAllOrders(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) CancelOrder(
	ctx context.Context,
	req *v4grpc.CancelOrderRequest,
	opts ...grpc.CallOption,
) (*v4grpc.CancelOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.CancelOrder(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) CancelOrderByCloid(
	ctx context.Context,
	req *v4grpc.CancelOrderByCloidRequest,
	opts ...grpc.CallOption,
) (*v4grpc.CancelOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.CancelOrderByCloid(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) ModifyOrder(
	ctx context.Context,
	req *v4grpc.ModifyOrderRequest,
	opts ...grpc.CallOption,
) (*v4grpc.ModifyOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.ModifyOrder(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) ModifyOrderByCloid(
	ctx context.Context,
	req *v4grpc.ModifyOrderByCloidRequest,
	opts ...grpc.CallOption,
) (*v4grpc.ModifyOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.ModifyOrderByCloid(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) PlaceBatchOrder(
	ctx context.Context,
	req *v4grpc.PlaceOrderRequest,
	opts ...grpc.CallOption,
) (*v4grpc.PlaceOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.PlaceBatchOrder(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) PlaceCompoundOrder(
	ctx context.Context,
	req *v4grpc.PlaceOrderRequest,
	opts ...grpc.CallOption,
) (*v4grpc.PlaceOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.PlaceCompoundOrder(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) PlacePositionTPAndSl(
	ctx context.Context,
	req *v4grpc.PlaceOrderRequest,
	opts ...grpc.CallOption,
) (*v4grpc.PlaceOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.PlacePositionTPAndSl(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) PlaceTWAPOrder(
	ctx context.Context,
	req *v4grpc.PlaceOrderRequest,
	opts ...grpc.CallOption,
) (*v4grpc.PlaceOrderResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.PlaceTWAPOrder(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) ScheduleCancel(
	ctx context.Context,
	req *v4grpc.ScheduleCancelRequest,
	opts ...grpc.CallOption,
) (*v4grpc.ScheduleCancelResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.ScheduleCancel(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) WithdrawCollateral(
	ctx context.Context,
	req *v4grpc.WithdrawCollateralRequest,
	opts ...grpc.CallOption,
) (*v4grpc.WithdrawCollateralResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.WithdrawCollateral(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) VoluntaryAutoExchange(
	ctx context.Context,
	req *v4grpc.VoluntaryAutoExchangeRequest,
	opts ...grpc.CallOption,
) (*v4grpc.VoluntaryAutoExchangeResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.VoluntaryAutoExchange(requestCtx, req, opts...)
}

func (c *timeoutTradingClient) GetWithdrawableAmounts(
	ctx context.Context,
	req *v4grpc.GetWithdrawableAmountsRequest,
	opts ...grpc.CallOption,
) (*v4grpc.GetWithdrawableAmountsResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	return c.TradingServiceClient.GetWithdrawableAmounts(requestCtx, req, opts...)
}
