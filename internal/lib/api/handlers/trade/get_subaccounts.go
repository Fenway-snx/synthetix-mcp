package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getSubAccounts
*/

// SubAccountWithDelegates extends SubAccountResponse with delegated signers.
// Uses embedding so JSON marshaling flattens the fields.
type SubAccountWithDelegates struct {
	SubAccountResponse
	DelegatedSigners []DelegatedSigner `json:"delegatedSigners"`
}

// SubAccountsListResponse is the response wrapper for getSubAccounts
type SubAccountsListResponse struct {
	SubAccounts []SubAccountWithDelegates `json:"subAccounts"`
}

// Handler for "getSubAccounts".
// Returns all subaccounts for the authenticated subaccount's master account with delegates included.
// Works for both owners and delegates - returns all subaccounts under the same master.
//
//dd:span
func Handle_getSubAccounts(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subaccountId is required", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.ListSubaccounts(ctx.Context, &v4grpc.ListSubaccountsRequest{
		TimestampMs:  timestampMs,
		TimestampUs:  timestampUs,
		SubAccountId: int64(ctx.SelectedAccountId),
	})
	if err != nil {
		failMessage := "Failed to list subaccounts"

		ctx.Logger.Error(failMessage, "error", err, "wallet_address", snx_lib_core.MaskAddress(string(ctx.WalletAddress)))
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	subAccounts := make([]SubAccountWithDelegates, 0, len(grpcResp.Subaccounts))
	for _, subaccount := range grpcResp.Subaccounts {
		subAccounts = append(subAccounts, mapSubaccountInfoWithDelegates(subaccount))
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, SubAccountsListResponse{
		SubAccounts: subAccounts,
	})
}

func mapSubaccountInfoWithDelegates(subaccount *v4grpc.SubaccountInfo) SubAccountWithDelegates {
	if subaccount == nil {
		return SubAccountWithDelegates{}
	}

	// Reuse existing mapping for the base response
	base := mapSubaccountInfo(subaccount)

	// Convert delegations to DelegatedSigner format
	delegatedSigners := make([]DelegatedSigner, 0, len(subaccount.Delegations))
	for _, delegation := range subaccount.Delegations {
		delegatedSigners = append(delegatedSigners, DelegatedSigner{
			SubAccountId:  snx_lib_api_types.SubAccountIdFromIntUnvalidated(delegation.SubAccountId),
			WalletAddress: WalletAddress(delegation.DelegateAddress),
			Permissions:   delegation.Permissions,
			ExpiresAt:     snx_lib_api_types.TimestampPtrFromTimestampPBOrNil(delegation.ExpiresAt),
		})
	}

	return SubAccountWithDelegates{
		SubAccountResponse: base,
		DelegatedSigners:   delegatedSigners,
	}
}
