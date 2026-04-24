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
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getFundingRate
*/

type FundingRateRequest struct {
	Symbol Symbol `json:"symbol"`
}

type FundingRateResponse struct {
	Symbol               Symbol    `json:"symbol"`               // e.g. "BTC-USDT"
	EstimatedFundingRate string    `json:"estimatedFundingRate"` // Estimated hourly funding rate (rolling 1-hour TWAP) - what users will likely be charged at next settlement
	LastSettlementRate   string    `json:"lastSettlementRate"`   // The actual hourly rate from the last settlement (what users were charged)
	LastSettlementTime   Timestamp `json:"lastSettlementTime"`   // When the last settlement occurred (milliseconds since epoch)
	NextFundingTime      Timestamp `json:"nextFundingTime"`      // When the next settlement will occur (milliseconds since epoch)
	FundingInterval      int64     `json:"fundingInterval"`      // Funding interval in milliseconds (e.g. 3600000 for 1 hour)
}

// Handler for "getFundingRate".
//
//dd:span
func Handle_getFundingRate(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req FundingRateRequest
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

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call subaccount service to get latest funding rate
	grpcReq := &v4grpc.GetLatestFundingRatesRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Symbols:     []string{string(req.Symbol)},
	}

	grpcResp, err := ctx.SubaccountClient.GetLatestFundingRates(ctx, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get latest funding rate from subaccount service",
			"error", err,
			"symbol", req.Symbol,
		)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to get funding rate", err)
	}

	if grpcResp == nil || len(grpcResp.FundingRates) == 0 {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeNotFound, "Funding rate not found", nil)
		return HTTPStatusCode_404_NotFound, resp
	}

	fr := grpcResp.FundingRates[0]

	symbol := Symbol(fr.Symbol)

	// Convert next funding time
	nextFundingTime, err := snx_lib_api_types.TimestampFromTimestampPB(fr.NextFundingTime)
	if err != nil {
		ctx.Logger.Error("Invalid next-funding-time in funding rate from subaccount service",
			"error", err,
			"next-funding-time", fr.NextFundingTime,
			"symbol", req.Symbol,
		)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to get funding rate", err)
	}

	// Convert last settlement time (may be nil if no settlements yet)
	var lastSettlementTime Timestamp
	if fr.LastSettlementTime != nil {
		lastSettlementTime, err = snx_lib_api_types.TimestampFromTimestampPB(fr.LastSettlementTime)
		if err != nil {
			ctx.Logger.Error("Invalid last-settlement-time in funding rate from subaccount service",
				"error", err,
				"last-settlement-time", fr.LastSettlementTime,
				"symbol", req.Symbol,
			)
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to get funding rate", err)
		}
	}

	ctx.Logger.Info("Retrieved funding rate data",
		"estimated_funding_rate", fr.EstimatedFundingRate,
		"last_settlement_rate", fr.LastSettlementRate,
		"symbol", symbol,
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, FundingRateResponse{
		Symbol:               symbol,
		EstimatedFundingRate: fr.EstimatedFundingRate,
		LastSettlementRate:   fr.LastSettlementRate,
		LastSettlementTime:   lastSettlementTime,
		NextFundingTime:      nextFundingTime,
		FundingInterval:      fr.FundingIntervalMs,
	})
}
