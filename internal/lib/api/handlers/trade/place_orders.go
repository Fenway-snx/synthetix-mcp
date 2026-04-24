package trade

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	minExpiresAt = 10 * time.Second
	maxExpiresAt = 24 * time.Hour
)

var (
	errExpiresAtNeedsToBeAtLeast10SecondsInFuture = errors.New("expiresAt needs to be at least 10 seconds in the future")
	errExpiresAtNeedsToBeLessThan24HoursInFuture  = errors.New("expiresAt needs to be less than 24 hours in the future")
	errInvalidOrderType                           = errors.New("invalid order type")
	errMissingPlaceOrdersPayload                  = errors.New("missing validated placeOrders payload in context")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/placeOrders
*/

// Handler for "placeOrders".
//
//dd:span
func Handle_placeOrders(
	ctx TradeContext,
	_params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Validation already performed at entry point (REST/WebSocket handler)
	validated, ok := ctx.ActionPayload().(*ValidatedPlaceOrdersAction)
	if !ok || validated == nil {
		ctx.Logger.Error("Missing validated placeOrders payload in context")

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingPlaceOrdersPayload)
	}

	// For _now_, we treat rate-limiting and whitelisting as entirely separate
	// aspects.

	if ctx.WhitelistArbitrator != nil {

		walletAddress := ctx.WalletAddress

		// NOTE: THIS IS A TEMPORARY MECHANISM (FOR SNX-5190)
		{
			// Because the context's WalletAddress may be a delegated address, we
			// first look up in Redis to see whether it is known as such. If it is
			// we instead use the

			key := delegateWhitelistKey(string(walletAddress))

			if ctx.Rc != nil && ctx.Rc.IsValid() {

				c := ctx.Context

				rres := ctx.Rc.Get(c, key)

				if rres != nil {

					res, err := rres.Result()

					if err == nil {

						res = strings.TrimSpace(res)

						walletAddress = WalletAddress(res)

						ctx.Logger.Info("obtained primary wallet address for delegate",
							"delegated_wallet_address", snx_lib_core.MaskAddress(string(ctx.WalletAddress)),
							"primary_wallet_address", snx_lib_core.MaskAddress(string(walletAddress)),
						)
					} else {
						if !strings.Contains(err.Error(), "redis: nil") {
							ctx.Logger.Error("failed to read from Redis for wallet address",
								"error", err,
								"wallet_address", snx_lib_core.MaskAddress(string(ctx.WalletAddress)),
							)
						}
					}
				}
			}
		}

		r, err := ctx.WhitelistArbitrator.CanOrdersBePlacedFor(walletAddress)

		if err != nil {
			// TODO: create simple util function to log and return failure (with consistent messages)

			const msg = "failed to obtain whitelist arbitration"

			ctx.Logger.Error(msg,
				"error", err,
				"wallet_address", snx_lib_core.MaskAddress(string(walletAddress)),
			)

			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, msg, err)

		}

		if !r {
			const msg = "failed whitelist arbitration"

			if ctx.WhitelistDiagnostics != nil {

				ctx.WhitelistDiagnostics.NumRejected.Add(1)
			} else {

				ctx.Logger.Warn(msg,
					"wallet_address", snx_lib_core.MaskAddress(string(walletAddress)),
				)
			}

			return HTTPStatusCode_403_Forbidden, snx_lib_api_json.NewErrorResponse[any](
				ctx.ClientRequestId,
				snx_lib_api_json.ErrorCodeForbidden,
				msg,
				nil,
			)
		} else {

			if ctx.WhitelistDiagnostics != nil {
				ctx.WhitelistDiagnostics.NumPermitted.Add(1)
			}
		}
	}

	switch validated.Payload.Grouping {
	// TODO: what does na even means? we should just keep it "nil"/optional
	case GroupingValues_na:
		return handleBatchOrder(ctx, *validated.Payload)
	case GroupingValues_normalTpsl:
		return handleCompoundOrder(ctx, *validated.Payload)
	case GroupingValues_positionsTpsl:
		return handlePositionTPSL(ctx, *validated.Payload)
	case GroupingValues_twap:
		return handleTWAPOrder(ctx, *validated.Payload)
	}

	return handleBatchOrder(ctx, *validated.Payload)
}

