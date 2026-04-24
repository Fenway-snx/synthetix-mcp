package info

import (
	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getOpenInterest
*/

// OpenInterest represents the open interest for a symbol
type OpenInterest struct {
	Symbol            Symbol    `json:"symbol"`            // Trading pair symbol (e.g., "BTC-USD")
	OpenInterest      string    `json:"openInterest"`      // Current open interest
	LongOpenInterest  string    `json:"longOpenInterest"`  // Current open interest long
	ShortOpenInterest string    `json:"shortOpenInterest"` // Current open interest short
	Timestamp         Timestamp `json:"timestamp"`         // Execution timestamp in milliseconds
}

// OpenInterests represents the response for open interest data
type OpenInterests []OpenInterest

// Handler for "getOpenInterest".
//
//dd:span
func Handle_getOpenInterest(
	ctx InfoContext,
	_params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {

	markets, err := snx_lib_api_handlers_utils.QueryMarkets(ctx, true)
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "failed to obtain markets configuration", err)
	}

	_, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()

	openInterests := make(OpenInterests, len(markets))

	for i := 0; i != len(markets); i++ {
		market := markets[i]

		if longOpenInterest, shortOpenInterest, err := snx_lib_api_handlers_utils.OpenInterestForMarket(ctx.Context, ctx.SubaccountClient, market.Symbol); err != nil {

			ctx.Logger.Error("failed to obtain positions for symbol",
				"error", err,
				"symbol", market.Symbol,
			)

			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Could not get positions data", err)
		} else {

			openInterest := longOpenInterest.Add(shortOpenInterest)

			// TODO: convert `OpenInterest` fields to `Volume`
			openInterests[i] = OpenInterest{
				Symbol:            market.Symbol,
				OpenInterest:      openInterest.String(),
				LongOpenInterest:  longOpenInterest.String(),
				ShortOpenInterest: shortOpenInterest.String(),
				Timestamp:         Timestamp(timestampMs),
			}
		}
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, openInterests)
}
