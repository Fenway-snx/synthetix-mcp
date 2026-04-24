package trade

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-viper/mapstructure/v2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/addDelegatedSigner
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getDelegatedSigners
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/removeDelegatedSigner
*/

// Request structs for delegation handlers
type CreateDelegationRequest struct {
	WalletAddress WalletAddress `json:"walletAddress"`
	Permissions   []string      `json:"permissions"`
	ExpiresAt     *int64        `json:"expiresAt,omitempty"`
}

type GetDelegationsForDelegateParams struct {
	OwningAddress WalletAddress `json:"owningAddress,omitempty" mapstructure:"owningAddress"`
}

type RemoveDelegationRequest struct {
	WalletAddress WalletAddress `json:"walletAddress"`
}

// Helper functions for delegation handlers

// Handles gRPC errors and converts them to appropriate HTTP responses.
//
// For `AlreadyExists`, `InvalidArgument`, and `FailedPrecondition`,
// the helper inspects any `errdetails.ErrorInfo` attached to the gRPC
// status and adopts its `Reason` as the client-facing `ErrorCode` and
// its `Metadata` as the response `details` map, but only when `Reason`
// matches a registered code in `lib/core/status_codes`. Unrecognised
// reasons and metadata are discarded, preventing upstream services from
// injecting arbitrary codes or leaking internal identifiers. For
// `PermissionDenied`, the same inspection applies with `FORBIDDEN` as
// the fallback, allowing tier-validation failures to surface as
// `VALIDATION_ERROR` when the gRPC service attaches the appropriate
// `ErrorInfo`. When no `ErrorInfo` is present the fallback code is used.
func handleGRPCError(err error, clientRequestId ClientRequestId) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.AlreadyExists, codes.InvalidArgument, codes.FailedPrecondition:
			errorCode, details := clientErrorCodeAndDetailsFromStatus(st, ErrorCodeValidationError)
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](clientRequestId, errorCode, st.Message(), details)
		case codes.NotFound:
			return HTTPStatusCode_404_NotFound, snx_lib_api_json.NewErrorResponse[any](clientRequestId, snx_lib_api_json.ErrorCodeNotFound, st.Message(), nil)
		case codes.Unauthenticated:
			return HTTPStatusCode_401_Unauthorized, snx_lib_api_json.NewErrorResponse[any](clientRequestId, snx_lib_api_json.ErrorCodeUnauthorized, st.Message(), nil)
		case codes.PermissionDenied:
			errorCode, details := clientErrorCodeAndDetailsFromStatus(st, snx_lib_api_json.ErrorCodeForbidden)
			return HTTPStatusCode_403_Forbidden, snx_lib_api_json.NewErrorResponse[any](clientRequestId, errorCode, st.Message(), details)
		case codes.Unimplemented:
			return HTTPStatusCode_501_StatusNotImplemented, snx_lib_api_json.NewSystemErrorResponse[any](clientRequestId, "Not implemented", err)
		case codes.Internal:
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](clientRequestId, st.Message(), err)
		}
	}
	// Default for unknown errors
	return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](clientRequestId, "Internal server error", err)
}

// Inspects st for an `errdetails.ErrorInfo` and returns the client-facing
// error code and response details derived from it. The `Reason` is only
// adopted as the client error code when it matches a registered
// `ErrorCode` in `lib/core/status_codes`; otherwise the fallback code
// is returned with `nil` details so that arbitrary upstream-supplied
// reasons and metadata cannot reach external clients.
func clientErrorCodeAndDetailsFromStatus(
	st *status.Status,
	fallback snx_lib_api_json.ErrorCode,
) (snx_lib_api_json.ErrorCode, map[string]string) {
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		reason := snx_lib_api_json.ErrorCode(info.GetReason())
		if !snx_lib_api_json.IsKnownErrorCode(reason) {
			return fallback, nil
		}
		return reason, info.GetMetadata()
	}
	return fallback, nil
}

// represents a delegated signer response
type DelegatedSigner struct {
	SubAccountId  SubAccountId   `json:"subAccountId"`      // The subaccount ID the signer was added to
	WalletAddress WalletAddress  `json:"walletAddress"`     // The delegated signer's wallet address
	Permissions   []string       `json:"permissions"`       // The permission levels granted
	ExpiresAt     *Timestamp     `json:"expiresAt"`         // Expiration timestamp (null if no expiration)
	AddedBy       *WalletAddress `json:"addedBy,omitempty"` // Address that created this delegation (nil for pre-migration records)
}

// represents the response for getDelegatedSigners
type DelegatedSignerList struct {
	DelegatedSigners []DelegatedSigner `json:"delegatedSigners"` // Array of delegated signer objects
}

