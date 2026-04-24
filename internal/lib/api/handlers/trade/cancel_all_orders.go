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
	errMissingCancelAllOrdersPayload = errors.New("missing validated cancelAllOrders payload in context")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/cancelAllOrders
*/

// Endpoint response item type.
type CancelAllOrdersResponseItem struct {
	OrderId                 OrderId      `json:"order"`   // order (paired)
	DEPRECATED_VenueOrderId VenueOrderId `json:"orderId"` // [DEPRECATED] // TODO: SNX-4911
	Message                 string       `json:"message"`
	Symbol                  *Symbol      `json:"symbol"`
}

// Handler for "cancelAllOrders".
//
//dd:span
func Handle_cancelAllOrders(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*ValidatedCancelAllOrdersAction)
	if !ok || validated == nil {
		ctx.Logger.Error("Missing validated cancelAllOrders payload in context")
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingCancelAllOrdersPayload)
	}

	cancelReq := &v4grpc.CancelAllOrdersRequest{
		SubAccountId:  int64(ctx.SelectedAccountId),
		Symbols:       snx_lib_api_types.SymbolsToStringsUnfiltered(validated.Symbols),
		RequestId:     ctx.RequestId.String(),
		TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
	}

	grpcResp, err := ctx.TradingClient.CancelAllOrders(ctx, cancelReq)
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to cancel all orders", err)
	}

	statuses := make([]CancelAllOrdersResponseItem, 0, len(grpcResp.Orders))
	for _, orderRes := range grpcResp.Orders {
		orderId := snx_lib_api_types.OrderIdFromGRPCOrderIdUnvalidated(orderRes.OrderId)

		var symbol *Symbol
		if orderRes.Symbol != "" {
			s := Symbol(orderRes.Symbol)
			symbol = &s
		}

		statuses = append(statuses, CancelAllOrdersResponseItem{
			OrderId:                 orderId,
			DEPRECATED_VenueOrderId: orderId.VenueId,
			Message:                 orderRes.ErrorMessage,
			Symbol:                  symbol,
		})
	}

	// Create success response
	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, statuses)
}
