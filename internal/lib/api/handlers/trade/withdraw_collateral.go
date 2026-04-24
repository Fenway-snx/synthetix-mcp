package trade

import (
	shopspring_decimal "github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_constants "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/constants"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/withdrawCollateral
*/

// WithdrawCollateralRequest represents the request to withdraw collateral
type WithdrawCollateralRequest struct {
	Symbol      Symbol        `json:"symbol" validate:"required"`
	Amount      string        `json:"amount" validate:"required"`
	Destination WalletAddress `json:"destination" validate:"required"`
}

// WithdrawCollateralResponse represents the response after initiating a withdrawal
type WithdrawCollateralResponse struct {
	RequestId                             string        `json:"requestId"`
	Symbol                                Symbol        `json:"symbol"`
	Amount                                string        `json:"amount"`
	Destination                           WalletAddress `json:"destination"`
	DEPRECATED_Timestamp_NOW_WithdrawTime Timestamp     `json:"timestamp"` // [DEPRECATED] TODO: SNX-4911
	WithdrawTime                          Timestamp     `json:"withdrawTime"`
}

// Handler for "withdrawCollateral".
//
//dd:span
func Handle_withdrawCollateral(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*ValidatedWithdrawCollateralAction)
	if !ok || validated == nil || validated.Payload == nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	req := WithdrawCollateralRequest{
		Amount:      validated.Payload.Amount,
		Destination: validated.Payload.Destination,
		Symbol:      Symbol(validated.Payload.Symbol),
	}

	amountDecimal, err := shopspring_decimal.NewFromString(req.Amount)
	if err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid amount format", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Validate amount is positive
	if !amountDecimal.IsPositive() {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Amount must be greater than 0", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	// Call trading service via gRPC with sanitized inputs
	grpcReq := &v4grpc.WithdrawCollateralRequest{
		TmRequestedAt: timestamppb.New(snx_lib_utils_time.Now()),
		SubAccountId:  int64(ctx.SelectedAccountId),
		Symbol:        string(req.Symbol),
		Amount:        amountDecimal.String(),
		Destination:   snx_lib_api_types.WalletAddressToString(req.Destination),
		RequestId:     ctx.RequestId.String(),
		WalletAddress: string(ctx.WalletAddress),
	}

	grpcResp, err := ctx.TradingClient.WithdrawCollateral(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("Failed to request withdrawal from trading service", "error", err)

		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, st.Message(), nil)
			case codes.NotFound:
				return HTTPStatusCode_404_NotFound, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeNotFound, st.Message(), nil)
			case codes.PermissionDenied:
				return HTTPStatusCode_403_Forbidden, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, st.Message(), nil)
			case codes.Internal:
				return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, st.Message(), err)
			}
		}

		// Default for unknown errors
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Internal server error", err)
	}

	ctx.Logger.Info("Withdrawal request processed",
		snx_lib_api_constants.API_WKS_amount, amountDecimal.String(), // Use decimal string for exact precision
		snx_lib_api_constants.API_WKS_destination, req.Destination,
		snx_lib_api_constants.API_WKS_requestID, ctx.RequestId, // Internal server-generated request ID
		snx_lib_api_constants.API_WKS_symbol, req.Symbol,
		snx_lib_api_constants.API_WKS_user, ctx.SelectedAccountId,
	)

	/*
		// Create success response
		timestamp, err := snx_lib_api_types.TimestampFromTimestampPB(grpcResp.Timestamp)
		if err != nil {

			// TODO: combine log+HTTP_response into utility function to reduce repetiton
			ctx.Logger.Error("invalid timestamp on record",
				"error", err,
			)

			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewErrorResponse[any](
				ctx.ClientRequestId,
				snx_lib_api_json.ErrorCodeInternalError,
				"invalid timestamp on record",
				map[string]string{"error": err.Error()},
			)
		}
	*/

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, WithdrawCollateralResponse{
		RequestId:   snx_lib_api_types.ClientRequestIdToStringUnvalidated(ctx.ClientRequestId), // Use API request ID for end-to-end correlation
		Symbol:      Symbol(grpcResp.Symbol),
		Amount:      grpcResp.Amount,
		Destination: snx_lib_api_types.WalletAddressFromStringUnvalidated(grpcResp.Destination),
	})
}
