package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/placeIsolatedOrder
*/

// T.B.C.
type PlaceIsolatedOrderRequest struct {
}

// T.B.C.
type PlaceIsolatedOrderResponse struct {
}

// T.B.C.
type PlaceIsolatedOrdersResponse []PlaceIsolatedOrderResponse

// Handler for "placeIsolatedOrder".
//
//dd:span
func Handle_placeIsolatedOrder(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {

	placeIsolatedOrderResponse := make([]PlaceIsolatedOrderResponse, 0)

	resp := snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, placeIsolatedOrderResponse)

	return HTTPStatusCode_501_StatusNotImplemented, resp
}
