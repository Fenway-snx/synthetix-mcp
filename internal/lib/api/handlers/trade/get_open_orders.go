package trade

import (
	"encoding/json"

	"github.com/go-viper/mapstructure/v2"
	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getOpenOrders
*/

const (
	failedToGetOpenOrders  = "Failed to get open orders"
	failedToGetTimeInForce = "Failed to get time-in-force"
)

type OpenOrderRequest struct {
	ClientOrderId ClientOrderId `json:"clientOrderId,omitempty"`
	Symbol        Symbol        `json:"symbol"`
	Side          string        `json:"side"`
	Type          string        `json:"type"`
	Status        string        `json:"status"`
	Limit         int32         `json:"limit"`
	Offset        int32         `json:"offset"`
}

// TWAP-specific execution details, present only for orders with type "twap".
type TWAPDetails struct {
	AveragePrice    Price  `json:"averagePrice"`
	IntervalMs      int64  `json:"intervalMs"`
	TotalTrades     int    `json:"totalTrades"`
	TradesFilled    int    `json:"tradesFilled"`
	TotalFees       string `json:"totalFees"`
	StartedAtMs     int64  `json:"startedAtMs"`
	TotalDurationMs int64  `json:"totalDurationMs"`
}

// Represents what an open order looks like when returned from the API.
type GetOpenOrdersResponseItem struct {
	OrderId                      OrderId          `json:"order"`                       // order (paired)
	DEPRECATED_VenueOrderId      VenueOrderId     `json:"orderId"`                     // [DEPRECATED] // TODO: SNX-4911
	Symbol                       Symbol           `json:"symbol"`                      // Trading pair symbol (e.g., "BTC-USD")
	Side                         Side             `json:"side"`                        // Order side (BUY/SELL)
	Type                         string           `json:"type"`                        // Order type (LIMIT, MARKET, STOP_LOSS, etc.)
	Quantity                     Quantity         `json:"quantity"`                    // Order quantity
	Price                        Price            `json:"price"`                       // Order price
	TriggerPrice                 Price            `json:"triggerPrice"`                // Trigger price for conditional orders
	TriggerPriceType             TriggerPriceType `json:"triggerPriceType"`            // Trigger price type (e.g., mark, last, index)
	TimeInForce                  TimeInForce      `json:"timeInForce"`                 // Time in force (GTC, IOC, FOK)
	ReduceOnly                   bool             `json:"reduceOnly"`                  // Whether the order is reduce-only
	PostOnly                     bool             `json:"postOnly"`                    // Whether the order is post-only
	CreatedTime                  Timestamp        `json:"createdTime"`                 // Order creation timestamp
	UpdatedTime                  Timestamp        `json:"updatedTime"`                 // Last update timestamp
	FilledQuantity               Quantity         `json:"filledQuantity"`              // Quantity that has been filled
	TakeProfitOrderId            *OrderId         `json:"takeProfitOrder,omitempty"`   // order (paired)
	DEPRECATED_TakeProfitOrderId VenueOrderId     `json:"takeProfitOrderId,omitempty"` // [DEPRECATED] // TODO: SNX-4911
	StopLossOrderId              *OrderId         `json:"stopLossOrder,omitempty"`     // order (paired)
	DEPRECATED_StopLossOrderId   VenueOrderId     `json:"stopLossOrderId,omitempty"`   // [DEPRECATED] // TODO: SNX-4911
	ClosePosition                bool             `json:"closePosition"`
	ExpiresAt                    *Timestamp       `json:"expiresAt,omitempty"`
	TwapDetails                  *TWAPDetails     `json:"twapDetails,omitempty"` // Present only for TWAP orders
}

// Represents the response for open orders data.
type GetOpenOrdersResponse []GetOpenOrdersResponseItem

// Handler for "getOpenOrders".
//
//dd:span
func Handle_getOpenOrders(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req OpenOrderRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
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
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.Status, snx_lib_api_validation.MaxEnumFieldLength, "status"); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}
	if req.ClientOrderId != ClientOrderId_Empty {
		validatedCLOID, err := snx_lib_api_types.ValidateClientOrderId(req.ClientOrderId)
		if err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
		}
		req.ClientOrderId = validatedCLOID
	}
	if req.Limit == 0 {
		req.Limit = 50
	}

	symbol_str := string(req.Symbol)
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.GetOpenOrders(ctx.Context, &v4grpc.GetOpenOrdersRequest{
		TimestampMs:   timestamp_ms,
		TimestampUs:   timestamp_us,
		SubAccountId:  int64(ctx.SelectedAccountId),
		Symbol:        &symbol_str,
		Limit:         &req.Limit,
		Offset:        &req.Offset,
		ClientOrderId: string(req.ClientOrderId),
	})
	if err != nil {
		failMessage := "Failed to get open orders"

		ctx.Logger.Error(failMessage, "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
	}

	ordersResponse := make(GetOpenOrdersResponse, len(grpcResp.Orders))
	for i, order := range grpcResp.Orders {

		orderId := snx_lib_api_types.OrderIdFromGRPCOrderIdUnvalidated(order.OrderId)

		timeInForce, err := snx_lib_api_types.TimeInForceFromGRPC(order.TimeInForce)
		if err != nil {
			ctx.Logger.Error(failedToGetTimeInForce,
				"error", err,
			)
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to convert time-in-force", err)
		}

		remainingQuantity, err := shopspring_decimal.NewFromString(order.RemainingQuantity)
		if err != nil {
			failMessage := "Failed to convert remaining quantity to decimal"

			ctx.Logger.Error(failMessage, "error", err)
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
		}
		quantity, err := shopspring_decimal.NewFromString(order.Quantity)
		if err != nil {
			failMessage := "Failed to convert quantity to decimal"

			ctx.Logger.Error(failMessage, "error", err)
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
		}
		filledQuantity := quantity.Sub(remainingQuantity).String()

		createdTime := snx_lib_api_types.TimestampFromTimestampPBOrZero(order.CreatedAt)
		updatedTime := snx_lib_api_types.TimestampFromTimestampPBOrZero(order.UpdatedAt)
		expiresAt := snx_lib_api_types.TimestampPtrFromTimestampPBOrNil(order.ExpiresAt)

		var deprecated_takeProfitVenueOrderId VenueOrderId
		takeProfitOrderId := snx_lib_api_types.OrderIdPtrOrNilFromGRPCOrderIdUnvalidated(order.TakeProfitOrderId)
		if takeProfitOrderId != nil {

			deprecated_takeProfitVenueOrderId = takeProfitOrderId.VenueId
		}

		var deprecated_stopLossVenueOrderId VenueOrderId
		stopLossOrderId := snx_lib_api_types.OrderIdPtrOrNilFromGRPCOrderIdUnvalidated(order.StopLossOrderId)
		if stopLossOrderId != nil {

			deprecated_stopLossVenueOrderId = stopLossOrderId.VenueId
		}

		var twapDetails *TWAPDetails
		if rawState := order.GetTwapExecutionState(); rawState != "" {
			var xs snx_lib_core.TWAPExecutionState
			if err := json.Unmarshal([]byte(rawState), &xs); err != nil {
				ctx.Logger.Warn("Failed to unmarshal twap_execution_state",
					"error", err,
					"order_id", order.OrderId,
				)
			} else {
				twapDetails = &TWAPDetails{
					AveragePrice: snx_lib_api_types.PriceFromDecimalUnvalidated(
						xs.AveragePrice(int32(order.PriceExponent)),
					),
					IntervalMs:      xs.ChunkIntervalMs,
					TotalTrades:     xs.ChunksTotal,
					TradesFilled:    xs.ChunksFilled,
					TotalFees:       xs.TotalFees.String(),
					StartedAtMs:     xs.StartedAtMs,
					TotalDurationMs: int64(xs.ChunksTotal) * xs.ChunkIntervalMs,
				}
			}
		}

		ordersResponse[i] = GetOpenOrdersResponseItem{
			OrderId:                      orderId,
			DEPRECATED_VenueOrderId:      orderId.VenueId, // [DEPRECATED] // TODO: SNX-4911
			Symbol:                       Symbol(order.Symbol),
			Side:                         Side(order.Side),
			Type:                         order.Type,
			Quantity:                     snx_lib_api_types.QuantityFromStringUnvalidated(order.Quantity),
			Price:                        snx_lib_api_types.PriceFromStringUnvalidated(order.Price),
			TriggerPrice:                 snx_lib_api_types.PriceFromStringUnvalidated(order.TriggerPrice),
			TriggerPriceType:             TriggerPriceType(order.TriggerPriceType),
			TimeInForce:                  timeInForce,
			ReduceOnly:                   order.ReduceOnly,
			PostOnly:                     order.PostOnly,
			CreatedTime:                  createdTime,
			UpdatedTime:                  updatedTime,
			FilledQuantity:               snx_lib_api_types.QuantityFromStringUnvalidated(filledQuantity),
			TakeProfitOrderId:            takeProfitOrderId,
			DEPRECATED_TakeProfitOrderId: deprecated_takeProfitVenueOrderId, // [DEPRECATED] // TODO: SNX-4911
			StopLossOrderId:              stopLossOrderId,
			DEPRECATED_StopLossOrderId:   deprecated_stopLossVenueOrderId, // [DEPRECATED] // TODO: SNX-4911
			ClosePosition:                order.ClosePosition,
			ExpiresAt:                    expiresAt,
			TwapDetails:                  twapDetails,
		}
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, ordersResponse)
}