// Handles "addDelegatedSigner" requests.
//
//dd:span
func Handle_addDelegatedSigner(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*snx_lib_api_validation.ValidatedAddDelegatedSignerAction)
	if !ok || validated == nil || validated.Payload == nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request format", nil)
	}
	req := CreateDelegationRequest{
		WalletAddress: WalletAddress(validated.Payload.DelegateAddress),
		Permissions:   validated.Payload.Permissions,
		ExpiresAt:     validated.Payload.ExpiresAt,
	}

	// Use the authenticated subaccount ID from context (from top-level subaccountId)
	subAccountIdInt := int64(ctx.SelectedAccountId)

	// Convert expiresAt from milliseconds to protobuf timestamp
	// Treat 0 as "no expiration" (nil) rather than epoch time 1970
	var expiresAt *timestamppb.Timestamp
	if req.ExpiresAt != nil && *req.ExpiresAt > 0 {
		expiresAt = timestamppb.New(time.UnixMilli(*req.ExpiresAt))
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call subaccount service via gRPC
	grpcReq := &v4grpc.CreateDelegationRequest{
		TimestampMs:     timestamp_ms,
		TimestampUs:     timestamp_us,
		SubAccountId:    subAccountIdInt,
		DelegateAddress: string(req.WalletAddress),
		Permissions:     req.Permissions,
		ExpiresAt:       expiresAt,
		SignerAddress:   string(ctx.WalletAddress),
	}

	grpcResp, err := ctx.SubaccountClient.CreateDelegation(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to create delegation",
			"error", err,
			"sub_account_id", subAccountIdInt,
			"wallet_address", snx_lib_core.MaskAddress(string(req.WalletAddress)),
		)
		return handleGRPCError(err, ctx.ClientRequestId)
	}

	// Convert gRPC response to API response

	delegatedSigner := DelegatedSigner{
		SubAccountId:  snx_lib_api_types.SubAccountIdFromIntUnvalidated(subAccountIdInt),
		WalletAddress: WalletAddress(grpcResp.DelegateAddress),
		Permissions:   grpcResp.Permissions,
		ExpiresAt:     snx_lib_api_types.TimestampPtrFromTimestampPBOrNil(grpcResp.ExpiresAt),
	}

	ctx.Logger.Info("Successfully created delegation",
		"permissions", grpcResp.Permissions,
		"sub_account_id", subAccountIdInt,
		"wallet_address", snx_lib_core.MaskAddress(string(req.WalletAddress)),
	)

	// NOTE: THIS IS A TEMPORARY MECHANISM (FOR SNX-5190)
	{
		if ctx.Rc != nil {
			key := delegateWhitelistKey(grpcResp.DelegateAddress)
			if err := ctx.Rc.Set(ctx.Context, key, string(ctx.WalletAddress), -1).Err(); err != nil {
				ctx.Logger.Error("could not set delegate",
					"error", err,
				)
			}
		}
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, delegatedSigner)
}

// represents the response for removeDelegatedSigner
type RemoveDelegatedSignerResponse struct {
	SubAccountId          SubAccountId    `json:"subAccountId"`                    // The subaccount ID the signer was removed from
	WalletAddress         WalletAddress   `json:"walletAddress"`                   // The removed delegated signer's wallet address
	CascadeRemovedSigners []WalletAddress `json:"cascadeRemovedSigners,omitempty"` // Addresses cascade-deleted because they were created by the removed delegate
}

// Handles "getDelegatedSigners" requests.
//
//dd:span
func Handle_getDelegatedSigners(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Use the authenticated subaccount ID from context (from top-level subaccountId)
	subAccountId := snx_lib_api_types.SubAccountIdFromIntUnvalidated(int64(ctx.SelectedAccountId)) // TODO: conversion function

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call subaccount service via gRPC
	grpcReq := &v4grpc.GetDelegationsRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(ctx.SelectedAccountId),
	}

	grpcResp, err := ctx.SubaccountClient.GetDelegations(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get delegations",
			"error", err,
			"sub_account_id", subAccountId,
		)

		return handleGRPCError(err, ctx.ClientRequestId)
	}

	// Convert gRPC response to API response
	delegatedSigners := make([]DelegatedSigner, len(grpcResp.Delegations))
	for i, delegation := range grpcResp.Delegations {
		signer := DelegatedSigner{
			SubAccountId:  subAccountId,
			WalletAddress: WalletAddress(delegation.DelegateAddress),
			Permissions:   delegation.Permissions,
			ExpiresAt:     snx_lib_api_types.TimestampPtrFromTimestampPBOrNil(delegation.ExpiresAt),
		}
		if delegation.AddedBy != nil {
			addedBy := WalletAddress(*delegation.AddedBy)
			signer.AddedBy = &addedBy
		}
		delegatedSigners[i] = signer
	}

	ctx.Logger.Info("Successfully retrieved delegations",
		"count", len(delegatedSigners),
		"sub_account_id", subAccountId,
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, DelegatedSignerList{
		DelegatedSigners: delegatedSigners,
	})
}

