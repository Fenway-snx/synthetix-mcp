package trade

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:

	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/voluntaryAutoExchange
*/

var (
	errMissingVoluntaryAutoExchangePayload = errors.New("missing validated voluntaryAutoExchange payload in context")
)

type VoluntaryAutoExchangeResponse struct {
	SourceAsset       string                  `json:"sourceAsset"`
	SourceAmountTaken string                  `json:"sourceAmountTaken"`
	TargetAsset       string                  `json:"targetAsset"`
	TargetAmount      string                  `json:"targetAmount"`
	IndexPrice        Price                   `json:"indexPrice"`
	EffectiveHaircut  string                  `json:"effectiveHaircut"`
	Collateral        []CollateralBalanceItem `json:"collateral"`
}

type CollateralBalanceItem struct {
	Symbol   Asset    `json:"symbol"` // TODO: SNX-6098: rename to `AssetName`
	Quantity Quantity `json:"quantity"`
}

// Handler for "voluntaryAutoExchange".
//
//dd:span
func Handle_voluntaryAutoExchange(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*ValidatedVoluntaryAutoExchangeAction)
	if !ok {
		ctx.Logger.Error("missing validated voluntaryAutoExchange payload in context")
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingVoluntaryAutoExchangePayload)
	}

	tm_now := snx_lib_utils_time.Now()

	grpcReq := &v4grpc.VoluntaryAutoExchangeRequest{
		TmRequestedAt:    timestamppb.New(tm_now),
		SubAccountId:     int64(ctx.SelectedAccountId),
		TargetUsdtAmount: validated.TargetUSDTAmount,
		SourceAsset:      validated.SourceAsset,
	}

	grpcResp, err := ctx.TradingClient.VoluntaryAutoExchange(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("failed to process voluntary auto-exchange", "error", err)
		return handleVoluntaryAutoExchangeGRPCError(err, ctx.ClientRequestId)
	}

	collateral := make([]CollateralBalanceItem, 0, len(grpcResp.Collateral))
	for _, col := range grpcResp.Collateral {
		collateral = append(collateral, CollateralBalanceItem{
			Symbol:   snx_lib_api_types.AssetNameFromStringUnvalidated(col.Symbol),
			Quantity: snx_lib_api_types.QuantityFromStringUnvalidated(col.Quantity),
		})
	}

	response := VoluntaryAutoExchangeResponse{
		SourceAsset:       grpcResp.SourceAsset,
		SourceAmountTaken: grpcResp.SourceAmountTaken,
		TargetAsset:       grpcResp.TargetAsset,
		TargetAmount:      grpcResp.TargetAmount,
		IndexPrice:        snx_lib_api_types.PriceFromStringUnvalidated(grpcResp.IndexPrice),
		EffectiveHaircut:  grpcResp.EffectiveHaircut,
		Collateral:        collateral,
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, response)
}

func handleVoluntaryAutoExchangeGRPCError(err error, clientRequestId ClientRequestId) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	st, ok := status.FromError(err)
	if !ok {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](clientRequestId, "Internal error", err)
	}

	switch st.Code() {
	case codes.NotFound:
		return HTTPStatusCode_404_NotFound, snx_lib_api_json.NewErrorResponse[any](clientRequestId, snx_lib_status_codes.ErrorCodeNotFound, st.Message(), nil)
	case codes.InvalidArgument:
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](clientRequestId, snx_lib_status_codes.ErrorCodeValidationError, st.Message(), nil)
	case codes.FailedPrecondition:
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](clientRequestId, snx_lib_status_codes.ErrorCodeValidationError, st.Message(), nil)
	default:
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](clientRequestId, st.Message(), err)
	}
}