func convertJSONOrderToGRPC(
	order snx_lib_api_json.PlaceOrderRequest,
	symbol Symbol,
) (*v4grpc.PlaceOrderRequestItem, error) {
	grpcOrder := &v4grpc.PlaceOrderRequestItem{
		Symbol:   string(symbol),
		Quantity: string(order.Quantity),
	}

	if order.Side == "buy" {
		grpcOrder.Side = v4grpc.Side_BUY
	} else {
		grpcOrder.Side = v4grpc.Side_SELL
	}

	switch order.OrderType {
	case "limitIoc":
		grpcOrder.Type = v4grpc.OrderType_LIMIT
		grpcOrder.Price = snx_lib_api_types.PricePtrToStringPtr(&order.Price)
		tif := v4grpc.TimeInForce_IOC
		grpcOrder.TimeInForce = &tif

	case "limitGtc":
		grpcOrder.Type = v4grpc.OrderType_LIMIT
		grpcOrder.Price = snx_lib_api_types.PricePtrToStringPtr(&order.Price)
		gtc := v4grpc.TimeInForce_GTC
		grpcOrder.TimeInForce = &gtc
		grpcOrder.PostOnly = order.PostOnly

	case "limitGtd":
		grpcOrder.Type = v4grpc.OrderType_LIMIT
		grpcOrder.Price = snx_lib_api_types.PricePtrToStringPtr(&order.Price)
		gtd := v4grpc.TimeInForce_GTD
		grpcOrder.TimeInForce = &gtd
		grpcOrder.PostOnly = order.PostOnly

		if order.ExpiresAt != nil && *order.ExpiresAt > 0 {
			now := snx_lib_utils_time.Now()
			expiresAt := time.Unix(*order.ExpiresAt, 0).UTC()
			if expiresAt.Before(now.Add(minExpiresAt)) {
				return nil, errExpiresAtNeedsToBeAtLeast10SecondsInFuture
			}
			if expiresAt.After(now.Add(maxExpiresAt)) {
				return nil, errExpiresAtNeedsToBeLessThan24HoursInFuture
			}
			grpcOrder.ExpiresAt = timestamppb.New(expiresAt)
		}

	case "market":
		grpcOrder.Type = v4grpc.OrderType_MARKET
	case API_WKS_triggerTp:
		if order.IsTriggerMarket {
			grpcOrder.Type = v4grpc.OrderType_TAKE_PROFIT_MARKET
		} else {
			grpcOrder.Type = v4grpc.OrderType_TAKE_PROFIT
			price := order.Price
			grpcOrder.Price = snx_lib_api_types.PricePtrToStringPtr(&price)
		}

		grpcOrder.TriggerPrice = snx_lib_api_types.PricePtrToStringPtr(&order.TriggerPrice)
		grpcOrder.TriggerPriceType = string(order.TriggerPriceType)
	case API_WKS_triggerSl:
		if order.IsTriggerMarket {
			grpcOrder.Type = v4grpc.OrderType_STOP_MARKET
		} else {
			grpcOrder.Type = v4grpc.OrderType_STOP
			price := order.Price
			grpcOrder.Price = snx_lib_api_types.PricePtrToStringPtr(&price)
		}
		grpcOrder.TriggerPrice = snx_lib_api_types.PricePtrToStringPtr(&order.TriggerPrice)
		grpcOrder.TriggerPriceType = string(order.TriggerPriceType)
	case "twap":
		grpcOrder.Type = v4grpc.OrderType_TWAP
		gtc := v4grpc.TimeInForce_GTC
		grpcOrder.TimeInForce = &gtc
		grpcOrder.TwapParams = &v4grpc.TWAPOrderRequest{
			DurationSeconds: order.DurationSeconds,
			IntervalSeconds: order.IntervalSeconds,
		}
		if order.Price != Price_None {
			grpcOrder.Price = snx_lib_api_types.PricePtrToStringPtr(&order.Price)
		}
	default:
		return nil, fmt.Errorf("%w: %s", errInvalidOrderType, order.OrderType)

	}

	grpcOrder.ReduceOnly = order.ReduceOnly
	grpcOrder.ClosePosition = order.ClosePosition
	grpcOrder.NewClientOrderId = snx_lib_api_types.ClientOrderIdToStringPtrUnvalidated(order.ClientOrderId)

	return grpcOrder, nil
}

