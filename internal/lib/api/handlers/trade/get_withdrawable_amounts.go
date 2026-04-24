package trade

import (
	"github.com/go-viper/mapstructure/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type GetWithdrawableAmountsRequest struct {
	Symbols []string `json:"symbols"`
}

type GetWithdrawableAmountsResponseItem struct {
	Symbol             string `json:"symbol"`
	WithdrawableAmount string `json:"withdrawableAmount"`
	Quantity           string `json:"quantity"`
	PendingWithdraw    string `json:"pendingWithdraw"`
	WithdrawFee        string `json:"withdrawFee"`
}

type GetWithdrawableAmountsResponse struct {
	Items                 []GetWithdrawableAmountsResponseItem `json:"items"`
	TotalWithdrawableUSDT string                               `json:"totalWithdrawableUsdt"`
}

func Handle_getWithdrawableAmounts(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req GetWithdrawableAmountsRequest
	if err := mapstructure.Decode(params, &req); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil,
		)
	}

	if len(req.Symbols) == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](
			ctx.ClientRequestId, "At least one symbol is required", nil,
		)
	}

	grpcReq := &v4grpc.GetWithdrawableAmountsRequest{
		TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
		SubAccountId:  int64(ctx.SelectedAccountId),
		Symbols:       req.Symbols,
	}

	grpcResp, err := ctx.TradingClient.GetWithdrawableAmounts(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get withdrawable amounts from trading service", "error", err)

		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, st.Message(), nil)
			case codes.NotFound:
				return HTTPStatusCode_404_NotFound, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeNotFound, st.Message(), nil)
			case codes.FailedPrecondition:
				return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, st.Message(), nil)
			case codes.Unavailable:
				return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, st.Message(), err)
			case codes.Internal:
				return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, st.Message(), err)
			}
		}

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Internal server error", err)
	}

	items := make([]GetWithdrawableAmountsResponseItem, 0, len(grpcResp.Items))
	for _, item := range grpcResp.Items {
		items = append(items, GetWithdrawableAmountsResponseItem{
			Symbol:             item.Symbol,
			WithdrawableAmount: item.WithdrawableAmount,
			Quantity:           item.Quantity,
			PendingWithdraw:    item.PendingWithdraw,
			WithdrawFee:        item.WithdrawFee,
		})
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, GetWithdrawableAmountsResponse{
		Items:                 items,
		TotalWithdrawableUSDT: grpcResp.TotalWithdrawableUsdt,
	})
}
