package trade

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getBalanceUpdates
*/

const (
	defaultAmountOfUpdates = 50
	maxAmountOfUpdates     = 1_000
	maxOffset              = 10_000

	balanceUpdatesDefaultWindowMs = int64(7 * 24 * time.Hour / time.Millisecond)
	balanceUpdatesMaximumDuration = 365 * 24 * time.Hour
)

var (
	validFilters = []string{
		snx_lib_core.TransactionActionDeposit,
		snx_lib_core.TransactionActionWithdrawal,
		snx_lib_core.TransactionActionTransfer,
	}

	validFiltersString = strings.Join(validFilters, ", ")
)

type GetBalanceUpdatesRequest struct {
	ActionFilter string    `json:"actionFilter"` // filter by action type (DEPOSIT, WITHDRAWAL, TRANSFER)
	Limit        int64     `json:"limit"`        // max number of results (default 50, max 1000)
	Offset       int64     `json:"offset"`       // pagination offset (default 0, max 10000)
	StartTime    Timestamp `json:"startTime"`    // optional: start of time range (ms)
	EndTime      Timestamp `json:"endTime"`      // optional: end of time range (ms)
}

// A single deposit, withdrawal, or transfer row returned to clients.
type BalanceUpdate struct {
	Id                 string         `json:"id"`                           // Transaction ID
	SubAccountId       SubAccountId   `json:"subAccountId"`                 // Subaccount ID
	Action             string         `json:"action"`                       // Action type (DEPOSIT, WITHDRAWAL, TRANSFER)
	Status             string         `json:"status"`                       // Transaction status (e.g., "success", "completed", "pending")
	Amount             string         `json:"amount"`                       // Transaction amount (net)
	Fee                string         `json:"fee"`                          // Fee charged for the transaction
	GrossAmount        string         `json:"grossAmount"`                  // Total amount including fee (amount + fee)
	Collateral         string         `json:"collateral"`                   // Collateral symbol (e.g., "USDT", "WETH")
	Timestamp          Timestamp      `json:"timestamp"`                    // When the transaction was created
	DestinationAddress *WalletAddress `json:"destinationAddress,omitempty"` // Destination address for withdrawals
	TxHash             *TxHash        `json:"txHash,omitempty"`             // On-chain transaction hash if applicable
	FromSubAccountId   *SubAccountId  `json:"fromSubAccountId,omitempty"`   // Source subaccount ID for transfers
	ToSubAccountId     *SubAccountId  `json:"toSubAccountId,omitempty"`     // Target subaccount ID for transfers
}

// JSON envelope listing balance update rows.
type GetBalanceUpdatesResponse struct {
	BalanceUpdates []BalanceUpdate `json:"balanceUpdates"` // Array of balance updates
}

func validateGetBalanceUpdatesRequest(req *GetBalanceUpdatesRequest) error {
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.ActionFilter, snx_lib_api_validation.MaxEnumFieldLength*4, "actionFilter"); err != nil {
		return err
	}

	// Validate limit
	if err := snx_lib_api_validation.ValidateNonNegative(int(req.Limit), API_WKS_limit); err != nil {
		return err
	}

	if err := snx_lib_api_validation.ValidateMaxLimit(int(req.Limit), maxAmountOfUpdates, API_WKS_limit); err != nil {
		return err
	}

	// Validate offset
	if err := snx_lib_api_validation.ValidateNonNegative(int(req.Offset), API_WKS_offset); err != nil {
		return err
	}

	if err := snx_lib_api_validation.ValidateMaxLimit(int(req.Offset), maxOffset, API_WKS_offset); err != nil {
		return err
	}

	if err := snx_lib_api_validation.ValidateTimestampRange(req.StartTime, req.EndTime, balanceUpdatesMaximumDuration, "balanceUpdates"); err != nil {
		return err
	}

	return nil
}

// Fills missing time bounds using the current instant from TimestampNow().
// When both omitted, uses [now−7d, now]; when one side is set, extends by seven days (capping end at
// now; if the computed start would be non-positive it is set to 1 ms so the lower bound is not
// treated as omitted downstream).
func applyBalanceUpdatesDefaultTimeWindow(req *GetBalanceUpdatesRequest) {
	if req.StartTime != 0 && req.EndTime != 0 {
		return
	}
	now := snx_lib_api_types.TimestampNow()
	if req.StartTime == 0 && req.EndTime == 0 {
		req.EndTime = now
		startMs := int64(now) - balanceUpdatesDefaultWindowMs
		if startMs <= 0 {
			req.StartTime = Timestamp(1)
		} else {
			req.StartTime = Timestamp(startMs)
		}
		return
	}
	if req.StartTime != 0 {
		req.EndTime = req.StartTime + Timestamp(balanceUpdatesDefaultWindowMs)
		if req.EndTime > now {
			req.EndTime = now
		}
		return
	}
	if req.EndTime != 0 {
		if req.EndTime > now {
			req.EndTime = now
		}
		startMs := int64(req.EndTime) - balanceUpdatesDefaultWindowMs
		if startMs <= 0 {
			req.StartTime = Timestamp(1)
			return
		}
		req.StartTime = Timestamp(startMs)
	}
}

