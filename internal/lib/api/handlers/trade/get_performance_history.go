package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getPerformanceHistory
*/

type PerformanceHistoryPeriod struct {
	History []PerformanceHistoryPoint `json:"history"`
	Volume  string                    `json:"volume"`
}

type PerformanceHistoryPoint struct {
	SampledAt    int64  `json:"sampledAt"`
	AccountValue string `json:"accountValue"`
	Pnl          string `json:"pnl"`
}

type PerformanceHistoryResponse struct {
	SubAccountId SubAccountId             `json:"subAccountId"`
	Period       string                   `json:"period"`
	Performance  PerformanceHistoryPeriod `json:"performanceHistory"`
}

// Handler for "getPerformanceHistory".
//
//dd:span
func Handle_getPerformanceHistory(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subaccountId is required", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Default period is "day" unless provided
	period := "day"
	if v, ok := params["period"]; ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			period = s
		}
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()

	grpcReq := &v4grpc.GetPerformanceHistoryRequest{
		TimestampMs:  timestampMs,
		TimestampUs:  timestampUs,
		SubAccountId: int64(ctx.SelectedAccountId),
		Period:       period,
	}

	grpcResp, err := ctx.SubaccountClient.GetPerformanceHistory(ctx.Context, grpcReq)
	if err != nil {
		failMessage := "Failed to get performance history"

		ctx.Logger.Error(failMessage, "error", err, "sub_account_id", ctx.SelectedAccountId)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, failMessage, err)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	mapPoints := func(points []*v4grpc.PerformanceHistoryPoint) []PerformanceHistoryPoint {
		mapped := make([]PerformanceHistoryPoint, 0, len(points))
		for _, p := range points {
			mapped = append(mapped, PerformanceHistoryPoint{
				SampledAt:    p.SampledAt,
				AccountValue: p.AccountValue,
				Pnl:          p.Pnl,
			})
		}
		return mapped
	}

	responsePayload := PerformanceHistoryResponse{
		SubAccountId: snx_lib_api_types.SubAccountIdFromIntUnvalidated(int64(ctx.SelectedAccountId)),
		Period:       grpcResp.Period,
		Performance: func(period *v4grpc.PerformanceHistoryPeriod) PerformanceHistoryPeriod {
			if period == nil {
				return PerformanceHistoryPeriod{
					History: []PerformanceHistoryPoint{},
				}
			}
			return PerformanceHistoryPeriod{
				History: mapPoints(period.History),
				Volume:  period.Volume,
			}
		}(grpcResp.Performance),
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, responsePayload)
}
