package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/modifyOrdersBatch
*/

// T.B.C.
type ModifyOrderBatchRequest struct {
}

// T.B.C.
type ModifyOrderBatchResponse struct {
}

// T.B.C.
type ModifyOrderBatchsResponse []ModifyOrderBatchResponse

// Handler for "modifyOrderBatch".
//
//dd:span
func Handle_modifyOrderBatch(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {

	modifyOrderBatchResponse := make([]ModifyOrderBatchResponse, 0)

	resp := snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, modifyOrderBatchResponse)

	return HTTPStatusCode_501_StatusNotImplemented, resp
}