func validateActionFilters(filters string) ([]string, error) {
	trimmedFilters := strings.TrimSpace(filters)

	// If no filters provided, return nil (service will default to all action types)
	if len(trimmedFilters) == 0 {
		return nil, nil
	}

	data := strings.Split(trimmedFilters, ",")
	parsedFilters := make([]string, 0, len(data))

	for _, d := range data {
		trimmed := strings.TrimSpace(d)
		if trimmed == "" {
			continue
		}

		if slices.Contains(validFilters, trimmed) {
			parsedFilters = append(parsedFilters, trimmed)
		} else {
			return nil, fmt.Errorf("invalid action filter, available filters: %v", validFiltersString)
		}
	}

	return parsedFilters, nil
}

//dd:span
func Handle_getBalanceUpdates(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	// Validate that a subAccountId was provided and authenticated
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subAccountId is required", nil)
	}

	var req GetBalanceUpdatesRequest
	if err := mapstructure.Decode(params, &req); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request format", nil)
	}

	// Validate request
	if err := validateGetBalanceUpdatesRequest(&req); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	applyBalanceUpdatesDefaultTimeWindow(&req)

	// Set defaults
	if req.Limit == 0 {
		req.Limit = defaultAmountOfUpdates
	}

	actionFilters, err := validateActionFilters(req.ActionFilter)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	// Use the authenticated subaccount ID from context
	subAccountId := ctx.SelectedAccountId

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// Create gRPC request
	grpcReq := &v4grpc.GetBalanceUpdatesRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(subAccountId),
		ActionFilter: actionFilters,
		Offset:       req.Offset,
		Limit:        req.Limit,
	}

	now := snx_lib_api_types.TimestampNow()
	if startTime, endTime, err, failureQualifier := snx_lib_api_handlers_utils.APIStartEndToCoreStartEndPtrs(req.StartTime, req.EndTime, now); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](
			ctx.ClientRequestId,
			"invalid request parameters",
			map[string]string{
				"error":     err.Error(),
				"qualifier": failureQualifier,
			},
		)
	} else {
		grpcReq.StartTime = startTime
		grpcReq.EndTime = endTime
	}

	// Call subaccount service
	grpcResp, err := ctx.SubaccountClient.GetBalanceUpdates(ctx, grpcReq)
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Failed to retrieve balance updates", err)
	}

	// Transform gRPC response to API response
	balanceUpdates := make([]BalanceUpdate, 0, len(grpcResp.BalanceUpdates))
	for _, grpcUpdate := range grpcResp.BalanceUpdates {
		var timestamp Timestamp
		if grpcUpdate.CreatedAt != nil {
			timestamp = snx_lib_api_types.TimestampFromTimestampPBOrZero(grpcUpdate.CreatedAt)
		}

		update := BalanceUpdate{
			Id:           strconv.FormatInt(grpcUpdate.Id, 10),
			SubAccountId: snx_lib_api_types.SubAccountIdFromIntUnvalidated(grpcUpdate.SubAccountId),
			Action:       grpcUpdate.Action,
			Status:       grpcUpdate.Status,
			Amount:       grpcUpdate.Amount,
			Fee:          grpcUpdate.Fee,
			GrossAmount:  grpcUpdate.GrossAmount,
			Collateral:   grpcUpdate.Collateral,
			Timestamp:    timestamp,
		}

		// Set optional fields
		if grpcUpdate.DestinationAddress != nil {
			update.DestinationAddress = snx_lib_api_types.WalletAddressPtrFromStringPtrUnvalidated(grpcUpdate.DestinationAddress)
		}
		if grpcUpdate.TxHash != nil {
			update.TxHash = snx_lib_api_types.TxHashPtrFromStringPtrUnvalidated(grpcUpdate.TxHash)
		}
		if grpcUpdate.FromSubAccountId != nil {
			fromSubAccountId := snx_lib_api_types.SubAccountIdFromIntUnvalidated(*grpcUpdate.FromSubAccountId)
			update.FromSubAccountId = &fromSubAccountId
		}
		if grpcUpdate.ToSubAccountId != nil {
			toSubAccountId := snx_lib_api_types.SubAccountIdFromIntUnvalidated(*grpcUpdate.ToSubAccountId)
			update.ToSubAccountId = &toSubAccountId
		}

		balanceUpdates = append(balanceUpdates, update)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, GetBalanceUpdatesResponse{
		BalanceUpdates: balanceUpdates,
	})
}
