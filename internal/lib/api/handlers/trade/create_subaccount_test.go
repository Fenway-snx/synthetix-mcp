package trade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type mockSubaccountClientForCreate struct {
	v4grpc.SubaccountServiceClient

	createSubaccountResponse *v4grpc.SubaccountResponse
	createSubaccountErr      error
	createSubaccountRequest  *v4grpc.CreateSubaccountRequest
}

func (m *mockSubaccountClientForCreate) CreateSubaccount(ctx context.Context, in *v4grpc.CreateSubaccountRequest, opts ...grpc.CallOption) (*v4grpc.SubaccountResponse, error) {
	m.createSubaccountRequest = in
	return m.createSubaccountResponse, m.createSubaccountErr
}

func createSubaccountTestContext(mockClient *mockSubaccountClientForCreate) snx_lib_api_handlers_types.TradeContext {
	return snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil, nil,
		mockClient,
		nil, nil, nil,
		"req-id",
		"client-req-id",
		snx_lib_api_handlers_types.WalletAddress("0x1234567890abcdef"),
		snx_lib_core.SubAccountId(0),
	)
}

func createSubaccountActionContext(
	t *testing.T,
	mockClient *mockSubaccountClientForCreate,
	name string,
) snx_lib_api_handlers_types.TradeContext {
	t.Helper()

	payload := &snx_lib_api_validation.CreateSubaccountActionPayload{
		Action: "createSubaccount",
		Name:   name,
	}
	validated, err := snx_lib_api_validation.NewValidatedCreateSubaccountAction(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	return createSubaccountTestContext(mockClient).WithAction("createSubaccount", validated)
}

func Test_Handle_createSubaccount_FailedPreconditionWithErrorInfo(t *testing.T) {
	st := status.New(codes.FailedPrecondition, "sub-account limit exceeded for current tier")
	detailed, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "MAX_SUB_ACCOUNTS_EXCEEDED",
		Metadata: map[string]string{
			"currentCount": "5",
			"maxAllowed":   "5",
			"tierName":     "Tier 1",
		},
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	mockClient := &mockSubaccountClientForCreate{
		createSubaccountErr: detailed.Err(),
	}

	ctx := createSubaccountActionContext(t, mockClient, "test-account")
	params := map[string]any{"name": "test-account"}

	httpStatus, resp := Handle_createSubaccount(ctx, params)

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeMaxSubAccountsExceeded, resp.Error.Code)
	assert.Equal(t, snx_lib_api_json.ErrorCategoryTrading, resp.Error.Category)
	assert.False(t, resp.Error.Retryable)
	require.NotNil(t, resp.Error.Details)
	assert.Equal(t, "5", resp.Error.Details["currentCount"])
	assert.Equal(t, "5", resp.Error.Details["maxAllowed"])
	assert.Equal(t, "Tier 1", resp.Error.Details["tierName"])
}

func Test_Handle_createSubaccount_FailedPreconditionWithoutErrorInfo(t *testing.T) {
	mockClient := &mockSubaccountClientForCreate{
		createSubaccountErr: status.Error(codes.FailedPrecondition, "some precondition failed"),
	}

	ctx := createSubaccountActionContext(t, mockClient, "test-account")
	params := map[string]any{"name": "test-account"}

	httpStatus, resp := Handle_createSubaccount(ctx, params)

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "some precondition failed")
}

func Test_Handle_createSubaccount_AlreadyExists(t *testing.T) {
	mockClient := &mockSubaccountClientForCreate{
		createSubaccountErr: status.Error(codes.AlreadyExists, "subaccount name already exists"),
	}

	ctx := createSubaccountActionContext(t, mockClient, "test-account")
	params := map[string]any{"name": "test-account"}

	httpStatus, resp := Handle_createSubaccount(ctx, params)

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "subaccount name already exists")
}

func Test_Handle_createSubaccount_InvalidArgument(t *testing.T) {
	mockClient := &mockSubaccountClientForCreate{
		createSubaccountErr: status.Error(codes.InvalidArgument, "invalid subaccount name"),
	}

	ctx := createSubaccountActionContext(t, mockClient, "test-account")
	params := map[string]any{"name": "test-account"}

	httpStatus, resp := Handle_createSubaccount(ctx, params)

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid subaccount name")
}

func Test_Handle_createSubaccount_Success(t *testing.T) {
	mockClient := &mockSubaccountClientForCreate{
		createSubaccountResponse: &v4grpc.SubaccountResponse{
			Id:                   42,
			SubAccountId:         1,
			Name:                 "my-account",
			AccountValue:         "0",
			AdjustedAccountValue: "0",
			MaxSubAccounts:       5,
		},
	}

	ctx := createSubaccountActionContext(t, mockClient, "my-account")
	params := map[string]any{"name": "my-account"}

	httpStatus, resp := Handle_createSubaccount(ctx, params)

	assert.Equal(t, HTTPStatusCode_200_OK, httpStatus)
	assert.Equal(t, "ok", resp.Status)
	require.NotNil(t, resp.Response)

	result, ok := resp.Response.(SubAccountResponse)
	require.True(t, ok)
	assert.Equal(t, "my-account", result.Name)
	assert.Equal(t, int64(5), result.AccountLimits.MaxSubAccounts)
}

func Test_Handle_createSubaccount_InternalError(t *testing.T) {
	mockClient := &mockSubaccountClientForCreate{
		createSubaccountErr: status.Error(codes.Internal, "database error"),
	}

	ctx := createSubaccountActionContext(t, mockClient, "test-account")
	params := map[string]any{"name": "test-account"}

	httpStatus, resp := Handle_createSubaccount(ctx, params)

	assert.Equal(t, HTTPStatusCode_500_InternalServerError, httpStatus)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
}

func Test_Handle_createSubaccount_InvalidWithoutValidatedAction(t *testing.T) {
	mockClient := &mockSubaccountClientForCreate{}

	ctx := createSubaccountTestContext(mockClient)
	params := map[string]any{"name": "test-account"}

	httpStatus, resp := Handle_createSubaccount(ctx, params)

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInvalidFormat, resp.Error.Code)
}
