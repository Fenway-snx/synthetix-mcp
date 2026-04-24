package info

import (
	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getFundingRateHistory
*/

// FundingRateHistoryRequest represents the request parameters for funding rate history
type FundingRateHistoryRequest struct {
	Symbol    Symbol    `json:"symbol"`    // Required: Trading pair symbol (e.g., "BTC-USDT")
	StartTime Timestamp `json:"startTime"` // Required: Start time in milliseconds
	EndTime   Timestamp `json:"endTime"`   // Required: End time in milliseconds
	Limit     int32     `json:"limit"`     // Required: Max results to return (0 = default max of 2160)
}

// FundingRateHistoryResponse represents the response for funding rate history
type FundingRateHistoryResponse struct {
	Symbol       Symbol                     `json:"symbol"`       // Trading pair symbol
	FundingRates []FundingRateHistoryRecord `json:"fundingRates"` // Historical funding rates (newest first)
}

// FundingRateHistoryRecord represents a single historical funding rate record
type FundingRateHistoryRecord struct {
	FundingRate string    `json:"fundingRate"` // The published funding rate
	FundingTime Timestamp `json:"fundingTime"` // When the rate was published (ms since epoch)
	AppliedAt   Timestamp `json:"appliedAt"`   // When the rate was applied to positions (ms since epoch)
}

// Handler for "getFundingRateHistory".
//
//dd:span
func Handle_getFundingRateHistory(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req FundingRateHistoryRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	normalizedSymbol, err := snx_lib_api_validation.ValidateAndNormalizeSymbol(req.Symbol)
	if err != nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, err.Error(), nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	req.Symbol = normalizedSymbol
	if req.StartTime == 0 {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "startTime is required", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	if req.EndTime == 0 {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "endTime is required", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	if req.StartTime >= req.EndTime {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "startTime must be before endTime", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Convert timestamps to protobuf format
	startTimePb, err := snx_lib_api_types.TimestampToTimestampPB(req.StartTime)
	if err != nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "invalid startTime value", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}
	endTimePb, err := snx_lib_api_types.TimestampToTimestampPB(req.EndTime)
	if err != nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "invalid endTime value", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call subaccount service to get funding rate history
	grpcReq := &v4grpc.GetFundingRateHistoryRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Symbol:      string(req.Symbol),
		StartTime:   startTimePb,
		EndTime:     endTimePb,
		Limit:       req.Limit,
	}

	grpcResp, err := ctx.SubaccountClient.GetFundingRateHistory(ctx, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get funding rate history from subaccount service",
			"error", err,
			"symbol", req.Symbol,
		)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to get funding rate history", err)
	}

	symbol := Symbol(grpcResp.Symbol)

	// Convert to API response format
	fundingRates := make([]FundingRateHistoryRecord, len(grpcResp.FundingRates))
	for i, fr := range grpcResp.FundingRates {
		fundingTime := snx_lib_api_types.TimestampFromTimestampPBOrZero(fr.FundingTime)
		appliedAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(fr.AppliedAt)

		fundingRates[i] = FundingRateHistoryRecord{
			FundingRate: fr.FundingRate,
			FundingTime: fundingTime,
			AppliedAt:   appliedAt,
		}
	}

	ctx.Logger.Info("Retrieved funding rate history",
		"symbol", symbol,
		"count", len(fundingRates),
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, FundingRateHistoryResponse{
		Symbol:       symbol,
		FundingRates: fundingRates,
	})
}
