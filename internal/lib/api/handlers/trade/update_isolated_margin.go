package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/updateIsolatedMargin
*/

// T.B.C.
type IsolatedMarginRequest struct {
}

// T.B.C.
type IsolatedMarginResponse struct {
}

// Handler for "updateIsolatedMargin".
//
//dd:span
func Handle_updateIsolatedMargin(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {

	isolatedMarginResponse := make([]IsolatedMarginResponse, 0)

	resp := snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, isolatedMarginResponse)

	return HTTPStatusCode_501_StatusNotImplemented, resp
}
