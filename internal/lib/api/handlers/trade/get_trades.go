package trade

import (
	"fmt"
	"strconv"
	"time"

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
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getTrades
*/

type GetTradeHistoryRequest struct {
	Account   string       `json:"account"`   // Optional: vault address for delegated trading
	Symbol    Symbol       `json:"symbol"`    // Optional: filter by market symbol
	Limit     int          `json:"limit"`     // Optional: max number of trades (default 100, max 1000)
	Offset    int          `json:"offset"`    // Optional: pagination offset (default 0)
	StartTime Timestamp    `json:"startTime"` // Optional: start time (milliseconds)
	EndTime   Timestamp    `json:"endTime"`   // Optional: end time (milliseconds)
	OrderId   VenueOrderId `json:"orderId"`   // Optional: filter by venue order ID
}

// Represents a single trade in the response.
type GetTradesResponseItem struct {
	TradeId                 TradeId                     `json:"tradeId"`
	OrderId                 OrderId                     `json:"order"`       // order (paired)
	DEPRECATED_VenueOrderId VenueOrderId                `json:"orderId"`     // [DEPRECATED] // TODO: SNX-4911
	Symbol                  Symbol                      `json:"symbol"`      // Market symbol
	OrderType               snx_lib_api_types.OrderType `json:"orderType"`   // Order type for order
	Side                    string                      `json:"side"`        // Trade side: "buy" or "sell"
	Price                   Price                       `json:"price"`       // Execution price
	Quantity                Quantity                    `json:"quantity"`    // Executed quantity
	RealizedPnl             string                      `json:"realizedPnl"` // Realized P&L from position closure
	Fee                     string                      `json:"fee"`         // Trading fee amount
	FeeRate                 string                      `json:"feeRate"`     // Fee rate applied
	Timestamp               Timestamp                   `json:"timestamp"`   // Execution timestamp in milliseconds
	Maker                   bool                        `json:"maker"`       // True if maker order
	ReduceOnly              bool                        `json:"reduceOnly"`  // True if trade reduced existing position
	MarkPrice               Price                       `json:"markPrice"`   // Mark price at time of trade
	EntryPrice              Price                       `json:"entryPrice"`  // Position (average) entry price at time of trade
	TriggeredByLiquidation  bool                        `json:"triggeredByLiquidation"`
	Direction               string                      `json:"direction"`
	PostOnly                bool                        `json:"postOnly"`
}

// Liquidation represents liquidation details for a trade
type Liquidation struct {
	LiquidatedUser WalletAddress `json:"liquidatedUser"` // Address of liquidated account
	MarkPrice      Price         `json:"markPrice"`      // Mark price at liquidation
	Method         string        `json:"method"`         // Liquidation method: "market" or "liquidator"
}

// GetTradesResponse represents the getTrades response data
type GetTradesResponse struct {
	Trades  []GetTradesResponseItem `json:"trades"`  // Array of trades
	HasMore bool                    `json:"hasMore"` // Whether more trades exist
	Total   int                     `json:"total"`   // Total number of trades matching query
}

const tradesEndpointMaxDuration = 30 * 24 * time.Hour

func validateGetTradesRequest(req *GetTradeHistoryRequest) error {
	if req.Account != "" {
		if err := snx_lib_api_validation.ValidateStringMaxLength(req.Account, snx_lib_api_validation.MaxEthAddressLength, "account"); err != nil {
			return err
		}
	}
	if req.Symbol != "" {
		normalizedSymbol, err := snx_lib_api_validation.ValidateAndNormalizeSymbol(req.Symbol)
		if err != nil {
			return err
		}
		req.Symbol = normalizedSymbol
	}
	// Validate limit
	if err := snx_lib_api_validation.ValidateNonNegative(req.Limit, API_WKS_limit); err != nil {
		return err
	}
	if err := snx_lib_api_validation.ValidateMaxLimit(req.Limit, 1000, API_WKS_limit); err != nil {
		return err
	}

	// Validate offset
	if err := snx_lib_api_validation.ValidateNonNegative(req.Offset, API_WKS_offset); err != nil {
		return err
	}

	// Validate timestamp range (timestamps are in milliseconds per API spec)
	if err := snx_lib_api_validation.ValidateTimestampRange(req.StartTime, req.EndTime, tradesEndpointMaxDuration, API_WKS_trades); err != nil {
		return err
	}

	// Validate orderId if provided — use the canonical VenueOrderId rules
	// (rejects zero, negative, non-numeric, and values > 2^63-1)
	if req.OrderId != "" {
		parsed, err := strconv.ParseUint(string(req.OrderId), 10, 64)
		if err != nil {
			return fmt.Errorf("%s must be a valid positive integer", API_WKS_orderId)
		}
		if _, err := snx_lib_api_types.VenueOrderIdFromUint(parsed); err != nil {
			return fmt.Errorf("%s: %w", API_WKS_orderId, err)
		}
	}

	return nil
}

// Handler for "getTrades".
//
//dd:span
func Handle_getTrades(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req GetTradeHistoryRequest
	if err := mapstructure.Decode(params, &req); err != nil {
		ctx.Logger.Error("Failed to decode getTrades request", "error", err)
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request format", nil)
	}

	// Validate request
	if err := validateGetTradesRequest(&req); err != nil {
		ctx.Logger.Error("Trade request snx_lib_api_validation.failed", "error", err)
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Determine subaccount ID
	subAccountId := ctx.SelectedAccountId
	if req.Account != "" {
		// TODO: Validate delegated access to the specified account
		// For now, just use the signing account
		ctx.Logger.Warn("Account parameter specified but delegation not yet implemented", "account", req.Account)
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Create gRPC request

	var subAccountIdInt int64 = int64(subAccountId)
	grpcReq := &v4grpc.GetTradeHistoryRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: &subAccountIdInt,
	}

	// Only set timestamps if they have meaningful values (non-zero)
	// This allows the gRPC service to apply its own defaults when fields are nil

	now := snx_lib_api_types.TimestampNow()

	if startTime, endTime, err, failureQualifier := snx_lib_api_handlers_utils.APIStartEndToCoreStartEndPtrs(req.StartTime, req.EndTime, now); err != nil {

		resp := snx_lib_api_json.NewValidationErrorResponse[any](
			ctx.ClientRequestId,
			"Invalid request parameters",
			map[string]string{
				"error":     err.Error(),
				"qualifier": failureQualifier,
			},
		)
		return HTTPStatusCode_400_BadRequest, resp
	} else {

		grpcReq.StartTime = startTime
		grpcReq.EndTime = endTime
	}

	// Add pagination
	offset := int32(req.Offset)
	limit := int32(req.Limit)
	grpcReq.Offset = &offset
	grpcReq.Limit = &limit

	// Add symbol filter if provided
	if req.Symbol != "" {
		sym := string(req.Symbol)
		grpcReq.Symbol = &sym
	}

	// Add order ID filter if provided
	if req.OrderId != "" {
		oid := snx_lib_api_types.VenueOrderIdToUintUnvalidated(req.OrderId)
		grpcReq.VenueOrderId = &oid
	}

	// Call subaccount service
	grpcResp, err := ctx.SubaccountClient.GetTradeHistory(ctx, grpcReq)
	if err != nil {
		failMessage := "Failed to retrieve trade history"

		ctx.Logger.Error(failMessage, "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
	}

	// Transform gRPC response to API response
	trades := make([]GetTradesResponseItem, 0, len(grpcResp.Trades))
	for _, grpcTrade := range grpcResp.Trades {

		orderId := snx_lib_api_types.OrderIdFromGRPCOrderIdUnvalidated(grpcTrade.OrderId)

		side, err := tradeDirectionToSide(grpcTrade.Direction)
		if err != nil {
			ctx.Logger.Error("Unrecognized trade direction in persisted data", "direction", grpcTrade.Direction, "trade_id", grpcTrade.Id)
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Internal error: unrecognized trade direction", err)
		}

		tradedAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcTrade.TradedAt)

		// Determine if maker (based on fill type)
		isMaker := snx_lib_core.IsMakerFillType(grpcTrade.FillType)

		trade := GetTradesResponseItem{
			TradeId:                 snx_lib_api_types.TradeIdFromUintUnvalidated(grpcTrade.Id),
			OrderId:                 orderId,
			DEPRECATED_VenueOrderId: orderId.VenueId, // [DEPRECATED] // TODO: SNX-4911
			Symbol:                  Symbol(grpcTrade.Symbol),
			OrderType:               snx_lib_api_types.OrderType(grpcTrade.OrderType),
			Side:                    side,
			Price:                   snx_lib_api_types.PriceFromStringUnvalidated(grpcTrade.FilledPrice),
			Quantity:                snx_lib_api_types.QuantityFromStringUnvalidated(grpcTrade.FilledQuantity),
			RealizedPnl:             grpcTrade.ClosedPnl,
			Fee:                     grpcTrade.Fee,
			FeeRate:                 grpcTrade.FeeRate,
			Timestamp:               tradedAt,
			Maker:                   isMaker,
			ReduceOnly:              grpcTrade.ReduceOnly,
			MarkPrice:               snx_lib_api_types.PriceFromStringUnvalidated(grpcTrade.MarkPrice),
			EntryPrice:              snx_lib_api_types.PriceFromStringUnvalidated(grpcTrade.EntryPrice),
			TriggeredByLiquidation:  grpcTrade.TriggeredByLiquidation,
			Direction:               grpcTrade.Direction,
			PostOnly:                grpcTrade.PostOnly,
		}

		trades = append(trades, trade)
	}

	actualTotal := int(grpcResp.TotalCount)
	hasMore := grpcResp.HasMore

	// Create response
	response := GetTradesResponse{
		Trades:  trades,
		HasMore: hasMore,
		Total:   actualTotal,
	}

	ctx.Logger.Info("GetTrades request completed",
		"sub_account_id", subAccountId,
		"returned_count", len(trades),
		"total_count", actualTotal,
		"has_more", hasMore)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, response)
}
