package info

import (
	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getLastTrades
*/

type LastTradesRequest struct {
	Symbol Symbol `json:"symbol" validate:"required"` // Trading pair symbol
	Limit  int    `json:"limit,omitempty"`            // Number of trades to return (default 50, max 100)
}

// PublicTrade represents a single trade in the public feed
type PublicTrade struct {
	TradeId   TradeId   `json:"tradeId"`   // Unique trade identifier
	Symbol    Symbol    `json:"symbol"`    // Trading pair symbol
	Side      string    `json:"side"`      // Trade side: "buy" or "sell"
	Price     Price     `json:"price"`     // Execution price
	Quantity  Quantity  `json:"quantity"`  // Executed quantity
	Timestamp Timestamp `json:"timestamp"` // Execution timestamp in milliseconds
	IsMaker   bool      `json:"isMaker"`   // True if this side was the maker
}

// GetLastTradesResponse represents the response for getLastTrades
type GetLastTradesResponse struct {
	Trades []PublicTrade `json:"trades"` // Array of recent trades
}

// Handler for "getLastTrades" - gets recent trades for a market.
//
//dd:span
func Handle_getLastTrades(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req LastTradesRequest
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

	// Set default and validate limit
	if req.Limit == 0 {
		req.Limit = 50 // Default limit
	}
	if req.Limit > 100 {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Limit cannot exceed 100", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	if req.Limit < 1 {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Limit must be at least 1", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Use dedicated GetLastTrades method for public market trades
	limit := int32(req.Limit)
	symbol := req.Symbol

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	grpcReq := &v4grpc.GetLastTradesRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Symbol:      string(symbol),
		Limit:       &limit,
	}

	// Call subaccount service with GetLastTrades (returns public taker-only data)
	grpcResp, err := ctx.SubaccountClient.GetLastTrades(ctx, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get last trades from subaccount service",
			"error", err,
			"symbol", req.Symbol,
		)

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to retrieve last trades", err)
	}

	// Transform gRPC response to API response
	trades := make([]PublicTrade, 0, len(grpcResp.Trades))
	for _, grpcTrade := range grpcResp.Trades {

		tradedAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcTrade.TradedAt)

		side, err := snx_lib_api_handlers_utils.TradeDirectionToSide(grpcTrade.Direction)
		if err != nil {
			ctx.Logger.Error("Unrecognized trade direction in persisted data", "direction", grpcTrade.Direction, "trade_id", grpcTrade.Id)
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Internal error: unrecognized trade direction", err)
		}

		// Determine if maker (based on fill type)
		isMaker := snx_lib_core.IsMakerFillType(grpcTrade.FillType)

		trade := PublicTrade{
			TradeId:   snx_lib_api_types.TradeIdFromUintUnvalidated(grpcTrade.Id),
			Symbol:    Symbol(grpcTrade.Symbol),
			Side:      side,
			Price:     snx_lib_api_types.PriceFromStringUnvalidated(grpcTrade.FilledPrice),
			Quantity:  snx_lib_api_types.QuantityFromStringUnvalidated(grpcTrade.FilledQuantity),
			Timestamp: tradedAt,
			IsMaker:   isMaker,
		}

		trades = append(trades, trade)
	}

	ctx.Logger.Info("Last trades retrieved successfully",
		"returned_count", len(trades),
		API_WKS_limit, req.Limit,
		API_WKS_symbol, req.Symbol,
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, GetLastTradesResponse{
		Trades: trades,
	})
}
