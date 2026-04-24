package trade

import (
	"strconv"
	"time"

	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	transferDefaultLimit  = 50
	transferMaxLimit      = 1_000
	transfersMaxTimeRange = 30 * 24 * time.Hour
)

type GetTransfersRequest struct {
	Symbol    Asset     `json:"symbol"`    // Optional: filter by collateral symbol // TODO: SNX-6098: rename to `AssetName`
	Limit     int       `json:"limit"`     // Optional: max number of transfers (default 50, max 1000)
	Offset    int       `json:"offset"`    // Optional: pagination offset (default 0)
	StartTime Timestamp `json:"startTime"` // Optional: start timestamp in milliseconds
	EndTime   Timestamp `json:"endTime"`   // Optional: end timestamp in milliseconds
}

type GetTransfersResponseItem struct {
	TransferId   string       `json:"transferId"`
	From         SubAccountId `json:"from"`
	To           SubAccountId `json:"to"`
	Symbol       Symbol       `json:"symbol"`
	Amount       string       `json:"amount"`
	TransferType string       `json:"transferType"`
	Status       string       `json:"status"`
	ErrorMessage string       `json:"errorMessage,omitempty"`
	Timestamp    Timestamp    `json:"timestamp"`
}

type GetTransfersResponse struct {
	Transfers []GetTransfersResponseItem `json:"transfers"`
	Total     int                        `json:"total"`
}

func validateGetTransfersRequest(req *GetTransfersRequest) error {
	if req.Symbol != "" {
		symbol, err := snx_lib_api_validation.ValidateAndNormalizeCollateralSymbol(req.Symbol)
		if err != nil {
			return err
		}
		req.Symbol = symbol
	}

	if err := snx_lib_api_validation.ValidateNonNegative(req.Limit, API_WKS_limit); err != nil {
		return err
	}

	if err := snx_lib_api_validation.ValidateMaxLimit(req.Limit, transferMaxLimit, API_WKS_limit); err != nil {
		return err
	}

	if err := snx_lib_api_validation.ValidateNonNegative(req.Offset, API_WKS_offset); err != nil {
		return err
	}

	if err := snx_lib_api_validation.ValidateTimestampRange(req.StartTime, req.EndTime, transfersMaxTimeRange, "transfers"); err != nil {
		return err
	}

	return nil
}

func Handle_getTransfers(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_status_codes.ErrorCodeValidationError, "subAccountId is required", nil)
	}

	var req GetTransfersRequest
	if err := mapstructure.Decode(params, &req); err != nil {

		ctx.Logger.Error("failed to decode getTransfers request",
			"error", err,
		)

		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_status_codes.ErrorCodeInvalidFormat, "invalid request format", nil)
	}

	if err := validateGetTransfersRequest(&req); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	if req.Limit == 0 {
		req.Limit = transferDefaultLimit
	}

	sid := ctx.SelectedAccountId

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	grpcReq := &v4grpc.GetTransfersRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(sid),
	}

	// Add time range if provided
	now := snx_lib_api_types.TimestampNow()
	if startTime, endTime, err, failureQualifier := snx_lib_api_handlers_utils.APIStartEndToCoreStartEndPtrs(req.StartTime, req.EndTime, now); err != nil {

		resp := snx_lib_api_json.NewValidationErrorResponse[any](
			ctx.ClientRequestId,
			"invalid request parameters",
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

	// Add pagination
	grpcReq.Offset = int64(req.Offset)
	grpcReq.Limit = int64(req.Limit)

	// Add symbol filter if provided
	if req.Symbol != "" {
		sym := string(req.Symbol)
		grpcReq.Symbol = &sym
	}

	grpcResp, err := ctx.SubaccountClient.GetTransfers(ctx, grpcReq)
	if err != nil {

		ctx.Logger.Error("failed to get transfers from subaccount service",
			"error", err,
		)

		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](
			ctx.ClientRequestId,
			"failed to get transfers",
			err,
		)
	}

	transfers := make([]GetTransfersResponseItem, 0, len(grpcResp.Transfers))
	for _, grpcTransfer := range grpcResp.Transfers {
		transferredAt := snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcTransfer.TransferredAt)

		transfer := GetTransfersResponseItem{
			TransferId:   strconv.FormatInt(grpcTransfer.TransferId, 10),
			From:         snx_lib_api_types.SubAccountIdFromIntUnvalidated(int64(grpcTransfer.FromSubAccountId)),
			To:           snx_lib_api_types.SubAccountIdFromIntUnvalidated(int64(grpcTransfer.ToSubAccountId)),
			Symbol:       Symbol(grpcTransfer.Symbol),
			Amount:       grpcTransfer.Amount,
			TransferType: grpcTransfer.TransferType,
			Status:       grpcTransfer.Status,
			ErrorMessage: grpcTransfer.ErrorMessage,
			Timestamp:    transferredAt,
		}

		transfers = append(transfers, transfer)
	}

	response := GetTransfersResponse{
		Transfers: transfers,
		Total:     int(grpcResp.TotalCount),
	}

	ctx.Logger.Info("GetTransfers request completed",
		"sub_account_id", sid,
		"returned_count", len(transfers),
		"total_count", grpcResp.TotalCount,
	)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, response)
}
