package trade

import (
	"errors"
	"fmt"
	"time"

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
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getFundingPayments
*/

const (
	defaultFundingPaymentsLimit = 100
	maxFundingPaymentsLimit     = 1_000
)

var (
	errLimitInvalid     = errors.New("limit must be a positive integer")
	errLimitTooLarge    = fmt.Errorf("limit must not exceed %d", maxFundingPaymentsLimit)
	errStartTimeInvalid = errors.New("startTime must be a valid timestamp")
	errTimeRangeInvalid = errors.New("startTime must be less than endTime")
)

// GetFundingPaymentsRequest represents the request parameters for funding payments
type GetFundingPaymentsRequest struct {
	Symbol    Symbol    `json:"symbol,omitempty"`    // Filter by specific trading pair
	StartTime Timestamp `json:"startTime,omitempty"` // Start timestamp for filtering (ms)
	EndTime   Timestamp `json:"endTime,omitempty"`   // End timestamp for filtering (ms)
	Limit     int       `json:"limit,omitempty"`     // Maximum number of records (default: 100)
}

// GetFundingPaymentsResponse represents the response for funding payments
type GetFundingPaymentsResponse struct {
	Summary        FundingSummary   `json:"summary"`
	FundingHistory []FundingPayment `json:"fundingHistory"`
}

// FundingSummary represents the summary of funding payments
type FundingSummary struct {
	TotalFundingReceived string `json:"totalFundingReceived"` // Total funding payments received
	TotalFundingPaid     string `json:"totalFundingPaid"`     // Total funding payments paid
	NetFunding           string `json:"netFunding"`           // Net funding (received - paid)
	TotalPayments        string `json:"totalPayments"`        // Total number of funding payments
	AveragePaymentSize   string `json:"averagePaymentSize"`   // Average payment size (absolute value)
}

// Represents a single funding payment.
type FundingPayment struct {
	PaymentId                                   string    `json:"paymentId"`        // Unique identifier for the funding payment
	Symbol                                      Symbol    `json:"symbol"`           // Trading pair symbol
	PositionSize                                string    `json:"positionSize"`     // Position size at funding time (signed)
	FundingRate                                 string    `json:"fundingRate"`      // Funding rate applied (1-hour rate)
	Payment                                     string    `json:"payment"`          // Funding payment amount (negative = paid out)
	DEPRECATED_Timestamp_NOW_PaymentTime        Timestamp `json:"timestamp"`        // [DEPRECATED] // TODO: SNX-4911
	PaymentTime                                 Timestamp `json:"paymentTime"`      // When payment was processed, e.g. 1735689600000
	DEPRECATED_FundingTimestamp_NOW_FundingTime Timestamp `json:"fundingTimestamp"` // [DEPRECATED] // TODO: SNX-4911
	FundingTime                                 Timestamp `json:"fundingTime"`      // Funding period timestamp, e.g. 1735689600000
}

// Converts gRPC response to API response format.
func convertGrpcToApiResponse(grpcResp *v4grpc.GetFundingPaymentsResponse) (
	GetFundingPaymentsResponse, // result
	error, // err
	string, // failureQualifier
) {
	// Convert funding history
	fundingHistory := make([]FundingPayment, len(grpcResp.FundingHistory))
	for i, fp := range grpcResp.FundingHistory {

		paymentTime, err := snx_lib_api_types.TimestampFromTimestampPB(fp.PaymentTime)
		if err != nil {

			return GetFundingPaymentsResponse{}, err, "invalid timestamp"
		}

		fundingTime, err := snx_lib_api_types.TimestampFromTimestampPB(fp.FundingTime)
		if err != nil {

			return GetFundingPaymentsResponse{}, err, "invalid funding-timestamp"
		}

		fundingHistory[i] = FundingPayment{
			PaymentId:                            fp.PaymentId,
			Symbol:                               Symbol(fp.Symbol),
			PositionSize:                         fp.PositionSize,
			FundingRate:                          fp.FundingRate,
			Payment:                              fp.Payment,
			DEPRECATED_Timestamp_NOW_PaymentTime: paymentTime, // [DEPRECATED] // TODO: SNX-4911
			PaymentTime:                          paymentTime,
			DEPRECATED_FundingTimestamp_NOW_FundingTime: fundingTime, // [DEPRECATED] // TODO: SNX-4911
			FundingTime: fundingTime,
		}
	}

	// Convert summary
	summary := FundingSummary{
		TotalFundingReceived: grpcResp.Summary.TotalFundingReceived,
		TotalFundingPaid:     grpcResp.Summary.TotalFundingPaid,
		NetFunding:           grpcResp.Summary.NetFunding,
		TotalPayments:        grpcResp.Summary.TotalPayments,
		AveragePaymentSize:   grpcResp.Summary.AveragePaymentSize,
	}

	return GetFundingPaymentsResponse{
		Summary:        summary,
		FundingHistory: fundingHistory,
	}, nil, ""
}

// Handler for "getFundingPayments".
//
//dd:span
func Handle_getFundingPayments(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Validate that a subAccountId was provided and authenticated
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subAccountId is required", nil)
	}

	var req GetFundingPaymentsRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}

	// Validate request parameters
	if err := validateGetFundingPaymentsRequest(&req); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, err.Error(), nil)
	}

	// Use the authenticated subaccount ID from context
	subAccountId := ctx.SelectedAccountId

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Create gRPC request
	grpcReq := &v4grpc.GetFundingPaymentsRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(subAccountId),
	}

	// Apply default limit when caller omits it; validation has already capped to max.
	if req.Limit == 0 {
		req.Limit = defaultFundingPaymentsLimit
	}
	limit := int32(req.Limit)
	grpcReq.Limit = &limit

	// Add symbol filter if provided
	if req.Symbol != "" {
		sym := string(req.Symbol)
		grpcReq.Symbol = &sym
	}

	// Add time filters if provided (convert milliseconds to timestamp)

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

	// Call subaccount service
	grpcResp, err := ctx.SubaccountClient.GetFundingPayments(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to get funding payments from subaccount service", "error", err)
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to retrieve funding payments", err)
	}

	// Convert gRPC response to API response format
	fundingPayments, err, qualifier := convertGrpcToApiResponse(grpcResp)
	if err != nil {

		ctx.Logger.Error("Failed to get funding payments from subaccount service",
			"error", err,
			"qualifier", qualifier,
		)

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to retrieve funding payments", err)
	}

	ctx.Logger.Info("GetFundingPayments request completed",
		"sub_account_id", subAccountId,
		"total_payments", len(fundingPayments.FundingHistory),
		"total_funding_received", fundingPayments.Summary.TotalFundingReceived,
		"total_funding_paid", fundingPayments.Summary.TotalFundingPaid,
		"net_funding", fundingPayments.Summary.NetFunding,
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, fundingPayments)
}

// validateGetFundingPaymentsRequest validates the funding payments request
func validateGetFundingPaymentsRequest(req *GetFundingPaymentsRequest) error {
	if req.Symbol != "" {
		normalizedSymbol, err := snx_lib_api_validation.ValidateAndNormalizeSymbol(req.Symbol)
		if err != nil {
			return err
		}
		req.Symbol = normalizedSymbol
	}

	// Validate time range if both are provided
	if req.StartTime > 0 && req.EndTime > 0 {
		if req.StartTime >= req.EndTime {
			return errTimeRangeInvalid
		}

		// Check if timestamps are reasonable (not too far in the past/future)
		now := snx_lib_api_types.TimestampNow()
		if req.StartTime > now || req.EndTime > now {
			return errStartTimeInvalid
		}

		// Check if start time is not too far in the past (e.g., more than 1 year)
		oneYearAgo, err := now.SubDuration(time.Hour * 24 * 365)
		if err != nil || req.StartTime < oneYearAgo {
			return errStartTimeInvalid
		}
	}

	// Validate limit
	if req.Limit < 0 {
		return errLimitInvalid
	}
	if req.Limit > maxFundingPaymentsLimit {
		return errLimitTooLarge
	}

	return nil
}
