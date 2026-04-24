package info

import (
	"fmt"
	"slices"

	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getOrderbook
*/

type DepthRequest struct {
	Symbol Symbol `json:"symbol" validate:"required"`
	Limit  int    `json:"limit,omitempty"`
}

type DepthResponse struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

// Handler for "getOrderbook".
//
//dd:span
func Handle_getOrderbook(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req DepthRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	normalizedSymbol, err := snx_lib_api_validation.ValidateAndNormalizeSymbol(req.Symbol)
	if err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	req.Symbol = normalizedSymbol

	// Parse limit parameter with default value
	if req.Limit == 0 {
		req.Limit = 500 // Default limit
	}

	validLimits := []int{5, 10, 20, 50, 100, 500, 1000}
	if !slices.Contains(validLimits, req.Limit) {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid limit parameter", map[string]string{
			"details": fmt.Sprintf("Valid limits are: %v", validLimits),
		})
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Request orderbook depth via gRPC
	grpcReq := &v4grpc.OrderbookDepthRequest{
		Symbol: string(req.Symbol),
		Limit:  int32(req.Limit),
	}

	// Call market data service
	grpcResp, err := ctx.MarketDataClient.GetOrderbookDepth(ctx, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get orderbook depth from market data service", "error", err, API_WKS_symbol, req.Symbol)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to get market data", err)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	// Apply limit to gRPC response
	bids := grpcResp.Bids
	asks := grpcResp.Asks

	if len(bids) > req.Limit {
		bids = bids[:req.Limit]
	}
	if len(asks) > req.Limit {
		asks = asks[:req.Limit]
	}
	// Convert to Binance-style depth response
	bidsFormatted := make([][]string, len(bids))
	for i, bid := range bids {
		bidsFormatted[i] = []string{bid.Price, bid.Quantity}
	}

	asksFormatted := make([][]string, len(asks))
	for i, ask := range asks {
		asksFormatted[i] = []string{ask.Price, ask.Quantity}
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, DepthResponse{
		Bids: bidsFormatted,
		Asks: asksFormatted,
	})
}
