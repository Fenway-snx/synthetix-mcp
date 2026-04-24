package info

import (
	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_db_repository "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/repository"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getMids
*/

// MidsResponse represents the response for mid price data
// Maps symbol to mid price (e.g., {"BTC-USD": "45025.37500000"})
type MidsResponse map[Symbol]Price

// Handler for "getMids".
//
//dd:span
func Handle_getMids(
	ctx InfoContext,
	_params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {

	// first, get list of markets from marketconfig service
	markets, err := snx_lib_api_handlers_utils.QueryMarkets(ctx, true)
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Could not pull market configuration", err)
	}

	priceRepo := snx_lib_db_repository.NewPriceRepository(ctx.Rc)

	prices := make(MidsResponse)
	for _, market := range markets {
		symbol := string(market.Symbol)

		// get mid price from price feed
		// TODO: would be better if we could batch these calls but life
		midPrice, err := priceRepo.GetPriceHistory(ctx.Context, symbol, "price", 1)
		if err != nil {
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Could not get mid price", err)
		}

		prices[market.Symbol] = snx_lib_api_handlers_utils.PriceFromPriceData(midPrice)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, prices)
}
