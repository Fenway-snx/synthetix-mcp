package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

func Test_Handle_transferCollateral(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     456,
			Symbol: "USDT",
			Amount: shopspring_decimal.RequireFromString("1000.50"),
		}

		ctx := createTestTradeContext("test-request-id", 123, validated)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{
			"action": "transferCollateral",
			"from":   "123",
			"to":     "456",
			"symbol": "USDT",
			"amount": "1000.50",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(TransferCollateralResponse)
		require.True(t, ok, "Response data should be TransferCollateralResponse")
		assert.Equal(t, string(ctx.ClientRequestId), data.RequestId) // Echoes back the client-provided ID
		assert.Equal(t, "918023717031270", data.TransferId)
		assert.Equal(t, "pending", data.Status)
		assert.Equal(t, Asset("USDT"), data.Symbol)
		assert.Equal(t, "1000.5", data.Amount)
		assert.Equal(t, "123", data.From.SubAccountId)
		assert.Equal(t, "900", data.From.Amount)
		assert.Equal(t, "456", data.To.SubAccountId)
		assert.Equal(t, "1000.5", data.To.Amount)
		assert.Greater(t, int64(data.TransferredAt), int64(0))
	})

	t.Run("InvalidPayloadInContext", func(t *testing.T) {
		ctx := createTestTradeContext("test-request-id", 123, "invalid-payload")

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{
			"action": "transferCollateral",
			"from":   "123",
			"to":     "456",
			"symbol": "USDT",
			"amount": "1000",
		})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
		assert.Equal(t, "Invalid request context", resp.Error.Message)
	})

	t.Run("LargeAmount", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     456,
			Symbol: "USDT",
			Amount: shopspring_decimal.RequireFromString("999999999999.123456789"),
		}

		ctx := createTestTradeContext("test-request-id", 123, validated)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{
			"action": "transferCollateral",
			"from":   "123",
			"to":     "456",
			"symbol": "USDT",
			"amount": "999999999999.123456789",
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(TransferCollateralResponse)
		require.True(t, ok)
		assert.Equal(t, "pending", data.Status)
	})

	t.Run("SameAccount", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     123, // Same as from (subAccountId)
			Symbol: "USDT",
			Amount: shopspring_decimal.RequireFromString("100"),
		}

		ctx := createTestTradeContext("test-request-id", 123, validated)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
		assert.Equal(t, "invalid request", resp.Error.Message)
	})

	t.Run("TransferFailure/InsufficientBalance", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     456,
			Symbol: "USDT",
			Amount: shopspring_decimal.RequireFromString("1000"),
		}

		mock := &failureTransferMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.TransferBetweenSubAccountsResponse{
				Status:       "failure",
				ErrorCode:    string(snx_lib_api_json.ErrorCodeInsufficientMargin),
				ErrorMessage: "requested transfer of 1000 exceeds withdrawable amount of 500",
			},
		}
		ctx := createTestTradeContextWithClient("test-request-id", 123, validated, mock)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInsufficientMargin, resp.Error.Code)
		assert.Equal(t, "requested transfer of 1000 exceeds withdrawable amount of 500", resp.Error.Message)
	})

	t.Run("TransferFailure/CollateralNotFound", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     456,
			Symbol: "INVALID",
			Amount: shopspring_decimal.RequireFromString("100"),
		}

		mock := &failureTransferMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.TransferBetweenSubAccountsResponse{
				Status:       "failure",
				ErrorCode:    string(snx_lib_api_json.ErrorCodeAssetNotFound),
				ErrorMessage: "collateral not found",
			},
		}
		ctx := createTestTradeContextWithClient("test-request-id", 123, validated, mock)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeAssetNotFound, resp.Error.Code)
		assert.Equal(t, "collateral not found", resp.Error.Message)
	})

	t.Run("TransferFailure/InternalError", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     456,
			Symbol: "USDT",
			Amount: shopspring_decimal.RequireFromString("100"),
		}

		mock := &failureTransferMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.TransferBetweenSubAccountsResponse{
				Status:       "failure",
				ErrorCode:    string(snx_lib_api_json.ErrorCodeInternalError),
				ErrorMessage: "failed to send to destination actor",
			},
		}
		ctx := createTestTradeContextWithClient("test-request-id", 123, validated, mock)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
	})

	t.Run("TransferFailure/OperationTimeout", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     456,
			Symbol: "USDT",
			Amount: shopspring_decimal.RequireFromString("100"),
		}

		mock := &failureTransferMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.TransferBetweenSubAccountsResponse{
				Status:       "failure",
				ErrorCode:    string(snx_lib_api_json.ErrorCodeOperationTimeout),
				ErrorMessage: "transfer timed out",
			},
		}
		ctx := createTestTradeContextWithClient("test-request-id", 123, validated, mock)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeOperationTimeout, resp.Error.Code)
	})

	t.Run("TransferFailure/DestinationNotFound", func(t *testing.T) {
		validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
			To:     456,
			Symbol: "USDT",
			Amount: shopspring_decimal.RequireFromString("100"),
		}

		mock := &failureTransferMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.TransferBetweenSubAccountsResponse{
				Status:       "failure",
				ErrorCode:    string(snx_lib_api_json.ErrorCodeNotFound),
				ErrorMessage: "destination account not loaded",
			},
		}
		ctx := createTestTradeContextWithClient("test-request-id", 123, validated, mock)

		statusCode, resp := Handle_transferCollateral(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeNotFound, resp.Error.Code)
	})

	t.Run("GRPCError/InvalidArgument", func(t *testing.T) {
		assertGRPCError(t, codes.InvalidArgument, "invalid amount",
			HTTPStatusCode_400_BadRequest, ErrorCodeValidationError, "invalid amount")
	})

	t.Run("GRPCError/NotFound", func(t *testing.T) {
		assertGRPCError(t, codes.NotFound, "sub account not found",
			HTTPStatusCode_404_NotFound, snx_lib_api_json.ErrorCodeNotFound, "sub account not found")
	})

	t.Run("GRPCError/PermissionDenied", func(t *testing.T) {
		assertGRPCError(t, codes.PermissionDenied, "not authorized",
			HTTPStatusCode_403_Forbidden, snx_lib_api_json.ErrorCodeForbidden, "not authorized")
	})

	t.Run("GRPCError/Internal", func(t *testing.T) {
		assertGRPCError(t, codes.Internal, "database error",
			HTTPStatusCode_500_InternalServerError, snx_lib_api_json.ErrorCodeInternalError, "database error")
	})

	t.Run("GRPCError/Unknown", func(t *testing.T) {
		assertGRPCError(t, codes.Unavailable, "service unavailable",
			HTTPStatusCode_500_InternalServerError, snx_lib_api_json.ErrorCodeInternalError, "Internal server error")
	})
}

