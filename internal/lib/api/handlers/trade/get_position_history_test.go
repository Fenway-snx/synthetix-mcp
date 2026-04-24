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

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type capturingPositionHistoryMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	captured *v4grpc.GetPositionHistoryRequest
	resp     *v4grpc.GetPositionHistoryResponse
	err      error
}

func (m *capturingPositionHistoryMock) GetPositionHistory(_ context.Context, req *v4grpc.GetPositionHistoryRequest, _ ...grpc.CallOption) (*v4grpc.GetPositionHistoryResponse, error) {
	m.captured = req
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil {
		return m.resp, nil
	}
	return &v4grpc.GetPositionHistoryResponse{}, nil
}

func Test_getPositionHistory(t *testing.T) {
	t.Run("success - defaults limit to 100 and offset to 0", func(t *testing.T) {
		mock := &capturingPositionHistoryMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.Limit)
		require.NotNil(t, mock.captured.Offset)
		assert.Equal(t, int32(100), *mock.captured.Limit)
		assert.Equal(t, int32(0), *mock.captured.Offset)
		assert.Equal(t, int64(100), mock.captured.SubAccountId)
	})

	t.Run("success - forwards symbol and pagination params", func(t *testing.T) {
		mock := &capturingPositionHistoryMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{
			"symbol": "BTC-USDT",
			"limit":  10,
			"offset": 2,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.Symbol)
		assert.Equal(t, "BTC-USDT", *mock.captured.Symbol)
		assert.Equal(t, int32(10), *mock.captured.Limit)
		assert.Equal(t, int32(2), *mock.captured.Offset)
	})

	t.Run("success - maps grpc response fields", func(t *testing.T) {
		now := timestamppb.Now()
		mock := &capturingPositionHistoryMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetPositionHistoryResponse{
				Positions: []*v4grpc.PositionHistoryItem{
					{
						PositionId:      12345,
						Symbol:          "ETH-USDT",
						Side:            "long",
						EntryPrice:      "3000",
						Quantity:        "1.5",
						ClosePrice:      "3100",
						CloseReason:     "close",
						RealizedPnl:     "150",
						AccumulatedFees: "2.5",
						NetFundingPnl:   "0.25",
						ClosedAt:        now,
						CreatedAt:       now,
						TradeId:         67890,
					},
				},
				HasMore: true,
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		data, ok := resp.Response.(GetPositionHistoryResponse)
		require.True(t, ok)
		require.Len(t, data.Positions, 1)
		assert.True(t, data.HasMore)
		assert.Equal(t, "12345", data.Positions[0].PositionId)
		assert.Equal(t, "ETH-USDT", string(data.Positions[0].Symbol))
		assert.Equal(t, "0.25", data.Positions[0].NetFunding)
		assert.Equal(t, TradeId("67890"), data.Positions[0].TradeId)
	})

	t.Run("validation error - missing subaccountId", func(t *testing.T) {
		mock := &capturingPositionHistoryMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 0, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "subaccountId is required")
		assert.Nil(t, mock.captured, "gRPC should not be called when subaccountId is missing")
	})

	t.Run("validation error - negative limit", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{
			"limit": -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "non-negative")
	})

	t.Run("validation error - negative offset", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{
			"offset": -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "non-negative")
	})

	t.Run("validation error - limit exceeds max", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{
			"limit": 1001,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Contains(t, resp.Error.Message, "cannot exceed")
	})

	t.Run("error - grpc InvalidArgument returns 400", func(t *testing.T) {
		mock := &capturingPositionHistoryMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         status.Error(codes.InvalidArgument, "offset exceeds maximum"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "Invalid request parameters", resp.Error.Message)
	})

	t.Run("error - grpc internal failure returns 500", func(t *testing.T) {
		mock := &capturingPositionHistoryMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         errors.New("subaccount unavailable"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "Failed to retrieve position history", resp.Error.Message)
	})
}

type positionHistoryTimeoutMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	err    error
	ctxErr error
}

func (m *positionHistoryTimeoutMock) GetPositionHistory(ctx context.Context, _ *v4grpc.GetPositionHistoryRequest, _ ...grpc.CallOption) (*v4grpc.GetPositionHistoryResponse, error) {
	m.ctxErr = ctx.Err()
	return nil, m.err
}

func Test_getPositionHistory_timeout(t *testing.T) {
	t.Run("error - context deadline exceeded returns timeout error", func(t *testing.T) {
		mock := &positionHistoryTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         context.DeadlineExceeded,
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

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
		mock := &positionHistoryTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         status.Error(codes.DeadlineExceeded, "deadline exceeded"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

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

		mock := &positionHistoryTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         expiredCtx.Err(),
		}
		ctx := createTestTradeContextWithContext(expiredCtx, "test-req", 100, nil, mock)

		statusCode, resp := Handle_getPositionHistory(ctx, map[string]any{})

		assert.ErrorIs(t, mock.ctxErr, context.DeadlineExceeded)
		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.True(t, resp.HasTimeoutError())
	})
}
