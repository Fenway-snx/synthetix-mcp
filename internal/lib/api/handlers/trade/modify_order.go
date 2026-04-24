package trade

import (
	"errors"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type ModifyOrderResponse struct {
	OrderId            OrderId      `json:"order"`   // order (paired)
	DEPRECATED_OrderId VenueOrderId `json:"orderId"` // [DEPRECATED]
	Status             string       `json:"status,omitempty"`
	Price              Price        `json:"price,omitempty"`    // Order price
	Quantity           Quantity     `json:"quantity,omitempty"` // Order quantity
	TriggerPrice       Price        `json:"triggerPrice,omitempty"`
	CumQty             string       `json:"cumQty,omitempty"`
	AvgPrice           Price        `json:"avgPrice,omitempty"`
	Error              string       `json:"error,omitempty"`
	ErrorCode          string       `json:"errorCode,omitempty"`
	Timestamp          Timestamp    `json:"timestamp"`
}

var (
	errMissingModifyOrderPayload = errors.New("missing validated modifyOrder payload in context")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/modifyOrder
*/

// Handler for "modifyOrder".
//
//dd:span
func Handle_modifyOrder(
	ctx TradeContext,
	_params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var grpcResp *v4grpc.ModifyOrderResponse
	var err error

	switch validated := ctx.ActionPayload().(type) {
	case *ValidatedModifyOrderAction:
		if validated == nil {
			ctx.Logger.Error("Missing validated modifyOrder payload in context")
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingModifyOrderPayload)
		}
		grpcResp, err = ctx.TradingClient.ModifyOrder(ctx, &v4grpc.ModifyOrderRequest{
			TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
			SubAccountId:  int64(ctx.SelectedAccountId),
			VenueOrderId:  snx_lib_api_types.VenueOrderIdToUintUnvalidated(validated.VenueOrderId),
			Quantity:      snx_lib_api_types.QuantityPtrToStringPtr(validated.Payload.Quantity),
			Price:         snx_lib_api_types.PricePtrToStringPtr(validated.Payload.Price),
			TriggerPrice:  snx_lib_api_types.PricePtrToStringPtr(validated.Payload.TriggerPrice),
			RequestId:     ctx.RequestId.String(),
		})
	case *ValidatedModifyOrderByCloidAction:
		if validated == nil {
			ctx.Logger.Error("Missing validated modifyOrder payload in context")
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingModifyOrderPayload)
		}
		grpcResp, err = ctx.TradingClient.ModifyOrderByCloid(ctx, &v4grpc.ModifyOrderByCloidRequest{
			TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
			SubAccountId:  int64(ctx.SelectedAccountId),
			ClientOrderId: snx_lib_api_types.ClientOrderIdToStringUnvalidated(validated.ClientOrderId),
			Quantity:      snx_lib_api_types.QuantityPtrToStringPtr(validated.Payload.Quantity),
			Price:         snx_lib_api_types.PricePtrToStringPtr(validated.Payload.Price),
			TriggerPrice:  snx_lib_api_types.PricePtrToStringPtr(validated.Payload.TriggerPrice),
			RequestId:     ctx.RequestId.String(),
		})
	default:
		ctx.Logger.Error("Missing validated modifyOrder payload in context")
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingModifyOrderPayload)
	}

	if err != nil {
		ctx.Logger.Debug("Failed to modify order",
			"error", err,
			"data", ctx.ActionPayload(),
		)

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to modify order", err)
	}

	ctx.Logger.Debug("Received modify order response from trading service",
		"data", grpcResp,
	)

	now := snx_lib_utils_time.Now()

	orderId := snx_lib_api_types.OrderIdFromGRPCOrderIdUnvalidated(grpcResp.OrderId)

	status := strings.ToLower(grpcResp.Status.String())

	var price Price
	if grpcResp.Price != nil {
		price = snx_lib_api_types.PriceFromStringUnvalidated(*grpcResp.Price)
	}

	var quantity Quantity
	if grpcResp.Quantity != nil {
		quantity = snx_lib_api_types.QuantityFromStringUnvalidated(*grpcResp.Quantity)
	}

	var triggerPrice Price
	if grpcResp.TriggerPrice != nil {
		triggerPrice = snx_lib_api_types.PriceFromStringUnvalidated(*grpcResp.TriggerPrice)
	}

	var cumQty string
	if grpcResp.CumulativeFillQty != nil {
		cumQty = *grpcResp.CumulativeFillQty
	}

	var avgPrice Price
	if grpcResp.AveragePrice != nil {
		avgPrice = snx_lib_api_types.PriceFromStringUnvalidated(*grpcResp.AveragePrice)
	}

	timestamp := Timestamp(now.UnixMilli())

	var errorMsg string
	if grpcResp.ErrorMessage != nil {
		errorMsg = *grpcResp.ErrorMessage
	}

	var errorCode string
	if grpcResp.ErrorCode != nil {
		errorCode = *grpcResp.ErrorCode
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, ModifyOrderResponse{
		OrderId:            orderId,
		DEPRECATED_OrderId: orderId.VenueId,
		Status:             status,
		Price:              price,
		Quantity:           quantity,
		TriggerPrice:       triggerPrice,
		CumQty:             cumQty,
		AvgPrice:           avgPrice,
		Error:              errorMsg,
		ErrorCode:          errorCode,
		Timestamp:          timestamp,
	})
}
