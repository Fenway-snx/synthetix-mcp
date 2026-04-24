package trade

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type timeoutTradingClientGetWithdrawableAmountsMock struct {
	v4grpc.TradingServiceClient
	ctx context.Context
}

func (m *timeoutTradingClientGetWithdrawableAmountsMock) GetWithdrawableAmounts(
	ctx context.Context,
	_ *v4grpc.GetWithdrawableAmountsRequest,
	_ ...grpc.CallOption,
) (*v4grpc.GetWithdrawableAmountsResponse, error) {
	m.ctx = ctx
	return &v4grpc.GetWithdrawableAmountsResponse{}, nil
}

func Test_TimeoutTradingClient_GetWithdrawableAmountsUsesRequestTimeout(t *testing.T) {
	mock := &timeoutTradingClientGetWithdrawableAmountsMock{}
	reqTimeout := 4000 * time.Millisecond
	client := NewTimeoutTradingClient(mock, reqTimeout)

	_, err := client.GetWithdrawableAmounts(context.Background(), &v4grpc.GetWithdrawableAmountsRequest{
		SubAccountId: 1,
		Symbols:      []string{"BTC"},
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, mock.ctx)

	deadline, ok := mock.ctx.Deadline()
	require.True(t, ok)
	remaining := time.Until(deadline)

	assert.LessOrEqual(t, remaining, reqTimeout)
	assert.Greater(t, remaining, reqTimeout-500*time.Millisecond)
}

type timeoutTradingClientScheduleCancelMock struct {
	v4grpc.TradingServiceClient
	ctx context.Context
}

func (m *timeoutTradingClientScheduleCancelMock) ScheduleCancel(
	ctx context.Context,
	_ *v4grpc.ScheduleCancelRequest,
	_ ...grpc.CallOption,
) (*v4grpc.ScheduleCancelResponse, error) {
	m.ctx = ctx
	return &v4grpc.ScheduleCancelResponse{}, nil
}

func Test_TimeoutTradingClient_ScheduleCancelUsesRequestTimeout(t *testing.T) {
	mock := &timeoutTradingClientScheduleCancelMock{}
	reqTimeout := 4000 * time.Millisecond
	client := NewTimeoutTradingClient(mock, reqTimeout)

	_, err := client.ScheduleCancel(context.Background(), &v4grpc.ScheduleCancelRequest{
		SubAccountId:   1,
		RequestId:      "req-1",
		TimeoutSeconds: 60,
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, mock.ctx)

	deadline, ok := mock.ctx.Deadline()
	require.True(t, ok)
	remaining := time.Until(deadline)

	assert.LessOrEqual(t, remaining, reqTimeout)
	assert.Greater(t, remaining, reqTimeout-500*time.Millisecond)
}
