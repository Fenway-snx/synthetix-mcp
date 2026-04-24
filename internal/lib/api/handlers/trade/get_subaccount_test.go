package trade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

func Test_MapSubaccountInfoHandlesNilMarginSummary(t *testing.T) {
	subaccount := &v4grpc.SubaccountInfo{
		Name:        "Test",
		Id:          1,
		Collaterals: []*v4grpc.CollateralInfo{},
		Positions:   []*v4grpc.PositionItem{},
		Leverages:   map[string]uint32{},
		// MarginSummary intentionally nil
	}

	require.NotPanics(t, func() {
		_ = mapSubaccountInfo(subaccount)
	})

	result := mapSubaccountInfo(subaccount)
	require.Equal(t, SubAccountId("1"), result.SubAccountId)
	require.Equal(t, MarginSummary{}, result.MarginSummary)
}

func Test_MapSubaccountInfoHandlesNilInput(t *testing.T) {
	require.Equal(t, SubAccountResponse{}, mapSubaccountInfo(nil))
}

func Test_MapSubaccountInfo_CollateralFields(t *testing.T) {
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)

	subaccount := &v4grpc.SubaccountInfo{
		Name: "Test",
		Id:   1,
		Collaterals: []*v4grpc.CollateralInfo{
			{
				Collateral:              "BTC",
				Quantity:                "0.5",
				WithdrawableAmount:      "0.4",
				PendingWithdrawalAmount: "0.1",
				CollateralValue:         "30000",
				AdjustedCollateralValue: "27000",
				HaircutRate:             "0.1",
				HaircutAdjustment:       "500",
				Price:                   "60000",
				CalculatedAt:            timestamppb.New(now),
			},
			{
				Collateral:              "USDT",
				Quantity:                "1000",
				WithdrawableAmount:      "1000",
				CollateralValue:         "1000",
				AdjustedCollateralValue: "1000",
				HaircutRate:             "0",
				HaircutAdjustment:       "0",
				Price:                   "1",
				// CalculatedAt nil for USDT
			},
		},
		Positions: []*v4grpc.PositionItem{},
		Leverages: map[string]uint32{},
		MarginSummary: &v4grpc.MarginSummary{
			AccountValue:         "30000",
			AvailableMargin:      "25000",
			UnrealizedPnl:        "0",
			MaintenanceMargin:    "1500",
			InitialMargin:        "3000",
			Withdrawable:         "25000",
			AdjustedAccountValue: "27000",
			Debt:                 "500",
		},
	}

	result := mapSubaccountInfo(subaccount)

	// BTC collateral
	require.Len(t, result.Collaterals, 2)
	btc := result.Collaterals[0]
	require.Equal(t, Symbol("BTC"), btc.Symbol)
	require.Equal(t, Quantity("0.5"), btc.Quantity)
	require.Equal(t, "0.4", btc.Withdrawable)
	require.Equal(t, "0.1", btc.PendingWithdraw)
	require.Equal(t, "30000", btc.CollateralValue)
	require.Equal(t, "27000", btc.AdjustedCollateralValue)
	require.Equal(t, "0.1", btc.HaircutRate)
	require.Equal(t, "500", btc.HaircutAdjustment)
	require.Equal(t, Price("60000"), btc.Price)
	expectedTs, _ := snx_lib_api_types.TimestampFromTimestampPB(timestamppb.New(now))
	require.Equal(t, expectedTs, btc.CalculatedAt)

	// USDT collateral — CalculatedAt should be zero (nil proto timestamp)
	usdt := result.Collaterals[1]
	require.Equal(t, Symbol("USDT"), usdt.Symbol)
	require.Equal(t, Price("1"), usdt.Price)
	require.Equal(t, "0", usdt.HaircutRate)
	require.Equal(t, "0", usdt.HaircutAdjustment)
	require.Equal(t, Timestamp(0), usdt.CalculatedAt)

	// Margin summary
	require.Equal(t, "30000", result.MarginSummary.AccountValue)
	require.Equal(t, "27000", result.MarginSummary.AdjustedAccountValue)
	require.Equal(t, "500", result.MarginSummary.Debt)
}
