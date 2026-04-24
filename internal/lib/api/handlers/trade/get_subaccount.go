package trade

import (
	"fmt"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getSubAccount
*/

// Handler for "getSubAccount".
//
//dd:span
func Handle_getSubAccount(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		resp := snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, ErrorCodeValidationError, "subaccountId is required", nil)
		return HTTPStatusCode_400_BadRequest, resp
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.GetSubaccount(ctx.Context, &v4grpc.GetSubaccountRequest{
		TimestampMs:  timestampMs,
		TimestampUs:  timestampUs,
		SubAccountId: int64(ctx.SelectedAccountId),
	})
	if err != nil {
		failMessage := "Failed to get subaccount by id"

		ctx.Logger.Error(failMessage, "error", err)
		resp := snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, fmt.Sprintf("%s %d", failMessage, ctx.SelectedAccountId), err)
		return HTTPStatusCode_500_InternalServerError, resp
	}

	result := mapSubaccountInfo(grpcResp)

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, result)
}

func mapSubaccountInfo(subaccount *v4grpc.SubaccountInfo) SubAccountResponse {
	if subaccount == nil {
		return SubAccountResponse{}
	}

	collaterals := make([]CollateralResponse, 0, len(subaccount.Collaterals))
	for _, collateral := range subaccount.Collaterals {
		cr := CollateralResponse{
			AdjustedCollateralValue: collateral.AdjustedCollateralValue,
			CollateralValue:         collateral.CollateralValue,
			HaircutRate:             collateral.HaircutRate,
			HaircutAdjustment:       collateral.HaircutAdjustment,
			PendingWithdraw:         collateral.PendingWithdrawalAmount,
			Price:                   snx_lib_api_types.PriceFromStringUnvalidated(collateral.Price),
			Quantity:                snx_lib_api_types.QuantityFromStringUnvalidated(collateral.Quantity),
			Symbol:                  Symbol(collateral.Collateral),
			Withdrawable:            collateral.WithdrawableAmount,
		}
		if collateral.CalculatedAt != nil {
			cr.CalculatedAt, _ = snx_lib_api_types.TimestampFromTimestampPB(collateral.CalculatedAt)
		}
		collaterals = append(collaterals, cr)
	}

	positions := make([]Position, 0, len(subaccount.Positions))
	for _, position := range subaccount.Positions {
		positions = append(positions, Position{
			Symbol:            Symbol(position.Symbol),
			Side:              position.Side,
			EntryPrice:        snx_lib_api_types.PriceFromStringUnvalidated(position.EntryPrice),
			Quantity:          snx_lib_api_types.QuantityFromStringUnvalidated(position.Quantity),
			Pnl:               position.Pnl,
			Upnl:              position.Upnl,
			UsedMargin:        position.UsedMargin,
			MaintenanceMargin: position.MaintenanceMargin,
			LiquidationPrice:  snx_lib_api_types.PriceFromStringUnvalidated(position.LiquidationPrice),
		})
	}

	var masterAccountId *SubAccountId
	if subaccount.MasterAccountId != nil {
		msid := snx_lib_api_types.SubAccountIdFromIntUnvalidated(*subaccount.MasterAccountId)

		masterAccountId = &msid
	}

	var summary MarginSummary
	if subaccount.MarginSummary != nil {
		summary = MarginSummary{
			AccountValue:         subaccount.MarginSummary.AccountValue,
			AvailableMargin:      subaccount.MarginSummary.AvailableMargin,
			UnrealizedPnl:        subaccount.MarginSummary.UnrealizedPnl,
			MaintenanceMargin:    subaccount.MarginSummary.MaintenanceMargin,
			InitialMargin:        subaccount.MarginSummary.InitialMargin,
			Withdrawable:         subaccount.MarginSummary.Withdrawable,
			AdjustedAccountValue: subaccount.MarginSummary.AdjustedAccountValue,
			Debt:                 subaccount.MarginSummary.Debt,
		}
	}

	feeRates := FeeRateInfoResponse{
		Maker:    subaccount.MakerFeeRate,
		Taker:    subaccount.TakerFeeRate,
		TierName: subaccount.TierName,
	}

	return SubAccountResponse{
		SubAccountId:    snx_lib_api_types.SubAccountIdFromIntUnvalidated(subaccount.Id),
		MasterAccountId: masterAccountId,
		Name:            subaccount.Name,
		Collaterals:     collaterals,
		MarginSummary:   summary,
		Positions:       positions,
		MarketPreferences: MarketPreferences{
			Leverages: subaccount.Leverages,
		},
		FeeRates: feeRates,
		AccountLimits: AccountLimitsResponse{
			MaxBorrowCapacity:  subaccount.MaxBorrowCapacity,
			MaxOrdersPerMarket: subaccount.MaxOrdersPerMarket,
			MaxSubAccounts:     subaccount.MaxSubAccounts,
			MaxTotalOrders:     subaccount.MaxTotalOrders,
		},
	}
}
