package trade

import (
	"strconv"

	"github.com/go-viper/mapstructure/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type GetTradesForPositionAPIRequest struct {
	PositionId string `json:"positionId"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

type GetTradesForPositionResponse struct {
	Trades  []GetTradesResponseItem `json:"trades"`
	HasMore bool                    `json:"hasMore"`
}

// Handler for "getTradesForPosition".
//
//dd:span
func Handle_getTradesForPosition(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subaccountId is required", nil)
	}

	var req GetTradesForPositionAPIRequest
	if err := mapstructure.Decode(params, &req); err != nil {
		ctx.Logger.Error("Failed to decode getTradesForPosition request", "error", err)
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request format", nil)
	}

	if req.PositionId == "" {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "positionId is required", nil)
	}

	positionId, err := strconv.ParseUint(req.PositionId, 10, 64)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "positionId must be a valid numeric value", nil)
	}

	// Validate limit
	if err := snx_lib_api_validation.ValidateNonNegative(req.Limit, API_WKS_limit); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}
	if err := snx_lib_api_validation.ValidateMaxLimit(req.Limit, 1000, API_WKS_limit); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	// Validate offset
	if err := snx_lib_api_validation.ValidateNonNegative(req.Offset, API_WKS_offset); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 100
	}

	subAccountID := ctx.SelectedAccountId

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	offset := int32(req.Offset)
	limit := int32(req.Limit)
	grpcReq := &v4grpc.GetTradesForPositionRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(subAccountID),
		PositionId:   positionId,
		Offset:       &offset,
		Limit:        &limit,
	}

	grpcResp, err := ctx.SubaccountClient.GetTradesForPosition(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
			ctx.Logger.Debug("Invalid getTradesForPosition request", "error", err)
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid request parameters", nil)
		}

		ctx.Logger.Error("Failed to retrieve trades for position", "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInternalError, "Failed to retrieve trades for position", nil)
	}

	trades := make([]GetTradesResponseItem, 0, len(grpcResp.Trades))
	for _, grpcTrade := range grpcResp.Trades {

		orderId := snx_lib_api_types.OrderIdFromGRPCOrderIdUnvalidated(grpcTrade.OrderId)

		side, err := tradeDirectionToSide(grpcTrade.Direction)
		if err != nil {
			ctx.Logger.Error("Unrecognized trade direction in persisted data", "direction", grpcTrade.Direction, "trade_id", grpcTrade.Id)
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInternalError, "Internal error: unrecognized trade direction", nil)
		}

		tradedAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcTrade.TradedAt)

		isMaker := snx_lib_core.IsMakerFillType(grpcTrade.FillType)

		trade := GetTradesResponseItem{
			TradeId:                 snx_lib_api_types.TradeIdFromUintUnvalidated(grpcTrade.Id),
			OrderId:                 orderId,
			DEPRECATED_VenueOrderId: orderId.VenueId,
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

	response := GetTradesForPositionResponse{
		Trades:  trades,
		HasMore: grpcResp.HasMore,
	}

	ctx.Logger.Info("GetTradesForPosition request completed",
		"has_more", grpcResp.HasMore,
		"position_id", positionId,
		"returned_count", len(trades),
		"sub_account_id", subAccountID)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, response)
}
