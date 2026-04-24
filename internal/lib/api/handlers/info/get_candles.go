package info

import (
	"errors"
	"fmt"

	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getCandles
*/

var (
	errIntervalRequired       = errors.New("interval is required")
	errLimitMustBeNonNegative = errors.New("limit must be non-negative")
	errStartTimeMustBeLess    = errors.New("startTime must be less than endTime")
)

// CandleRequest represents the request parameters for candle data
type CandleRequest struct {
	Symbol    Symbol    `json:"symbol" validate:"required"`   // Trading pair symbol (e.g., "BTC-USD")
	Interval  string    `json:"interval" validate:"required"` // Time interval (1m, 5m, 15m, 30m, 1h, 4h, 8h, 12h, 1d, 3d, 1w, 1M, 3M)
	Limit     int       `json:"limit,omitempty"`              // Number of candles to return (0 = no limit)
	StartTime Timestamp `json:"startTime,omitempty"`          // Start time in milliseconds
	EndTime   Timestamp `json:"endTime,omitempty"`            // End time in milliseconds
}

// CandleResponse represents the response for candle data
type CandleResponse struct {
	Symbol   Symbol   `json:"symbol"`
	Interval string   `json:"interval"`
	Candles  []Candle `json:"candles"`
}

// Candle represents a single candlestick data point
type Candle struct {
	OpenTime    Timestamp `json:"openTime"`    // Candle open time in milliseconds
	CloseTime   Timestamp `json:"closeTime"`   // Candle close time in milliseconds
	OpenPrice   Price     `json:"openPrice"`   // Opening price
	HighPrice   Price     `json:"highPrice"`   // Highest price
	LowPrice    Price     `json:"lowPrice"`    // Lowest price
	ClosePrice  Price     `json:"closePrice"`  // Closing price
	Volume      string    `json:"volume"`      // Base asset volume
	QuoteVolume string    `json:"quoteVolume"` // Quote asset volume
	TradeCount  int32     `json:"tradeCount"`  // Number of trades
}

// validateCandleRequest validates the candle request parameters
func validateCandleRequest(req *CandleRequest) error {
	symbol, err := snx_lib_api_validation.ValidateAndNormalizeSymbol(req.Symbol)
	if err != nil {
		return err
	}
	req.Symbol = symbol

	if req.Interval == "" {
		return errIntervalRequired
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.Interval, snx_lib_api_validation.MaxEnumFieldLength, "interval"); err != nil {
		return err
	}

	if _, err := snx_lib_utils_time.ParseTimeframe(req.Interval); err != nil {
		return fmt.Errorf("invalid interval: %s. Valid intervals: %v", req.Interval, snx_lib_utils_time.SupportedTimeframes)
	}

	if req.Limit < 0 {
		return errLimitMustBeNonNegative
	}

	if req.StartTime > 0 && req.EndTime > 0 && req.StartTime >= req.EndTime {
		return errStartTimeMustBeLess
	}

	return nil
}

// Handler for "getCandles".
//
//dd:span
func Handle_getCandles(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req CandleRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Validate the request
	if err := validateCandleRequest(&req); err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid request parameters", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Call marketdata service via gRPC
	grpcReq := &v4grpc.GetCandlesticksRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Symbol:      string(req.Symbol),
		Timeframe:   req.Interval,
		Limit:       int32(req.Limit),
	}

	// Only set timestamps if they have meaningful values (non-zero)
	// This allows the gRPC service to apply its own defaults when fields are nil

	now := snx_lib_api_types.TimestampNow()

	if startTime, endTime, err, failureQualifier := snx_lib_api_handlers_utils.APIStartEndToCoreStartEndPtrs(req.StartTime, req.EndTime, now); err != nil {

		resp := snx_lib_api_json.NewValidationErrorResponse[any](
			ctx.ClientRequestId,
			"Invalid request parameters",
			map[string]string{
				"error":     err.Error(),
				"qualifier": failureQualifier,
			},
		)
		return HTTPStatusCode_400_BadRequest, resp
	} else {

		grpcReq.StartTime = startTime
		grpcReq.EndTime = endTime
	}

	grpcResp, err := ctx.MarketDataClient.GetCandlesticks(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get candlesticks from marketdata service", "error", err, API_WKS_symbol, req.Symbol)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to retrieve candle data", err)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	// Convert gRPC response to API response format
	var candles []Candle
	for _, grpcCandle := range grpcResp.Candlesticks {
		openTime := snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcCandle.OpenTime)
		closeTime := snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcCandle.CloseTime)

		candle := Candle{
			OpenTime:    openTime,
			CloseTime:   closeTime,
			OpenPrice:   snx_lib_api_types.PriceFromStringUnvalidated(grpcCandle.OpenPrice),
			HighPrice:   snx_lib_api_types.PriceFromStringUnvalidated(grpcCandle.HighPrice),
			LowPrice:    snx_lib_api_types.PriceFromStringUnvalidated(grpcCandle.LowPrice),
			ClosePrice:  snx_lib_api_types.PriceFromStringUnvalidated(grpcCandle.ClosePrice),
			Volume:      grpcCandle.Volume,
			QuoteVolume: grpcCandle.QuoteVolume,
			TradeCount:  grpcCandle.TradeCount,
		}
		candles = append(candles, candle)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, CandleResponse{
		Symbol:   req.Symbol,
		Interval: req.Interval,
		Candles:  candles,
	})
}
