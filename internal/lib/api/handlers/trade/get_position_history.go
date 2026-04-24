package trade

import (
	"strconv"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const positionHistoryEndpointMaxDuration = 30 * 24 * time.Hour

type GetPositionHistoryRequest struct {
	Symbol    Symbol    `json:"symbol"`
	StartTime Timestamp `json:"startTime,omitempty"`
	EndTime   Timestamp `json:"endTime,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

type GetPositionHistoryResponseItem struct {
	PositionId      string    `json:"positionId"`
	Symbol          Symbol    `json:"symbol"`
	Side            string    `json:"side"`
	EntryPrice      Price     `json:"entryPrice"`
	Quantity        Quantity  `json:"quantity"`
	ClosePrice      Price     `json:"closePrice"`
	CloseReason     string    `json:"closeReason"`
	Leverage        *uint32   `json:"leverage,omitempty"`
	RealizedPnl     string    `json:"realizedPnl"`
	AccumulatedFees string    `json:"accumulatedFees"`
	NetFunding      string    `json:"netFunding"`
	ClosedAt        Timestamp `json:"closedAt"`
	CreatedAt       Timestamp `json:"createdAt"`
	TradeId         TradeId   `json:"tradeId"`
}

type GetPositionHistoryResponse struct {
	Positions []GetPositionHistoryResponseItem `json:"positions"`
	HasMore   bool                             `json:"hasMore"`
}

// Handler for "getPositionHistory".
//
//dd:span
func Handle_getPositionHistory(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subaccountId is required", nil)
	}

	var req GetPositionHistoryRequest
	if err := mapstructure.Decode(params, &req); err != nil {
		ctx.Logger.Error("Failed to decode getPositionHistory request", "error", err)
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request format", nil)
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

	subAccountId := ctx.SelectedAccountId

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	grpcReq := &v4grpc.GetPositionHistoryRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(subAccountId),
	}

	// Validate timestamp range
	if err := snx_lib_api_validation.ValidateTimestampRange(req.StartTime, req.EndTime, positionHistoryEndpointMaxDuration, "positionHistory"); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	// Parse timestamps
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

	// Call subaccount service
	grpcResp, err := ctx.SubaccountClient.GetPositionHistory(ctx, grpcReq)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
			ctx.Logger.Debug("Invalid position history request", "error", err)
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](
				ctx.ClientRequestId,
				"Invalid request parameters",
				nil,
			)
		}

		ctx.Logger.Error("Failed to retrieve position history", "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](
			ctx.ClientRequestId,
			"Failed to retrieve position history",
			err,
		)
	}

	// Transform gRPC response to API response
	positions := make([]GetPositionHistoryResponseItem, 0, len(grpcResp.Positions))
	for _, p := range grpcResp.Positions {
		closedAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(p.ClosedAt)
		createdAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(p.CreatedAt)

		item := GetPositionHistoryResponseItem{
			PositionId:      strconv.FormatUint(p.PositionId, 10),
			Symbol:          Symbol(p.Symbol),
			Side:            p.Side,
			EntryPrice:      snx_lib_api_types.PriceFromStringUnvalidated(p.EntryPrice),
			Quantity:        snx_lib_api_types.QuantityFromStringUnvalidated(p.Quantity),
			ClosePrice:      snx_lib_api_types.PriceFromStringUnvalidated(p.ClosePrice),
			CloseReason:     p.CloseReason,
			RealizedPnl:     p.RealizedPnl,
			AccumulatedFees: p.AccumulatedFees,
			NetFunding:      p.NetFundingPnl,
			ClosedAt:        closedAt,
			CreatedAt:       createdAt,
			TradeId:         snx_lib_api_types.TradeIdFromUintUnvalidated(uint64(p.TradeId)),
		}
		if p.Leverage != nil {
			lev := *p.Leverage
			item.Leverage = &lev
		}
		positions = append(positions, item)
	}

	response := GetPositionHistoryResponse{
		Positions: positions,
		HasMore:   grpcResp.HasMore,
	}

	ctx.Logger.Info("GetPositionHistory request completed",
		"has_more", grpcResp.HasMore,
		"returned_count", len(positions),
		"sub_account_id", subAccountId,
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, response)
}
