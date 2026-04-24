// NOTE: this implementation is incomplete, insofar as the SubAccount
// Service is not yet able to provide in its response the information
// required for the API response.
//
// NOTE: agreed to de-scope MaxLeverage and IsCross, to be dealt with
// at a later time (see SNX-3317 for details).

package trade

import (
	"fmt"
	"strconv"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/updateLeverage
*/

const (
	MinLeverage = 1
)

// Request structure
type updateLeverageRequest struct {
	Symbol   Symbol `json:"symbol"`
	Leverage string `json:"leverage"`
}

// Response structure, as per the spec.
type updateLeverageResponse struct {
	Symbol           Symbol `json:"symbol"`
	NewLeverage      string `json:"newLeverage"`
	PreviousLeverage string `json:"previousLeverage"`
}

// Local helper functions

// Transforms external request to internal request
func translateUpdateLeverageRequest(
	subAccountId snx_lib_core.SubAccountId,
	leverageValue uint32,
	req updateLeverageRequest,
) (
	*v4grpc.UpdateSubAccountMarketLeverageRequest, // coreReq
	error, // err
) {

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	coreReq := &v4grpc.UpdateSubAccountMarketLeverageRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: int64(subAccountId),
		Symbol:       string(req.Symbol),
		Leverage:     leverageValue,
	}

	return coreReq, nil
}

// Validates an incoming request, making corrections as appropriate or
// returning an error.
func validateUpdateLeverageRequest(req *updateLeverageRequest) (
	uint32, // leverageValue
	error, // err
) {

	// Rules:
	//
	// - `Symbol` not empty;
	// - `Leverage` a valid number;

	if req.Symbol == "" {
		return 0, ErrSymbolNameEmpty
	}

	leverageValue, err := strconv.ParseUint(req.Leverage, 10, 32)
	if err != nil {
		return 0, fmt.Errorf(`"leverage" must be a valid number: %v`, err)
	}

	return uint32(leverageValue), nil
}

// Handler for "updateLeverage".
//
//dd:span
func Handle_updateLeverage(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*ValidatedUpdateLeverageAction)
	if !ok || validated == nil || validated.Payload == nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	req := updateLeverageRequest{
		Leverage: validated.Payload.Leverage,
		Symbol:   Symbol(validated.Payload.Symbol),
	}

	// further (semantic) request validation
	leverageValue, err := validateUpdateLeverageRequest(&req)
	if err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid request", map[string]string{"s": err.Error()})

		return HTTPStatusCode_400_BadRequest, resp
	}

	// convert to core type(s)
	coreReq, err := translateUpdateLeverageRequest(
		ctx.SelectedAccountId,
		leverageValue,
		req,
	)
	if err != nil {
		resp := snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, "Invalid request", map[string]string{"s": err.Error()})

		return HTTPStatusCode_400_BadRequest, resp
	}

	grpcResp, err := ctx.SubaccountClient.UpdateSubAccountMarketLeverage(ctx, coreReq)
	if err != nil {
		const FailureMessage = "Failed to update leverage"

		ctx.Logger.Error(FailureMessage, "error", err)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, FailureMessage, err)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	ctx.Logger.Debug("updated leverage successfully",
		"message", grpcResp.Message,
		"prev_leverage", grpcResp.PrevLeverage,
		"new_leverage", grpcResp.NewLeverage,
	)

	prevLeverage := strconv.FormatUint(uint64(grpcResp.PrevLeverage), 10)
	newLeverage := strconv.FormatUint(uint64(grpcResp.NewLeverage), 10)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, &updateLeverageResponse{
		Symbol:           req.Symbol,
		NewLeverage:      newLeverage,
		PreviousLeverage: prevLeverage,
	})
}
