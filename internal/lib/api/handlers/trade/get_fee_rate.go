package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getFeeRate
*/

// T.B.C.
type FeeRateRequest struct {
}

// T.B.C.
type FeeRateResponse struct {
}

// Handler for "getFeeRate".
//
//dd:span
func Handle_getFeeRate(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {

	feeRateResponse := &FeeRateResponse{}

	resp := snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, feeRateResponse)

	return HTTPStatusCode_501_StatusNotImplemented, resp
}
