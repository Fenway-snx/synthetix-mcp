package trade

import (
	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getRateLimits
*/

// Optional client fields for the getRateLimits action (validated for shape only).
type RateLimitRequest struct {
	User string `json:"user"`
}

// JSON success body for getRateLimits: requestsUsed and requestsCap, derived
// from GetRateLimitsSubaccountSnapshot.PublicCounts (consumed + cap vs.
// remaining + capacity on the snapshot).
type RateLimitResponse struct {
	RequestsUsed int `json:"requestsUsed"`
	RequestsCap  int `json:"requestsCap"`
}

// Serves getRateLimits using the subaccount snapshot from TradeContext.
//
//dd:span
func Handle_getRateLimits(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req RateLimitRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}

	used, cap := ctx.GetRateLimitsSubaccountSnapshot().PublicCounts()
	rateLimits := RateLimitResponse{RequestsUsed: used, RequestsCap: cap}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, rateLimits)
}