func handleBatchOrder(ctx TradeContext, req PlaceOrdersActionPayload) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	grpcOrders := make([]*v4grpc.PlaceOrderRequestItem, 0, len(req.Orders))

	for _, order := range req.Orders {
		grpcOrder, err := convertJSONOrderToGRPC(order, order.Symbol)
		if err != nil {
			ctx.Logger.Error("Failed to convert order", "error", err)
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
				ctx.ClientRequestId,
				ErrorCodeValidationError,
				err.Error(),
				snx_lib_api_json.MapFromErr(err),
			)
		}
		grpcOrders = append(grpcOrders, grpcOrder)
	}

	batchReq := &v4grpc.PlaceOrderRequest{
		TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
		SubAccountId:  int64(ctx.SelectedAccountId),
		RequestId:     ctx.RequestId.String(),
		Orders:        grpcOrders,
		Source:        req.Source,
	}

	grpcResp, err := ctx.TradingClient.PlaceBatchOrder(ctx, batchReq)
	if err != nil {
		ctx.Logger.Error("Failed to place batch order", "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to place orders", err)
	}

	statuses := make([]snx_lib_api_json.OrderStatusResponse, 0, len(grpcResp.Orders))

	for _, orderResp := range grpcResp.Orders {
		var status snx_lib_api_json.OrderStatusResponse

		if !orderResp.IsSuccess {
			status = snx_lib_api_json.NewOrderStatusResponse().WithError(orderResp)
		} else {
			switch orderResp.Status {
			case v4grpc.OrderStatus_FILLED, v4grpc.OrderStatus_PARTIALLY_FILLED:
				status = snx_lib_api_json.NewOrderStatusResponse().WithFilled(orderResp)
			case v4grpc.OrderStatus_ACCEPTED:
				status = snx_lib_api_json.NewOrderStatusResponse().WithResting(orderResp)
			default:
				ctx.Logger.Error("UNEXPECTED: undiscrimated value of order status",
					"value", fmt.Sprintf("%[1]v <%[1]T>", orderResp.Status),
				)

				status = snx_lib_api_json.NewOrderStatusResponse().WithError(orderResp)
			}
		}

		statuses = append(statuses, status)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, OrderDataResponse{
		Statuses: statuses,
	})
}

func handleCompoundOrder(ctx TradeContext, req PlaceOrdersActionPayload) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if len(req.Orders) < 2 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"please provide one main order, a take profit and/or stop loss",
			nil,
		)
	}

	var mainOrder *snx_lib_api_json.PlaceOrderRequest
	var tpOrder *snx_lib_api_json.PlaceOrderRequest
	var slOrder *snx_lib_api_json.PlaceOrderRequest

	for i := range req.Orders {
		order := &req.Orders[i]
		switch order.OrderType {
		case API_WKS_triggerTp:
			tpOrder = order
		case API_WKS_triggerSl:
			slOrder = order
		default:
			mainOrder = order
		}
	}

	if mainOrder == nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"please provide one main order",
			nil,
		)
	}

	if tpOrder == nil && slOrder == nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"please provide a take profit and/or stop loss",
			nil,
		)
	}

	grpcOrders := make([]*v4grpc.PlaceOrderRequestItem, 0, len(req.Orders))

	grpcMainOrder, err := convertJSONOrderToGRPC(*mainOrder, mainOrder.Symbol)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			fmt.Sprintf("Invalid main order: %v", err),
			snx_lib_api_json.MapFromErr(err),
		)
	}
	grpcOrders = append(grpcOrders, grpcMainOrder)

	if tpOrder != nil {
		grpcTPOrder, err := convertJSONOrderToGRPC(*tpOrder, tpOrder.Symbol)
		if err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
				ctx.ClientRequestId,
				ErrorCodeValidationError,
				fmt.Sprintf("Invalid take profit order: %v", err),
				snx_lib_api_json.MapFromErr(err),
			)
		}
		grpcOrders = append(grpcOrders, grpcTPOrder)
	}

	if slOrder != nil {
		grpcSLOrder, err := convertJSONOrderToGRPC(*slOrder, slOrder.Symbol)
		if err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
				ctx.ClientRequestId,
				ErrorCodeValidationError,
				fmt.Sprintf("Invalid stop loss order: %v", err),
				snx_lib_api_json.MapFromErr(err),
			)
		}
		grpcOrders = append(grpcOrders, grpcSLOrder)
	}

	compoundReq := &v4grpc.PlaceOrderRequest{
		TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
		SubAccountId:  int64(ctx.SelectedAccountId),
		RequestId:     ctx.RequestId.String(),
		Orders:        grpcOrders,
		Source:        req.Source,
	}

	grpcResp, err := ctx.TradingClient.PlaceCompoundOrder(ctx, compoundReq)
	if err != nil {
		ctx.Logger.Error("Failed to place compound order", "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to place compound order", err)
	}

	statuses := make([]snx_lib_api_json.OrderStatusResponse, 0, len(grpcResp.Orders))

	for _, orderResp := range grpcResp.Orders {

		status := snx_lib_api_json.NewOrderStatusResponse()

		if !orderResp.IsSuccess {
			status = status.WithError(orderResp)
		} else {
			switch orderResp.Status {
			case v4grpc.OrderStatus_FILLED, v4grpc.OrderStatus_PARTIALLY_FILLED:
				status = status.WithFilled(orderResp)
			case v4grpc.OrderStatus_ACCEPTED:
				status = status.WithResting(orderResp)
			default:
				ctx.Logger.Error("UNEXPECTED: undiscrimated value of order status",
					"value", fmt.Sprintf("%[1]v <%[1]T>", orderResp.Status),
				)

				status = status.WithError(orderResp)
			}
		}
		statuses = append(statuses, status)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, OrderDataResponse{
		Statuses: statuses,
	})
}

