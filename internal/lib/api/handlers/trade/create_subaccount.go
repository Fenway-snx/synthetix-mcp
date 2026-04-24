package trade

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/createSubAccount
*/

// SubAccountCreationRequest represents the request to create a new subaccount
type SubAccountCreationRequest struct {
	Name string `json:"name"`
}

// SubAccountCreationResponse represents the response after creating a subaccount
type SubAccountResponse struct {
	SubAccountId      SubAccountId          `json:"subAccountId"`
	MasterAccountId   *SubAccountId         `json:"masterAccountId"`
	Name              string                `json:"subAccountName"`
	Collaterals       []CollateralResponse  `json:"collaterals"`
	MarginSummary     MarginSummary         `json:"crossMarginSummary"`
	Positions         []Position            `json:"positions"`
	MarketPreferences MarketPreferences     `json:"marketPreferences"`
	FeeRates          FeeRateInfoResponse   `json:"feeRates"`
	AccountLimits     AccountLimitsResponse `json:"accountLimits"`
}

type AccountLimitsResponse struct {
	MaxBorrowCapacity  string `json:"maxBorrowCapacity"`
	MaxOrdersPerMarket int64  `json:"maxOrdersPerMarket"`
	MaxSubAccounts     int64  `json:"maxSubAccounts"`
	MaxTotalOrders     int64  `json:"maxTotalOrders"`
}

type FeeRateInfoResponse struct {
	Maker    string `json:"makerFeeRate"`
	Taker    string `json:"takerFeeRate"`
	TierName string `json:"tierName"`
}

type Position struct {
	Symbol            Symbol   `json:"symbol"`
	Side              string   `json:"side"` // "short" or "long"
	EntryPrice        Price    `json:"entryPrice"`
	Quantity          Quantity `json:"quantity"`
	Pnl               string   `json:"pnl"`  // Realized PnL
	Upnl              string   `json:"upnl"` // Unrealized PnL
	UsedMargin        string   `json:"usedMargin"`
	MaintenanceMargin string   `json:"maintenanceMargin"`
	LiquidationPrice  Price    `json:"liquidationPrice"`
}

type CollateralResponse struct {
	AdjustedCollateralValue string                      `json:"adjustedCollateralValue"`
	CollateralValue         string                      `json:"collateralValue"`
	HaircutRate             string                      `json:"haircutRate"`
	HaircutAdjustment       string                      `json:"haircutAdjustment"`
	PendingWithdraw         string                      `json:"pendingWithdraw"`
	Price                   Price                       `json:"price"`
	CalculatedAt            snx_lib_api_types.Timestamp `json:"calculatedAt"` // Unix millis, 0 for USDT
	Quantity                Quantity                    `json:"quantity"`
	Symbol                  Symbol                      `json:"symbol"`
	Withdrawable            string                      `json:"withdrawable"`
}

type MarginSummary struct {
	AccountValue         string `json:"accountValue"`
	AvailableMargin      string `json:"availableMargin"`
	UnrealizedPnl        string `json:"totalUnrealizedPnl"`
	MaintenanceMargin    string `json:"maintenanceMargin"`
	InitialMargin        string `json:"initialMargin"`
	Withdrawable         string `json:"withdrawable"`
	AdjustedAccountValue string `json:"adjustedAccountValue"`
	Debt                 string `json:"debt"`
}

type MarketPreferences struct {
	Leverages map[string]uint32 `json:"leverages"`
}

// Handler for "createSubaccount".
//
//dd:span
func Handle_createSubaccount(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	validated, ok := ctx.ActionPayload().(*snx_lib_api_validation.ValidatedCreateSubaccountAction)
	if !ok || validated == nil || validated.Payload == nil {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	req := SubAccountCreationRequest{
		Name: validated.Payload.Name,
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	grpcResp, err := ctx.SubaccountClient.CreateSubaccount(ctx.Context, &v4grpc.CreateSubaccountRequest{
		TimestampMs:   timestamp_ms,
		TimestampUs:   timestamp_us,
		WalletAddress: string(ctx.WalletAddress),
		Name:          req.Name,
	})
	if err != nil {
		ctx.Logger.Error("Failed to create subaccount", "error", err)
		return handleGRPCError(err, ctx.ClientRequestId)
	}

	subAccountId := snx_lib_api_types.SubAccountIdFromIntUnvalidated(int64(grpcResp.Id))

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, SubAccountResponse{
		SubAccountId: subAccountId,
		Name:         grpcResp.Name,
		AccountLimits: AccountLimitsResponse{
			MaxBorrowCapacity:  grpcResp.MaxBorrowCapacity,
			MaxOrdersPerMarket: grpcResp.MaxOrdersPerMarket,
			MaxSubAccounts:     grpcResp.MaxSubAccounts,
			MaxTotalOrders:     grpcResp.MaxTotalOrders,
		},
	})
}
