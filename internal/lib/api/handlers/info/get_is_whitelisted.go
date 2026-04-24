package info

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_api_whitelist "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/whitelist"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

var (
	errWhitelistArbitratorNil = errors.New("whitelist arbitrator is nil")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getIsWhitelisted
*/

// GetIsWhitelistedRequest captures the wallet address to be checked
type GetIsWhitelistedRequest struct {
	WalletAddress WalletAddress `json:"walletAddress" mapstructure:"walletAddress"`
}

// Handler for "getIsWhitelisted".
//
//dd:span
func Handle_getIsWhitelisted(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req GetIsWhitelistedRequest
	if err := mapstructure.Decode(params, &req); err != nil || req.WalletAddress == "" {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	addr := strings.TrimSpace(string(req.WalletAddress))
	if err := snx_lib_api_validation.ValidateStringMaxLength(addr, snx_lib_api_validation.MaxEthAddressLength, "walletAddress"); err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	if !common.IsHexAddress(addr) {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid wallet address format", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	normalizedAddr := strings.ToLower(addr)

	if ctx.WhitelistArbitrator == nil {
		const msg = "whitelist arbitrator unavailable"

		ctx.Logger.Error(msg,
			"wallet_address", snx_lib_core.MaskAddress(normalizedAddr),
		)

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, msg, errWhitelistArbitratorNil)
	}

	allowed, err := ctx.WhitelistArbitrator.CanOrdersBePlacedFor(snx_lib_api_whitelist.WalletAddress(normalizedAddr))
	if err != nil {
		const msg = "failed to obtain whitelist arbitration"

		ctx.Logger.Error(msg,
			"error", err,
			"wallet_address", snx_lib_core.MaskAddress(normalizedAddr),
		)

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, msg, err)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, allowed)
}
