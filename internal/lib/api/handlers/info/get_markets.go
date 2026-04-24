package info

import (
	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getMarkets
*/

// Market represents a trading market configuration
type MarketsResponse []MarketResponse

// Handler for "getMarkets".
//
//dd:span
func Handle_getMarkets(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Default to false (get all markets), but allow override from params
	activeOnly := false
	if v, ok := params["activeOnly"]; ok {
		if b, ok := v.(bool); ok {
			activeOnly = b
		}
	}

	markets, err := snx_lib_api_handlers_utils.QueryMarkets(ctx, activeOnly)
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Could not pull market configuration", err)
	}

	// Return success response (markets is already []MarketResponse)
	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, markets)
}
