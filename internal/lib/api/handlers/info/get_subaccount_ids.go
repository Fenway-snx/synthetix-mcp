package info

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getSubAccountIds
*/

// SubAccountIdsRequest represents the request to fetch subaccount IDs for a wallet
type SubAccountIdsRequest struct {
	IncludeDelegations bool          `json:"includeDelegations"`
	WalletAddress      WalletAddress `json:"walletAddress" validate:"required"`
}

// SubAccountIdsWithDelegationsResponse is returned when includeDelegations is true.
type SubAccountIdsWithDelegationsResponse struct {
	DelegatedSubAccountIds []SubAccountId `json:"delegatedSubAccountIds"`
	SubAccountIds          []SubAccountId `json:"subAccountIds"`
}

// Handler for "getSubAccountIds" - obtains subaccount IDs for a given
// wallet address.
//
// When includeDelegations is true, returns an object with both owned and
// delegated subaccount IDs. Otherwise returns a flat array of owned IDs
// (legacy behaviour).
//
// Note:
// This is a public endpoint under /info and does not require
// authentication.
//
//dd:span
func Handle_getSubAccountIds(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req SubAccountIdsRequest
	if err := mapstructure.Decode(params, &req); err != nil || string(req.WalletAddress) == "" {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	walletAddress := strings.TrimSpace(string(req.WalletAddress))
	if err := snx_lib_api_validation.ValidateStringMaxLength(walletAddress, snx_lib_api_validation.MaxEthAddressLength, "walletAddress"); err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	if !common.IsHexAddress(walletAddress) {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid wallet address format", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	checksumAddress := common.HexToAddress(walletAddress).Hex()
	if walletAddress != checksumAddress {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](
			ctx.ClientRequestId,
			fmt.Sprintf("walletAddress must use EIP-55 checksum casing; expected %s", checksumAddress),
			nil,
		)
		return HTTPStatusCode_400_BadRequest, resp
	}
	req.WalletAddress = WalletAddress(checksumAddress)

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	if req.IncludeDelegations {
		return handleWithDelegations(
			ctx,
			req.WalletAddress,
			timestamp_us,
			timestamp_ms,
		)
	}

	return handleLegacy(
		ctx,
		req.WalletAddress,
		timestamp_us,
		timestamp_ms,
	)
}

// Returns owned IDs as a flat string array (legacy behaviour).
func handleLegacy(
	ctx InfoContext,
	walletAddress WalletAddress,
	timestamp_us int64,
	timestamp_ms int64,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	grpcResp, err := ctx.SubaccountClient.ListSubaccounts(ctx.Context, &v4grpc.ListSubaccountsRequest{
		TimestampMs:   timestamp_ms,
		TimestampUs:   timestamp_us,
		WalletAddress: snx_lib_api_types.WalletAddressToString(walletAddress),
	})
	if err != nil {
		failMessage := "Failed to list subaccounts"
		ctx.Logger.Error(failMessage, "error", err)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	ids := make([]SubAccountId, 0, len(grpcResp.Subaccounts))
	for _, sa := range grpcResp.Subaccounts {
		ids = append(ids, snx_lib_api_types.SubAccountIdFromIntUnvalidated(sa.Id))
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, ids)
}

// Returns owned and delegated IDs as an object.
func handleWithDelegations(
	ctx InfoContext,
	walletAddress WalletAddress,
	timestamp_us int64,
	timestamp_ms int64,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	addr := snx_lib_api_types.WalletAddressToString(walletAddress)

	var (
		wg        sync.WaitGroup
		listResp  *v4grpc.ListSubaccountsResponse
		listErr   error
		delegResp *v4grpc.GetDelegationsForDelegateResponse
		delegErr  error
	)

	wg.Go(func() {
		listResp, listErr = ctx.SubaccountClient.ListSubaccounts(ctx.Context, &v4grpc.ListSubaccountsRequest{
			TimestampMs:   timestamp_ms,
			TimestampUs:   timestamp_us,
			WalletAddress: addr,
		})
	})

	wg.Go(func() {
		delegResp, delegErr = ctx.SubaccountClient.GetDelegationsForDelegate(ctx.Context, &v4grpc.GetDelegationsForDelegateRequest{
			TimestampMs:     timestamp_ms,
			TimestampUs:     timestamp_us,
			DelegateAddress: addr,
		})
	})

	wg.Wait()

	if listErr != nil {
		failMessage := "Failed to list subaccounts"
		ctx.Logger.Error(failMessage, "error", listErr)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, listErr)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	if delegErr != nil {
		failMessage := "Failed to list delegated subaccounts"
		ctx.Logger.Error(failMessage, "error", delegErr)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, delegErr)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	ownedIds := make([]SubAccountId, 0, len(listResp.Subaccounts))
	for _, sa := range listResp.Subaccounts {
		ownedIds = append(ownedIds, SubAccountId(strconv.FormatInt(sa.Id, 10)))
	}

	delegatedIds := make([]SubAccountId, 0, len(delegResp.Delegations))
	for _, d := range delegResp.Delegations {
		delegatedIds = append(delegatedIds, SubAccountId(strconv.FormatInt(d.SubAccountId, 10)))
	}

	result := SubAccountIdsWithDelegationsResponse{
		DelegatedSubAccountIds: delegatedIds,
		SubAccountIds:          ownedIds,
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, result)
}
