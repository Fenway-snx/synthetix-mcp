package trade

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type UpdateSubAccountNameResponse struct {
	SubAccountId SubAccountId `json:"subAccountId"`
	Name         string       `json:"name"`
}

// Handler for "updateSubAccountName".
//
//dd:span
func Handle_updateSubAccountName(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Validate that a subAccountId was provided and authenticated
	if ctx.SelectedAccountId == 0 {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subAccountId is required", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Use already-validated payload from context (validated in decodeTradeActionPayload)
	validated, ok := ctx.ActionPayload().(*snx_lib_api_validation.ValidatedUpdateSubAccountNameAction)
	if !ok || validated == nil || validated.Payload == nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	name := validated.Payload.Name

	// Use the authenticated subaccount ID from context
	subAccountId := ctx.SelectedAccountId

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Update the subaccount name
	grpcResp, err := ctx.SubaccountClient.UpdateSubaccount(ctx.Context, &v4grpc.UpdateSubaccountRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Id:          int64(subAccountId),
		Name:        name,
	})
	if err != nil {
		ctx.Logger.Error("Failed to update subaccount",
			"error", err,
			"name", name,
			"sub_account_id", subAccountId,
		)

		// Parse gRPC status code to return appropriate HTTP error
		grpcStatus, ok := status.FromError(err)
		if ok {
			errMsg := grpcStatus.Message()
			switch grpcStatus.Code() {
			case codes.NotFound:
				resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeNotFound, errMsg, nil)
				return HTTPStatusCode_404_NotFound, resp
			case codes.AlreadyExists:
				resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, errMsg, nil)
				return HTTPStatusCode_400_BadRequest, resp
			case codes.InvalidArgument:
				resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, errMsg, nil)
				return HTTPStatusCode_400_BadRequest, resp
			default:
				resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, errMsg, err)
				return HTTPStatusCode_500_InternalServerError, resp
			}
		}

		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to update subaccount", err)
		return HTTPStatusCode_500_InternalServerError, resp
	}
	if grpcResp.Name != name {
		ctx.Logger.Error("Failed to update subaccount: name mismatch",
			"expected_name", name,
			"new_name", grpcResp.Name,
			"sub_account_id", subAccountId,
		)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to update subaccount: name was not updated correctly", fmt.Errorf("expected name %q, got %q", name, grpcResp.Name))
		return HTTPStatusCode_500_InternalServerError, resp
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, UpdateSubAccountNameResponse{
		SubAccountId: snx_lib_api_types.SubAccountIdFromIntUnvalidated(int64(subAccountId)), // TODO: conversion function required
		Name:         grpcResp.Name,
	})
}
