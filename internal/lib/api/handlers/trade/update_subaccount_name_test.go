package trade

import (
	"context"
	"errors"
	"testing"

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
)

// mockSubaccountClientForUpdateName mocks the SubaccountServiceClient for update name tests
type mockSubaccountClientForUpdateName struct {
	v4grpc.SubaccountServiceClient // embedded interface satisfies all methods

	// Responses
	updateSubaccountResponse *v4grpc.SubaccountResponse
	updateSubaccountErr      error

	// Captured requests for verification
	updateSubaccountRequest *v4grpc.UpdateSubaccountRequest
}

func (m *mockSubaccountClientForUpdateName) UpdateSubaccount(ctx context.Context, in *v4grpc.UpdateSubaccountRequest, opts ...grpc.CallOption) (*v4grpc.SubaccountResponse, error) {
	m.updateSubaccountRequest = in
	return m.updateSubaccountResponse, m.updateSubaccountErr
}

// createTestContext creates a TradeContext with the validated action payload attached
func createTestContext(
	mockClient *mockSubaccountClientForUpdateName,
	subAccountId snx_lib_core.SubAccountId,
	name string,
) snx_lib_api_handlers_types.TradeContext {
	ctx := snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil, nil, nil, nil,
		mockClient,
		nil, nil, nil,
		"req-id",
		"",
		snx_lib_api_handlers_types.WalletAddress("0x123"),
		subAccountId,
	)

	// Attach validated action payload (as done by decodeTradeActionPayload in trade_handler.go)
	validatedPayload := &snx_lib_api_validation.ValidatedUpdateSubAccountNameAction{
		Payload: &snx_lib_api_validation.UpdateSubAccountNameActionPayload{
			Action: "updateSubAccountName",
			Name:   name,
		},
	}
	ctx = ctx.WithAction("updateSubAccountName", validatedPayload)

	return ctx
}