func handlePositionTPSL(ctx TradeContext, req PlaceOrdersActionPayload) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {

	switch len(req.Orders) {
	case 1, 2:
	default:
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError, "position TP/SL requires 1-2 orders (take profit and/or stop loss)",
			nil,
		)
	}

	symbols := make(map[Symbol]bool, len(req.Orders))
	grpcOrders := make([]*v4grpc.PlaceOrderRequestItem, 0, len(req.Orders))

	for _, order := range req.Orders {
		symbols[order.Symbol] = true

		grpcOrder, err := convertJSONOrderToGRPC(order, order.Symbol)
		if err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
				ctx.ClientRequestId,
				ErrorCodeValidationError,
				err.Error(), // is this appropriate?
				snx_lib_api_json.MapFromErr(err),
			)
		}
		grpcOrders = append(grpcOrders, grpcOrder)
	}

	// Check that all orders are for the same symbol
	if len(symbols) > 1 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"all orders must be for the same symbol",
			nil,
		)
	}

	positionTPSLReq := &v4grpc.PlaceOrderRequest{
		TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
		SubAccountId:  int64(ctx.SelectedAccountId),
		RequestId:     ctx.RequestId.String(),
		Orders:        grpcOrders,
		Source:        req.Source,
	}

	grpcResp, err := ctx.TradingClient.PlacePositionTPAndSl(ctx, positionTPSLReq)
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to place position TP/SL", err)
	}

	statuses := make([]snx_lib_api_json.OrderStatusResponse, 0, len(grpcResp.Orders))

	for _, orderResp := range grpcResp.Orders {
		status := snx_lib_api_json.NewOrderStatusResponse()

		if !orderResp.IsSuccess {
			status = status.WithError(orderResp)
		} else {
			switch orderResp.Status {
			case v4grpc.OrderStatus_ACCEPTED:
				status = status.WithResting(orderResp)
			default:
				ctx.Logger.Error("UNEXPECTED: undiscrimated value of order status",
					"value", fmt.Sprintf("%[1]v <%[1]T>", orderResp.Status),
				)

				status = status.WithError(orderResp)
			}
		}

		statuses = append(statuses, status)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, OrderDataResponse{
		Statuses: statuses,
	})
}

func handleTWAPOrder(ctx TradeContext, req PlaceOrdersActionPayload) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if len(req.Orders) != 1 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"TWAP request must contain exactly one order",
			nil,
		)
	}

	order := req.Orders[0]
	// order.Symbol is set by ValidatePlaceOrdersAction, including when clients send symbol only on the payload.
	grpcOrder, err := convertJSONOrderToGRPC(order, order.Symbol)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			err.Error(),
			snx_lib_api_json.MapFromErr(err),
		)
	}

	twapReq := &v4grpc.PlaceOrderRequest{
		TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
		SubAccountId:  int64(ctx.SelectedAccountId),
		RequestId:     ctx.RequestId.String(),
		Orders:        []*v4grpc.PlaceOrderRequestItem{grpcOrder},
		Source:        req.Source,
	}

	grpcResp, err := ctx.TradingClient.PlaceTWAPOrder(ctx, twapReq)
	if err != nil {
		if st, ok := grpcstatus.FromError(err); ok && st.Code() == codes.InvalidArgument {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, st.Message(), nil)
		}
		ctx.Logger.Error("Failed to place TWAP order", "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to place TWAP order", err)
	}

	statuses := make([]snx_lib_api_json.OrderStatusResponse, 0, len(grpcResp.Orders))

	for _, orderResp := range grpcResp.Orders {
		var status snx_lib_api_json.OrderStatusResponse

		if !orderResp.IsSuccess {
			status = snx_lib_api_json.NewOrderStatusResponse().WithError(orderResp)
		} else {
			switch orderResp.Status {
			case v4grpc.OrderStatus_ACCEPTED:
				status = snx_lib_api_json.NewOrderStatusResponse().WithResting(orderResp)
			default:
				ctx.Logger.Error("UNEXPECTED: undiscrimated value of order status",
					"value", fmt.Sprintf("%[1]v <%[1]T>", orderResp.Status),
				)
				status = snx_lib_api_json.NewOrderStatusResponse().WithError(orderResp)
			}
		}

		statuses = append(statuses, status)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, OrderDataResponse{
		Statuses: statuses,
	})
}
