package trade

import (
	"strconv"

	"github.com/go-viper/mapstructure/v2"
	angols_slices "github.com/synesissoftware/ANGoLS/slices"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getPositions
*/

type GetPositionsRequest struct {
	Status    string    `json:"status"`
	Symbol    Symbol    `json:"symbol"`
	StartTime Timestamp `json:"startTime,omitempty"`
	EndTime   Timestamp `json:"endTime,omitempty"`
	FromTime  Timestamp `json:"fromTime,omitempty"` // Deprecated: use startTime
	ToTime    Timestamp `json:"toTime,omitempty"`   // Deprecated: use endTime
	Limit     int32     `json:"limit,omitempty"`
	Offset    int32     `json:"offset,omitempty"`
	SortBy    string    `json:"sortBy"`
	SortOrder string    `json:"sortOrder"`
}

// c/w `trade.Position`
type GetPositionResponseItem struct {
	ADLBucket                     int64          `json:"adlBucket"`                    // ADL priority bucket (1–5, 5 = highest risk)
	PositionId                    string         `json:"positionId"`                   // e.g. "pos_12346"
	SubAccountId                  SubAccountId   `json:"subAccountId"`                 // e.g. "1867542890123456789"
	Symbol                        Symbol         `json:"symbol"`                       // e.g. "BTC-USDT"
	Side                          string         `json:"side"`                         // e.g. "short"
	EntryPrice                    Price          `json:"entryPrice,omitempty"`         // e.g. "42000.00"
	Quantity                      Quantity       `json:"quantity,omitempty"`           // e.g. "0.1000"
	RealizedPnl                   string         `json:"realizedPnl,omitempty"`        // e.g. "25.00"
	UnrealizedPnl                 string         `json:"unrealizedPnl,omitempty"`      // e.g. "-12.50"
	UsedMargin                    string         `json:"usedMargin,omitempty"`         // e.g. "840.00"
	MaintenanceMargin             string         `json:"maintenanceMargin,omitempty"`  // e.g. "420.00"
	LiquidationPrice              Price          `json:"liquidationPrice,omitempty"`   // e.g. "45000.00"
	Status                        string         `json:"status,omitempty"`             // e.g. "open"
	NetFunding                    string         `json:"netFunding,omitempty"`         // e.g. "8.50"
	TakeProfitOrderIds            []OrderId      `json:"takeProfitOrders,omitempty"`   // e.g. []
	DEPRECATED_TakeProfitOrderIDs []VenueOrderId `json:"takeProfitOrderIds,omitempty"` // [DEPRECATED] // TODO: SNX-4911
	StopLossOrderIds              []OrderId      `json:"stopLossOrders,omitempty"`     // e.g. []
	DEPRECATED_StopLossOrderIDs   []VenueOrderId `json:"stopLossOrderIds,omitempty"`   // [DEPRECATED] // TODO: SNX-4911
	UpdatedAt                     Timestamp      `json:"updatedAt"`                    // e.g. 1735689600000
	CreatedAt                     Timestamp      `json:"createdAt"`                    // e.g. 1735680000000
}

type GetPositionsResponse []GetPositionResponseItem

// Handler for "getPositions".
//
//dd:span
func Handle_getPositions(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req GetPositionsRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	if req.Symbol != "" {
		normalizedSymbol, err := snx_lib_api_validation.ValidateAndNormalizeSymbol(req.Symbol)
		if err != nil {
			resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
			return HTTPStatusCode_400_BadRequest, resp
		}
		req.Symbol = normalizedSymbol
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.Status, snx_lib_api_validation.MaxEnumFieldLength, "status"); err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.SortBy, snx_lib_api_validation.MaxEnumFieldLength, "sortBy"); err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.SortOrder, snx_lib_api_validation.MaxEnumFieldLength, "sortOrder"); err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	startTime, endTime, err := snx_lib_api_handlers_utils.CoalesceTimeRange(req.StartTime, req.EndTime, req.FromTime, req.ToTime)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	grpcReq := &v4grpc.GetPositionsRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(ctx.SelectedAccountId),
		StartTime:    snx_lib_api_types.TimestampToTimestampPBOrNil(startTime),
		EndTime:      snx_lib_api_types.TimestampToTimestampPBOrNil(endTime),
		Offset:       &req.Offset,
	}
	if req.Limit > 0 {
		grpcReq.Limit = &req.Limit
	}

	grpcResp, err := ctx.SubaccountClient.GetPositions(ctx.Context, grpcReq)
	if err != nil {
		failMessage := "Failed to get positions"

		ctx.Logger.Error(failMessage, "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
	}

	positions := make([]GetPositionResponseItem, 0, len(grpcResp.Positions))
	for _, position := range grpcResp.Positions {

		// we prepare the fields separately before pushing into structure,
		// including several lists:
		//
		// - takeProfitOrderIds : []OrderId
		// - stopLossOrderIds : []OrderId
		//
		// - deprecated_takeProfitOrderIDs : []VenueOrderId
		// - deprecated_stopLossOrderIDs : []VenueOrderId

		positionId := strconv.FormatUint(position.Id, 10)

		sid := snx_lib_api_types.SubAccountIdFromIntUnvalidated(position.SubAccountId)

		var takeProfitOrderIds []OrderId
		var stopLossOrderIds []OrderId

		fn_grpc_OrderId_to_api_OrderId := func(_ int, input_item **v4grpc.OrderId) (OrderId, error) {

			orderId := snx_lib_api_types.OrderIdFromGRPCOrderIdUnvalidated(*input_item)

			return orderId, nil
		}

		takeProfitOrderIds, _ = angols_slices.CollectSlice(position.TakeProfitOrderIds, fn_grpc_OrderId_to_api_OrderId)
		stopLossOrderIds, _ = angols_slices.CollectSlice(position.StopLossOrderIds, fn_grpc_OrderId_to_api_OrderId)

		fn_grpc_OrderId_to_api_VenueOrderId := func(_ int, input_item *OrderId) (VenueOrderId, error) {
			return input_item.VenueId, nil
		}

		deprecated_takeProfitOrderIDs, _ := angols_slices.CollectSlice(takeProfitOrderIds, fn_grpc_OrderId_to_api_VenueOrderId)
		deprecated_stopLossOrderIDs, _ := angols_slices.CollectSlice(stopLossOrderIds, fn_grpc_OrderId_to_api_VenueOrderId)

		updatedAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(position.UpdatedAt)
		createdAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(position.CreatedAt)

		positions = append(positions, GetPositionResponseItem{
			ADLBucket:                     position.AdlBucket,
			PositionId:                    positionId,
			SubAccountId:                  sid,
			Symbol:                        Symbol(position.Symbol),
			Side:                          position.Side,
			EntryPrice:                    snx_lib_api_types.PriceFromStringUnvalidated(position.EntryPrice),
			Quantity:                      snx_lib_api_types.QuantityFromStringUnvalidated(position.Quantity),
			RealizedPnl:                   position.Pnl,
			UnrealizedPnl:                 position.Upnl,
			UsedMargin:                    position.UsedMargin,
			MaintenanceMargin:             position.MaintenanceMargin,
			LiquidationPrice:              snx_lib_api_types.PriceFromStringUnvalidated(position.LiquidationPrice),
			Status:                        position.Action,                // TODO: confirm this
			NetFunding:                    position.NetPositionFundingPnl, // TODO: confirm this
			TakeProfitOrderIds:            takeProfitOrderIds,
			DEPRECATED_TakeProfitOrderIDs: deprecated_takeProfitOrderIDs, // [DEPRECATED] // TODO: SNX-4911
			StopLossOrderIds:              stopLossOrderIds,
			DEPRECATED_StopLossOrderIDs:   deprecated_stopLossOrderIDs, // [DEPRECATED] // TODO: SNX-4911
			UpdatedAt:                     updatedAt,
			CreatedAt:                     createdAt,
		})
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, positions)
}
