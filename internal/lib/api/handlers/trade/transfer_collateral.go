package trade

import (
	"errors"
	"strconv"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/transfer"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

var (
	errMissingTransferCollateralPayload = errors.New("missing validated transferCollateral payload in context")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/transferCollateral
*/

type TransferCollateralResponse struct {
	Amount        string           `json:"amount"`
	From          SubAccountStatus `json:"from"`
	RequestId     string           `json:"requestId"`
	Status        string           `json:"status"`
	Symbol        Asset            `json:"symbol"` // TODO: SNX-6098: rename to `AssetName`
	To            SubAccountStatus `json:"to"`
	TransferId    string           `json:"transferId"`
	TransferredAt Timestamp        `json:"transferredAt"`
}

type SubAccountStatus struct {
	Amount       string `json:"amount"`
	SubAccountId string `json:"subAccountId"`
}

//dd:span
func Handle_transferCollateral(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Extract validated payload from context
	validated, ok := ctx.ActionPayload().(*ValidatedTransferCollateralAction)
	if !ok {
		ctx.Logger.Error("missing validated transferCollateral payload in context")

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Invalid request context", errMissingTransferCollateralPayload)
	}

	from := ctx.SelectedAccountId

	if from == validated.To {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_status_codes.ErrorCodeValidationError, "invalid request", nil)

		return HTTPStatusCode_400_BadRequest, resp
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	amountStr := validated.Amount.String()

	grpcReq := &v4grpc.TransferBetweenSubAccountsRequest{
		TimestampMs:      timestamp_ms,
		TimestampUs:      timestamp_us,
		FromSubAccountId: int64(from),
		ToSubAccountId:   int64(validated.To),
		Symbol:           string(validated.Symbol),
		Amount:           amountStr,
		RequestId:        ctx.RequestId.String(),
		WalletAddress:    string(ctx.WalletAddress),
	}

	grpcResp, err := ctx.SubaccountClient.TransferBetweenSubAccounts(ctx.Context, grpcReq)
	if err != nil {
		ctx.Logger.Error("failed to process transfer", "error", err)

		return handleGRPCError(err, ctx.ClientRequestId)
	}

	if grpcResp.Status == transfer.Status_Failure.String() {
		ctx.Logger.Warn("transfer failed",
			"error_message", grpcResp.ErrorMessage,
			"from", grpcReq.FromSubAccountId,
			"to", grpcReq.ToSubAccountId,
		)

		errorCode := snx_lib_status_codes.ErrorCode(grpcResp.ErrorCode)
		httpStatus := httpStatusFromTransferError(errorCode)

		return httpStatus, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			errorCode,
			grpcResp.ErrorMessage,
			nil,
		)
	}

	response := TransferCollateralResponse{
		RequestId:  snx_lib_api_types.ClientRequestIdToStringUnvalidated(ctx.ClientRequestId),
		TransferId: strconv.FormatInt(grpcResp.TransferId, 10),
		Status:     grpcResp.Status,
		Symbol:     validated.Symbol,
		Amount:     amountStr,
		From: SubAccountStatus{
			SubAccountId: snx_lib_api_types.SubAccountIdFromIntRaw(int64(from)),
			Amount:       grpcResp.FromBalance,
		},
		To: SubAccountStatus{
			SubAccountId: snx_lib_api_types.SubAccountIdFromIntRaw(int64(validated.To)),
			Amount:       grpcResp.ToBalance,
		},
		TransferredAt: snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcResp.TransferredAt),
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, response)
}

// Maps transfer error codes to appropriate HTTP status codes.
// Client errors (validation, insufficient balance) return 400;
// everything else (internal failures, timeouts, unknown/empty codes) returns 500.
func httpStatusFromTransferError(code snx_lib_api_json.ErrorCode) HTTPStatusCode {
	switch code {
	case
		snx_lib_status_codes.ErrorCodeAssetNotFound,
		snx_lib_status_codes.ErrorCodeInsufficientMargin,
		snx_lib_status_codes.ErrorCodeInvalidValue,
		snx_lib_status_codes.ErrorCodeValidationError:
		return HTTPStatusCode_400_BadRequest
	default:
		return HTTPStatusCode_500_InternalServerError
	}
}
