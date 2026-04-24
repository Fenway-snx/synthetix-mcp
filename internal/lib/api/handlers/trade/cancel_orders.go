package trade

import (
	"errors"

	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

var (
	errMissingCancelOrdersPayload = errors.New("missing validated cancelOrders payload in context")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/cancelOrders
*/

type CancelOrderRequest struct {
	VenueOrderIds []VenueOrderId `json:"orderIds"`
}

// Endpoint response item type.
type CancelOrdersResponseItem = OrderStatusResponse

// Handler for "cancelOrders".
//
//dd:span
func Handle_cancelOrders(
	ctx TradeContext,
	_params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var grpcResp *v4grpc.CancelOrderResponse
	var err error

	switch validated := ctx.ActionPayload().(type) {
	case *ValidatedCancelOrdersAction:
		if validated == nil {
			ctx.Logger.Error("Missing validated cancelOrders payload in context")
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingCancelOrdersPayload)
		}
		grpcResp, err = ctx.TradingClient.CancelOrder(ctx, &v4grpc.CancelOrderRequest{
			TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
			SubAccountId:  int64(ctx.SelectedAccountId),
			VenueOrderIds: snx_lib_api_types.VenueOrderIdArrayToUintArrayUnvalidated(validated.VenueOrderIds),
			RequestId:     ctx.RequestId.String(),
		})
	case *ValidatedCancelOrdersByCloidAction:
		if validated == nil {
			ctx.Logger.Error("Missing validated cancelOrders payload in context")
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingCancelOrdersPayload)
		}
		clientOrderIds := make([]string, len(validated.ClientOrderIds))
		for i, clientOrderId := range validated.ClientOrderIds {
			clientOrderIds[i] = snx_lib_api_types.ClientOrderIdToStringUnvalidated(clientOrderId)
		}
		grpcResp, err = ctx.TradingClient.CancelOrderByCloid(ctx, &v4grpc.CancelOrderByCloidRequest{
			TmRequestedAt:  timestamppb.New(snx_lib_utils_time.Now()),
			SubAccountId:   int64(ctx.SelectedAccountId),
			ClientOrderIds: clientOrderIds,
			RequestId:      ctx.RequestId.String(),
		})
	default:
		ctx.Logger.Error("Missing validated cancelOrders payload in context")
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingCancelOrdersPayload)
	}

	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to cancel orders", err)
	}

	statuses := make([]CancelOrdersResponseItem, 0, len(grpcResp.Orders))
	for _, orderRes := range grpcResp.Orders {
		status := snx_lib_api_json.NewOrderStatusResponse()

		if orderRes.ErrorMessage == "" {
			statuses = append(statuses, status.WithCanceled(orderRes))
		} else {
			statuses = append(statuses, status.WithCancelError(orderRes))
		}
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, OrderDataResponse{
		Statuses: statuses,
	})
}