func assertGRPCError(
	t *testing.T,
	grpcCode codes.Code,
	grpcMessage string,
	expectedHTTP HTTPStatusCode,
	expectedErrCode snx_lib_api_json.ErrorCode,
	expectedMessage string,
) {
	t.Helper()

	validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
		To:     456,
		Symbol: "USDT",
		Amount: shopspring_decimal.RequireFromString("100"),
	}

	mock := &failingTransferMock{
		MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		err:                         status.Error(grpcCode, grpcMessage),
	}
	ctx := createTestTradeContextWithClient("test-request-id", 123, validated, mock)

	statusCode, resp := Handle_transferCollateral(ctx, map[string]any{})

	require.Equal(t, expectedHTTP, statusCode)
	require.NotNil(t, resp)
	assert.Equal(t, "error", resp.Status)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, expectedErrCode, resp.Error.Code)
	assert.Equal(t, expectedMessage, resp.Error.Message)
}

// failingTransferMock embeds the real mock but overrides TransferBetweenSubAccounts to return an error.
type failingTransferMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	err error
}

func (m *failingTransferMock) TransferBetweenSubAccounts(ctx context.Context, req *v4grpc.TransferBetweenSubAccountsRequest, opts ...grpc.CallOption) (*v4grpc.TransferBetweenSubAccountsResponse, error) {
	return nil, m.err
}

// failureTransferMock returns a response with failure status (no gRPC error).
type failureTransferMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	resp *v4grpc.TransferBetweenSubAccountsResponse
}

func (m *failureTransferMock) TransferBetweenSubAccounts(ctx context.Context, req *v4grpc.TransferBetweenSubAccountsRequest, opts ...grpc.CallOption) (*v4grpc.TransferBetweenSubAccountsResponse, error) {
	return m.resp, nil
}

func createTestTradeContextWithClient(
	requestId string,
	subAccountId snx_lib_core.SubAccountId,
	validatedPayload any,
	client v4grpc.SubaccountServiceClient,
) TradeContext {
	logger := snx_lib_logging_doubles.NewStubLogger()

	ctx := snx_lib_api_handlers_types.NewTradeContext(
		logger,
		context.Background(),
		nil, // natsConn
		nil, // jetstream
		nil, // redis
		nil, // tradingClient
		client,
		nil, // authenticator
		nil, // whitelistArbitrator
		nil, // whitelistDiagnostics
		snx_lib_request.NewRequestID(),
		snx_lib_api_types.ClientRequestId(requestId),
		snx_lib_api_types.WalletAddress("0x1234567890123456789012345678901234567890"),
		subAccountId,
	)

	return ctx.WithAction("transferCollateral", validatedPayload)
}

func createTestTradeContext(
	requestId string,
	subAccountId snx_lib_core.SubAccountId,
	validatedPayload any,
) TradeContext {
	return createTestTradeContextWithClient(requestId, subAccountId, validatedPayload, snx_lib_authtest.NewMockSubaccountServiceClient())
}
