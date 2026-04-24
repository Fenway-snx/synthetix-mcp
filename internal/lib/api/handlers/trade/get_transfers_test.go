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
	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// transfersMock returns a preconfigured GetTransfersResponse.
type transfersMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	resp *v4grpc.GetTransfersResponse
}

func (m *transfersMock) GetTransfers(_ context.Context, _ *v4grpc.GetTransfersRequest, _ ...grpc.CallOption) (*v4grpc.GetTransfersResponse, error) {
	return m.resp, nil
}

// failingTransfersMock returns an error from GetTransfers.
type failingTransfersMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	err error
}

func (m *failingTransfersMock) GetTransfers(_ context.Context, _ *v4grpc.GetTransfersRequest, _ ...grpc.CallOption) (*v4grpc.GetTransfersResponse, error) {
	return nil, m.err
}

// capturingTransfersMock captures the gRPC request for assertion.
type capturingTransfersMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	captured *v4grpc.GetTransfersRequest
}

func (m *capturingTransfersMock) GetTransfers(_ context.Context, req *v4grpc.GetTransfersRequest, _ ...grpc.CallOption) (*v4grpc.GetTransfersResponse, error) {
	m.captured = req
	return &v4grpc.GetTransfersResponse{}, nil
}

func Test_getTransfers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		now := timestamppb.Now()
		mock := &transfersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTransfersResponse{
				TotalCount: 1,
				Transfers: []*v4grpc.TransferHistoryItem{
					{
						TransferId:       5001,
						FromSubAccountId: 100,
						ToSubAccountId:   200,
						Symbol:           "USDT",
						Amount:           "50.5",
						TransferType:     "user",
						Status:           "success",
						TransferredAt:    now,
					},
				},
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(GetTransfersResponse)
		require.True(t, ok)
		assert.Equal(t, 1, data.Total)
		require.Len(t, data.Transfers, 1)

		item := data.Transfers[0]
		assert.Equal(t, "5001", item.TransferId)
		assert.Equal(t, SubAccountId("100"), item.From)
		assert.Equal(t, SubAccountId("200"), item.To)
		assert.Equal(t, Symbol("USDT"), item.Symbol)
		assert.Equal(t, "50.5", item.Amount)
		assert.Equal(t, "user", item.TransferType)
		assert.Equal(t, "success", item.Status)
		assert.Empty(t, item.ErrorMessage)
		assert.Greater(t, int64(item.Timestamp), int64(0))
	})

	t.Run("Success/EmptyResult", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)

		data, ok := resp.Response.(GetTransfersResponse)
		require.True(t, ok)
		assert.Equal(t, 0, data.Total)
		assert.Empty(t, data.Transfers)
	})

	t.Run("Success/WithSymbolFilter", func(t *testing.T) {
		mock := &capturingTransfersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, _ := Handle_getTransfers(ctx, map[string]any{
			"symbol": "USDT",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.Symbol)
		assert.Equal(t, "USDT", *mock.captured.Symbol)
	})

	t.Run("Success/TrimsSymbolFilter", func(t *testing.T) {
		mock := &capturingTransfersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, _ := Handle_getTransfers(ctx, map[string]any{
			"symbol": " USDT ",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.Symbol)
		assert.Equal(t, "USDT", *mock.captured.Symbol)
	})

	t.Run("Success/DefaultLimit", func(t *testing.T) {
		mock := &capturingTransfersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, _ := Handle_getTransfers(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		assert.Equal(t, int64(50), mock.captured.Limit)
	})

	t.Run("Success/WithTransferErrorMessage", func(t *testing.T) {
		mock := &transfersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetTransfersResponse{
				TotalCount: 1,
				Transfers: []*v4grpc.TransferHistoryItem{
					{
						TransferId:       9001,
						FromSubAccountId: 100,
						ToSubAccountId:   200,
						Symbol:           "USDT",
						Amount:           "10.0",
						TransferType:     "user",
						Status:           "failure",
						RequestId:        "req-err",
						ErrorMessage:     "insufficient balance",
						TransferredAt:    timestamppb.Now(),
					},
				},
			},
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		data, ok := resp.Response.(GetTransfersResponse)
		require.True(t, ok)
		require.Len(t, data.Transfers, 1)
		assert.Equal(t, "failure", data.Transfers[0].Status)
		assert.Equal(t, "insufficient balance", data.Transfers[0].ErrorMessage)
	})

	t.Run("MissingSubAccountId", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 0, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_status_codes.ErrorCodeValidationError, snx_lib_api_json.ErrorCode(resp.Error.Code))
		assert.Equal(t, "subAccountId is required", resp.Error.Message)
	})

	t.Run("InvalidRequestFormat", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{
			"limit": "not-a-number",
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_status_codes.ErrorCodeInvalidFormat, snx_lib_api_json.ErrorCode(resp.Error.Code))
	})

	t.Run("ValidationError/NegativeLimit", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{
			"limit": -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidationError/LimitExceedsMax", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{
			"limit": 1001,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidationError/NegativeOffset", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{
			"offset": -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidationError/InvalidCollateralSymbol", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{
			"symbol": "US-DT",
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "invalid symbol", resp.Error.Message)
	})

	t.Run("ValidationError/NegativeTimestamp", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{
			"startTime": -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("ValidationError/TimestampOutOfRange", func(t *testing.T) {
		ctx := createTestTradeContext("test-req", 100, nil)

		// Exceeds TimestampMaximumValidValue (year 2100) -- passes ValidateTimestampRange
		// but fails TimestampToTimestampPB inside APIStartEndToCoreStartEndPtrs.
		statusCode, resp := Handle_getTransfers(ctx, map[string]any{
			"startTime": 9999999999999,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCServiceError", func(t *testing.T) {
		mock := &failingTransfersMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         errors.New("connection refused"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_status_codes.ErrorCodeInternalError, snx_lib_api_json.ErrorCode(resp.Error.Code))
	})
}

type transfersTimeoutMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	err    error
	ctxErr error
}

func (m *transfersTimeoutMock) GetTransfers(ctx context.Context, _ *v4grpc.GetTransfersRequest, _ ...grpc.CallOption) (*v4grpc.GetTransfersResponse, error) {
	m.ctxErr = ctx.Err()
	return nil, m.err
}

func Test_getTransfers_timeout(t *testing.T) {
	t.Run("error - context deadline exceeded returns timeout error", func(t *testing.T) {
		mock := &transfersTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         context.DeadlineExceeded,
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

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
		mock := &transfersTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         status.Error(codes.DeadlineExceeded, "deadline exceeded"),
		}
		ctx := createTestTradeContextWithClient("test-req", 100, nil, mock)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

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

		mock := &transfersTimeoutMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         expiredCtx.Err(),
		}
		ctx := createTestTradeContextWithContext(expiredCtx, "test-req", 100, nil, mock)

		statusCode, resp := Handle_getTransfers(ctx, map[string]any{})

		assert.ErrorIs(t, mock.ctxErr, context.DeadlineExceeded)
		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.True(t, resp.HasTimeoutError())
	})
}
