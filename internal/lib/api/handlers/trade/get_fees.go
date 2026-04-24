package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getFees
*/

// UserFee represents a user's fee structure

type FeeScheduleEntry struct {
	Symbol  Symbol `json:"symbol"`  // Trading pair symbol (e.g., "BTC-USD")
	FeeRate string `json:"feeRate"` // Fee rate (e.g., "0.001" for 0.1%)
}

type UserFeeResponse struct {
	FeeTiers        []FeeScheduleEntry `json:"feeTiers"`
	UserDailyVolume string             `json:"userDailyVolume"`
	UserFeeTier     FeeScheduleEntry   `json:"userFeeTier"`
}

// generateMockUserFees creates mock user fees data
func generateMockUserFees() UserFeeResponse {
	return UserFeeResponse{
		FeeTiers: []FeeScheduleEntry{
			{
				Symbol:  "BTC-USD",
				FeeRate: "0.001",
			},
		},
		UserDailyVolume: "1000000",
		UserFeeTier: FeeScheduleEntry{
			Symbol:  "BTC-USD",
			FeeRate: "0.001",
		},
	}
}

// Handler for "getFees".
//
//dd:span
func Handle_getFees(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	/*var req TradeRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any]("", snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}*/

	userFees := generateMockUserFees()

	resp := snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, userFees)

	return HTTPStatusCode_200_OK, resp
}