// Handles "removeDelegatedSigner" requests.
//
//dd:span
func Handle_removeDelegatedSigner(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*snx_lib_api_validation.ValidatedRemoveDelegatedSignerAction)
	if !ok || validated == nil || validated.Payload == nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request format", nil)
	}
	req := RemoveDelegationRequest{
		WalletAddress: WalletAddress(validated.Payload.DelegateAddress),
	}

	// Use the authenticated subaccount ID from context (from top-level subaccountId)
	subAccountIdInt := int64(ctx.SelectedAccountId)

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call subaccount service via gRPC
	grpcReq := &v4grpc.RemoveDelegationRequest{
		TimestampMs:     timestamp_ms,
		TimestampUs:     timestamp_us,
		SubAccountId:    subAccountIdInt,
		DelegateAddress: string(req.WalletAddress),
		SignerAddress:   string(ctx.WalletAddress),
	}

	grpcResp, err := ctx.SubaccountClient.RemoveDelegation(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to remove delegation",
			"error", err,
			"sub_account_id", subAccountIdInt,
			"wallet_address", snx_lib_core.MaskAddress(string(req.WalletAddress)),
		)
		return handleGRPCError(err, ctx.ClientRequestId)
	}

	subAccountId := snx_lib_api_types.SubAccountIdFromIntUnvalidated(subAccountIdInt)
	walletAddress := WalletAddress(grpcResp.DelegateAddress)

	// Map cascade-removed addresses
	var cascadeRemovedSigners []WalletAddress
	if len(grpcResp.CascadeRemovedAddresses) > 0 {
		cascadeRemovedSigners = make([]WalletAddress, len(grpcResp.CascadeRemovedAddresses))
		for i, addr := range grpcResp.CascadeRemovedAddresses {
			cascadeRemovedSigners[i] = WalletAddress(addr)
		}
	}

	ctx.Logger.Info("Successfully removed delegation",
		"cascade_removed_count", len(cascadeRemovedSigners),
		"sub_account_id", subAccountId,
		"wallet_address", snx_lib_core.MaskAddress(string(walletAddress)),
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, RemoveDelegatedSignerResponse{
		SubAccountId:          subAccountId,
		WalletAddress:         walletAddress,
		CascadeRemovedSigners: cascadeRemovedSigners,
	})
}

// RemoveAllDelegatedSignersResponse represents the response for removeAllDelegatedSigners
type RemoveAllDelegatedSignersResponse struct {
	SubAccountId   SubAccountId    `json:"subAccountId"`   // The subaccount ID from which signers were removed
	RemovedSigners []WalletAddress `json:"removedSigners"` // List of removed delegate wallet addresses
}

// Handles "removeAllDelegatedSigners" requests.
// Atomically removes all delegated signers from a subaccount.
//
//dd:span
func Handle_removeAllDelegatedSigners(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Use the authenticated subaccount ID from context (from top-level subaccountId)
	subAccountIdInt := int64(ctx.SelectedAccountId)

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call subaccount service via gRPC
	grpcReq := &v4grpc.RemoveAllDelegationsRequest{
		TimestampMs:   timestamp_ms,
		TimestampUs:   timestamp_us,
		SubAccountId:  subAccountIdInt,
		SignerAddress: string(ctx.WalletAddress),
	}

	grpcResp, err := ctx.SubaccountClient.RemoveAllDelegations(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to remove all delegations", "error", err, "sub_account_id", subAccountIdInt)
		return handleGRPCError(err, ctx.ClientRequestId)
	}

	// Convert gRPC response to API response
	removedSigners := make([]WalletAddress, len(grpcResp.RemovedSigners))
	for i, addr := range grpcResp.RemovedSigners {
		removedSigners[i] = WalletAddress(addr)
	}

	ctx.Logger.Info("Successfully removed all delegations",
		"sub_account_id", subAccountIdInt,
		"removed_count", len(removedSigners),
	)

	subAccountId := snx_lib_api_types.SubAccountIdFromIntUnvalidated(subAccountIdInt)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, RemoveAllDelegatedSignersResponse{
		SubAccountId:   subAccountId,
		RemovedSigners: removedSigners,
	})
}

