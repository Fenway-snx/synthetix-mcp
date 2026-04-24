package trade

import (
	"time"

	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getOrdersHistory
																				^
	                                                            should be     getOrderHistory
*/

type PlaceOrderRequest struct {
	ClientOrderId ClientOrderId `json:"clientOrderId,omitempty"`
	StartTime     Timestamp     `json:"startTime,omitempty"`
	EndTime       Timestamp     `json:"endTime,omitempty"`
	FromTime      Timestamp     `json:"fromTime,omitempty"` // Deprecated: use startTime
	ToTime        Timestamp     `json:"toTime,omitempty"`   // Deprecated: use endTime
	Limit         int32         `json:"limit"`
	Side          string        `json:"side"`
	Symbol        Symbol        `json:"symbol"`
	StatusFilter  []string      `json:"status"`
	Type          string        `json:"type"`
}

type GetOrderHistoryResponseItem struct {
	OrderId                 OrderId          `json:"order"`          // order (paired)
	DEPRECATED_VenueOrderId VenueOrderId     `json:"orderId"`        // [DEPRECATED] // TODO: SNX-4911
	Symbol                  Symbol           `json:"symbol"`         // Trading pair symbol (e.g., "BTC-USD")
	Side                    string           `json:"side"`           // Side (BUY/SELL)
	Type                    string           `json:"type"`           // Type (LIMIT, MARKET, STOP_LOSS, etc.)
	Quantity                Quantity         `json:"quantity"`       // Quantity
	Price                   Price            `json:"price"`          // Price
	Status                  string           `json:"status"`         // Status (NEW, PARTIALLY_FILLED, FILLED, etc.)
	TimeInForce             string           `json:"timeInForce"`    // Time in force (GTC, IOC, FOK)
	CreatedTime             Timestamp        `json:"createdTime"`    // Creation timestamp
	UpdateTime              Timestamp        `json:"updateTime"`     // Last update timestamp
	FilledQuantity          Quantity         `json:"filledQuantity"` // Quantity that has been filled
	FilledPrice             Price            `json:"filledPrice"`    // Average filled price
	TriggeredByLiquidation  bool             `json:"triggeredByLiquidation"`
	ReduceOnly              bool             `json:"reduceOnly"`
	PostOnly                bool             `json:"postOnly"`
	TriggerPrice            Price            `json:"triggerPrice,omitempty"`
	TriggerPriceType        TriggerPriceType `json:"triggerPriceType,omitempty"`
	ExpiresAt               *Timestamp       `json:"expiresAt,omitempty"`
	CancelReason            string           `json:"cancelReason,omitempty"`
}

type GetOrderHistoryResponse []GetOrderHistoryResponseItem

// Handler for "getOrderHistory".
//
//dd:span
func Handle_getOrderHistory(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req PlaceOrderRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", map[string]string{"error": err.Error()})
	}
	if req.Symbol != "" {
		normalizedSymbol, err := snx_lib_api_validation.ValidateAndNormalizeSymbol(req.Symbol)
		if err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		}
		req.Symbol = normalizedSymbol
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.Side, snx_lib_api_validation.MaxEnumFieldLength, "side"); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.Type, snx_lib_api_validation.MaxEnumFieldLength, "type"); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}
	for _, statusFilter := range req.StatusFilter {
		if err := snx_lib_api_validation.ValidateStringMaxLength(statusFilter, snx_lib_api_validation.MaxEnumFieldLength, "status"); err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		}
	}
	startTime, endTime, err := snx_lib_api_handlers_utils.CoalesceTimeRange(req.StartTime, req.EndTime, req.FromTime, req.ToTime)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}
	if req.ClientOrderId != ClientOrderId_Empty {
		validatedCLOID, err := snx_lib_api_types.ValidateClientOrderId(req.ClientOrderId)
		if err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		}
		req.ClientOrderId = validatedCLOID
	}
	const orderHistoryMaxDuration = time.Hour * 24 * 7
	if startTime != 0 && endTime != 0 {
		if startTime > endTime {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "startTime must be before endTime", nil)
		}
		if endTime.Sub(startTime) > orderHistoryMaxDuration {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "time range cannot exceed 7 days", nil)
		}
	}

	grpcResp, err := ctx.SubaccountClient.GetOrderHistory(ctx.Context, &v4grpc.GetOrderHistoryRequest{
		SubAccountId:  int64(ctx.SelectedAccountId),
		StartTime:     snx_lib_api_types.TimestampToTimestampPBOrNil(startTime),
		EndTime:       snx_lib_api_types.TimestampToTimestampPBOrNil(endTime),
		Limit:         &req.Limit,
		StatusFilter:  req.StatusFilter,
		IsPretty:      true,
		ClientOrderId: string(req.ClientOrderId),
	})
	if err != nil {
		failMessage := "Failed to get order history"

		ctx.Logger.Error(failMessage, "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
	}

	ordersResponse := make([]GetOrderHistoryResponseItem, len(grpcResp.Orders))
	for i, order := range grpcResp.Orders {

		orderId := snx_lib_api_types.OrderIdFromGRPCOrderIdUnvalidated(order.OrderId)

		createdAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(order.CreatedAt)
		updatedAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(order.UpdatedAt)
		expiresAt := snx_lib_api_types.TimestampPtrFromTimestampPBOrNil(order.ExpiresAt)

		ordersResponse[i] = GetOrderHistoryResponseItem{
			OrderId:                 orderId,
			DEPRECATED_VenueOrderId: orderId.VenueId, // [DEPRECATED] // TODO: SNX-4911
			Symbol:                  Symbol(order.Symbol),
			Side:                    order.Side,
			Type:                    order.Type,
			Quantity:                snx_lib_api_types.QuantityFromStringUnvalidated(order.Quantity),
			Price:                   snx_lib_api_types.PriceFromStringUnvalidated(order.Price),
			Status:                  order.Status,
			TimeInForce:             order.TimeInForce,
			CreatedTime:             createdAt,
			UpdateTime:              updatedAt,
			FilledQuantity:          snx_lib_api_types.QuantityFromStringUnvalidated(order.FilledQuantity),
			FilledPrice:             snx_lib_api_types.PriceFromStringUnvalidated(order.FilledPrice),
			TriggeredByLiquidation:  order.TriggeredByLiquidation,
			ReduceOnly:              order.ReduceOnly,
			PostOnly:                order.PostOnly,
			TriggerPrice:            snx_lib_api_types.PriceFromStringUnvalidated(order.GetTriggerPrice()),
			TriggerPriceType:        TriggerPriceType(order.GetTriggerPriceType()),
			ExpiresAt:               expiresAt,
			CancelReason:            order.GetCancelReason(),
		}
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, ordersResponse)
}
