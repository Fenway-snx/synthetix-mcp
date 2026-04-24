package trade

import (
	"errors"

	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

var (
	errMissingScheduleCancelPayload = errors.New("missing validated scheduleCancel payload in context")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/scheduleCancel
*/

type ScheduleCancelResponse struct {
	IsActive       bool       `json:"isActive"`
	Message        string     `json:"message,omitempty"`
	TimeoutSeconds int64      `json:"timeoutSeconds"`
	TriggerTime    *Timestamp `json:"triggerTime,omitempty"`
}

// Handler for "scheduleCancel".
//
//dd:span
func Handle_scheduleCancel(
	ctx TradeContext,
	_params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*ValidatedScheduleCancelAction)
	if !ok || validated == nil {
		ctx.Logger.Error("Missing validated scheduleCancel payload in context")
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingScheduleCancelPayload)
	}

	grpcResp, err := ctx.TradingClient.ScheduleCancel(ctx, &v4grpc.ScheduleCancelRequest{
		SubAccountId:   int64(ctx.SelectedAccountId),
		RequestId:      ctx.RequestId.String(),
		TimeoutSeconds: validated.TimeoutSeconds,
		TmRequestedAt:  timestamppb.New(snx_lib_utils_time.Now()),
	})
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to schedule cancel", err)
	}

	var triggerTime *Timestamp
	if grpcResp.TriggerTime != nil {
		t := Timestamp(grpcResp.TriggerTime.AsTime().UnixMilli())
		triggerTime = &t
	}

	resp := snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, ScheduleCancelResponse{
		IsActive:       grpcResp.IsActive,
		Message:        grpcResp.Message,
		TimeoutSeconds: grpcResp.TimeoutSeconds,
		TriggerTime:    triggerTime,
	})

	return HTTPStatusCode_200_OK, resp
}