// Represents an account that has delegated access to the caller.
type DelegatedAccountInfo struct {
	SubAccountId         SubAccountId  `json:"subAccountId"`         // The subaccount ID that delegated access
	OwnerAddress         WalletAddress `json:"ownerAddress"`         // The owner's wallet address
	AccountName          string        `json:"accountName"`          // The subaccount name
	AccountValue         string        `json:"accountValue"`         // Total account value in USDT with UPNL
	AdjustedAccountValue string        `json:"adjustedAccountValue"` // Total account value in USDT with UPNL and Haircut
	Permissions          []string      `json:"permissions"`          // The permission levels granted
	ExpiresAt            *Timestamp    `json:"expiresAt"`            // Expiration timestamp (null if no expiration)
}

// Represents the response for getDelegationsForDelegate.
type DelegatedAccountsList struct {
	DelegatedAccounts []DelegatedAccountInfo `json:"delegatedAccounts"` // Array of accounts that have delegated to the caller
}

// Handles "getDelegationsForDelegate" requests.
// Returns all accounts that have delegated access to the authenticated wallet.
// Supports an optional `owningAddress` parameter to allow a session signer to
// query delegations for its parent delegate, after authorization validation.
//
//dd:span
func Handle_getDelegationsForDelegate(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Parse optional owningAddress from params
	var req GetDelegationsForDelegateParams
	if err := mapstructure.Decode(params, &req); err != nil {
		ctx.Logger.Error("Failed to decode getDelegationsForDelegate params", "error", err)
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request format", nil)
	}

	// Default: use the authenticated wallet address as the delegate
	delegateAddress := string(ctx.WalletAddress)

	// If owningAddress is provided and differs from the caller, validate authorization
	if req.OwningAddress != "" && !snx_lib_core.AddressesEqual(string(req.OwningAddress), delegateAddress) {
		if !common.IsHexAddress(string(req.OwningAddress)) {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "Invalid owningAddress format", nil)
		}

		timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

		authResp, err := ctx.SubaccountClient.VerifySubaccountAuthorization(ctx.Context, &v4grpc.VerifySubaccountAuthorizationRequest{
			TimestampMs:  timestamp_ms,
			TimestampUs:  timestamp_us,
			SubAccountId: int64(ctx.SelectedAccountId),
			Address:      string(req.OwningAddress),
			Permissions:  []string{string(snx_lib_core.DelegationPermissionTrading)},
		})
		if err != nil {
			ctx.Logger.Error("Failed to verify owningAddress authorization",
				"error", err,
				"owning_address", snx_lib_core.MaskAddress(string(req.OwningAddress)),
				"sub_account_id", ctx.SelectedAccountId,
			)
			return handleGRPCError(err, ctx.ClientRequestId)
		}

		if !authResp.IsAuthorized {
			ctx.Logger.Info("Caller not authorized to query delegations for owningAddress",
				"owning_address", snx_lib_core.MaskAddress(string(req.OwningAddress)),
				"sub_account_id", ctx.SelectedAccountId,
				"wallet_address", snx_lib_core.MaskAddress(delegateAddress),
			)
			return HTTPStatusCode_403_Forbidden, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeForbidden, "Not authorized to query delegations for the specified owningAddress", nil)
		}

		delegateAddress = string(req.OwningAddress)
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call subaccount service via gRPC
	grpcReq := &v4grpc.GetDelegationsForDelegateRequest{
		TimestampMs:     timestamp_ms,
		TimestampUs:     timestamp_us,
		DelegateAddress: delegateAddress,
	}

	grpcResp, err := ctx.SubaccountClient.GetDelegationsForDelegate(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get delegations for delegate",
			"error", err,
			"delegate_address", snx_lib_core.MaskAddress(delegateAddress),
		)
		return handleGRPCError(err, ctx.ClientRequestId)
	}

	// Convert gRPC response to API response
	delegatedAccounts := make([]DelegatedAccountInfo, len(grpcResp.Delegations))
	for i, delegation := range grpcResp.Delegations {
		delegatedAccounts[i] = DelegatedAccountInfo{
			SubAccountId:         snx_lib_api_types.SubAccountIdFromIntUnvalidated(delegation.SubAccountId),
			OwnerAddress:         WalletAddress(delegation.OwnerAddress),
			AccountName:          delegation.AccountName,
			AccountValue:         delegation.AccountValue,
			AdjustedAccountValue: delegation.AdjustedAccountValue,
			Permissions:          delegation.Permissions,
			ExpiresAt:            snx_lib_api_types.TimestampPtrFromTimestampPBOrNil(delegation.ExpiresAt),
		}
	}

	delegatedAccountsList := DelegatedAccountsList{
		DelegatedAccounts: delegatedAccounts,
	}

	ctx.Logger.Info("Successfully retrieved delegations for delegate",
		"count", len(delegatedAccounts),
		"delegate_address", snx_lib_core.MaskAddress(delegateAddress),
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, delegatedAccountsList)
}