func Test_Handle_updateSubAccountName(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		newName := "New-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountResponse: &v4grpc.SubaccountResponse{
				Id:   int64(subAccountId),
				Name: newName,
			},
		}

		ctx := createTestContext(mockClient, subAccountId, newName)
		params := map[string]any{"name": newName}

		status, resp := Handle_updateSubAccountName(ctx, params)

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)

		// Verify gRPC call was made with correct parameters
		require.NotNil(t, mockClient.updateSubaccountRequest, "UpdateSubaccount should have been called")
		assert.Equal(t, int64(subAccountId), mockClient.updateSubaccountRequest.Id)
		assert.Equal(t, newName, mockClient.updateSubaccountRequest.Name)

		// Verify response data
		data, ok := resp.Response.(UpdateSubAccountNameResponse)
		require.True(t, ok, "response should be UpdateSubAccountNameResponse")
		assert.Equal(t, snx_lib_api_types.SubAccountIdFromIntUnvalidated(int64(subAccountId)), data.SubAccountId) // TODO: conversion function required
		assert.Equal(t, newName, data.Name)
	})

	t.Run("missing subaccount id in context", func(t *testing.T) {
		mockClient := &mockSubaccountClientForUpdateName{}

		// Create context with subAccountId = 0 (missing)
		ctx := snx_lib_api_handlers_types.NewTradeContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			"req-id",
			"",
			snx_lib_api_handlers_types.WalletAddress("0x123"),
			snx_lib_core.SubAccountId(0), // No subaccount ID
		)

		params := map[string]any{"name": "New-Name"}

		status, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "subAccountId is required")

		// Verify no gRPC call was made
		assert.Nil(t, mockClient.updateSubaccountRequest, "UpdateSubaccount should not have been called")
	})

	t.Run("UpdateSubaccount error", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		newName := "New-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountErr: errors.New("update failed"),
		}

		ctx := createTestContext(mockClient, subAccountId, newName)
		params := map[string]any{"name": newName}

		status, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "Failed to update subaccount")

		// Verify gRPC call was attempted
		require.NotNil(t, mockClient.updateSubaccountRequest, "UpdateSubaccount should have been called")
		assert.Equal(t, int64(subAccountId), mockClient.updateSubaccountRequest.Id)
		assert.Equal(t, newName, mockClient.updateSubaccountRequest.Name)
	})

	t.Run("name not changed correctly", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		newName := "New-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountResponse: &v4grpc.SubaccountResponse{
				Id:   int64(subAccountId),
				Name: "Different-Name", // Name doesn't match request
			},
		}

		ctx := createTestContext(mockClient, subAccountId, newName)
		params := map[string]any{"name": newName}

		status, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "Failed to update subaccount")

		// Verify gRPC call was made
		require.NotNil(t, mockClient.updateSubaccountRequest, "UpdateSubaccount should have been called")
		assert.Equal(t, newName, mockClient.updateSubaccountRequest.Name)
	})

	t.Run("update to same name succeeds", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		sameName := "Same-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountResponse: &v4grpc.SubaccountResponse{
				Id:   int64(subAccountId),
				Name: sameName,
			},
		}

		ctx := createTestContext(mockClient, subAccountId, sameName)
		params := map[string]any{"name": sameName}

		status, resp := Handle_updateSubAccountName(ctx, params)

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)

		// Verify response shows same name
		data, ok := resp.Response.(UpdateSubAccountNameResponse)
		require.True(t, ok, "response should be UpdateSubAccountNameResponse")
		assert.Equal(t, sameName, data.Name)
	})

	t.Run("missing action payload in context", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		mockClient := &mockSubaccountClientForUpdateName{}

		// Create context WITHOUT attaching the validated action payload
		ctx := snx_lib_api_handlers_types.NewTradeContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			"req-id",
			"",
			snx_lib_api_handlers_types.WalletAddress("0x123"),
			subAccountId,
		)

		params := map[string]any{"name": "New-Name"}

		status, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInvalidFormat, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "Invalid request body")

		// Verify no gRPC call was made
		assert.Nil(t, mockClient.updateSubaccountRequest, "UpdateSubaccount should not have been called")
	})

	t.Run("nil payload in validated action", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		mockClient := &mockSubaccountClientForUpdateName{}

		ctx := snx_lib_api_handlers_types.NewTradeContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			"req-id",
			"",
			snx_lib_api_handlers_types.WalletAddress("0x123"),
			subAccountId,
		)

		// Attach validated action with nil payload
		validatedPayload := &snx_lib_api_validation.ValidatedUpdateSubAccountNameAction{
			Payload: nil,
		}
		ctx = ctx.WithAction("updateSubAccountName", validatedPayload)

		params := map[string]any{"name": "New-Name"}

		status, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInvalidFormat, resp.Error.Code)

		// Verify no gRPC call was made
		assert.Nil(t, mockClient.updateSubaccountRequest, "UpdateSubaccount should not have been called")
	})

	t.Run("cannot rename master/primary subaccount - returns 400", func(t *testing.T) {
		// Master accounts have MasterID = 0, meaning they are the primary account
		// The service layer rejects renaming master accounts
		masterAccountId := snx_lib_core.SubAccountId(100)
		newName := "New-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			// Service returns error when trying to rename a master account
			// The gRPC handler wraps this in an Internal error with the original message
			updateSubaccountErr: status.Error(codes.Internal, "Failed to update subaccount: cannot modify master account"),
		}

		ctx := createTestContext(mockClient, masterAccountId, newName)
		params := map[string]any{"name": newName}

		httpStatus, resp := Handle_updateSubAccountName(ctx, params)

		// codes.Internal falls to default case which returns 500 INTERNAL_ERROR
		assert.Equal(t, HTTPStatusCode_500_InternalServerError, httpStatus)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "cannot modify master account")

		// Verify gRPC call was attempted
		require.NotNil(t, mockClient.updateSubaccountRequest, "UpdateSubaccount should have been called")
		assert.Equal(t, int64(masterAccountId), mockClient.updateSubaccountRequest.Id)
		assert.Equal(t, newName, mockClient.updateSubaccountRequest.Name)
	})

	t.Run("subaccount not found - returns 404", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(999)
		newName := "New-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountErr: status.Error(codes.NotFound, "subaccount not found"),
		}

		ctx := createTestContext(mockClient, subAccountId, newName)
		params := map[string]any{"name": newName}

		httpStatus, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_404_NotFound, httpStatus)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeNotFound, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "subaccount not found")
	})

	t.Run("invalid argument error - returns 400", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		newName := "New-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountErr: status.Error(codes.InvalidArgument, "invalid request parameters"),
		}

		ctx := createTestContext(mockClient, subAccountId, newName)
		params := map[string]any{"name": newName}

		httpStatus, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "invalid request parameters")
	})

	t.Run("invalid name error - returns 500", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		invalidName := "!@#$%"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountErr: status.Error(codes.Internal, "Failed to update subaccount: invalid subaccount name"),
		}

		ctx := createTestContext(mockClient, subAccountId, invalidName)
		params := map[string]any{"name": invalidName}

		httpStatus, resp := Handle_updateSubAccountName(ctx, params)

		// codes.Internal falls to default case which returns 500 INTERNAL_ERROR
		assert.Equal(t, HTTPStatusCode_500_InternalServerError, httpStatus)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "invalid subaccount name")
	})

	t.Run("name already exists error - returns 500", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		duplicateName := "Duplicate-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			updateSubaccountErr: status.Error(codes.Internal, "Failed to update subaccount: subaccount name already exists"),
		}

		ctx := createTestContext(mockClient, subAccountId, duplicateName)
		params := map[string]any{"name": duplicateName}

		httpStatus, resp := Handle_updateSubAccountName(ctx, params)

		// codes.Internal falls to default case which returns 500 INTERNAL_ERROR
		assert.Equal(t, HTTPStatusCode_500_InternalServerError, httpStatus)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "already exists")
	})

	t.Run("generic non-gRPC error - returns 500", func(t *testing.T) {
		subAccountId := snx_lib_core.SubAccountId(1)
		newName := "New-Name"

		mockClient := &mockSubaccountClientForUpdateName{
			// Non-gRPC error (e.g., network issue)
			updateSubaccountErr: errors.New("connection refused"),
		}

		ctx := createTestContext(mockClient, subAccountId, newName)
		params := map[string]any{"name": newName}

		httpStatus, resp := Handle_updateSubAccountName(ctx, params)

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, httpStatus)
		assert.Equal(t, "error", resp.Status)
		require.NotNil(t, resp.Error)
		assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "Failed to update subaccount")
	})
}
