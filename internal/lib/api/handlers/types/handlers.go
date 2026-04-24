package types

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

// Handler-specific parameters (after the outer JSON object is stripped,
// this is the contents of the `"params"` field)
type HandlerParams map[string]any

// T.B.C.
func (hp HandlerParams) Map() map[string]any {
	return hp
}

// T.B.C.
func (hp HandlerParams) Raw() []byte {
	return nil
}

/*
// Handlers map type for "/history"
//
// Handler function parameters:
// - ctx : the execution/request context pertinent to "/history" requests;
// - params : the request payload parameters;
type HistoryHandlers map[RequestAction]func(ctx history.HistoryContext, params map[string]any) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any])
*/

// Handlers map type for "/info"
//
// Handler function parameters:
// - ctx : the execution/request context pertinent to "/info" requests;
// - params : the request payload parameters;
type InfoHandlers map[RequestAction]func(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any])

// Handlers map type for "/trade"
//
// Handler function parameters:
// - ctx : the execution/request context pertinent to "/trade" requests;
// - params : the request payload parameters;
type TradeHandlers map[RequestAction]func(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any])
