// Package sdkparity verifies digest parity between the SDK and service builders.
// It intentionally imports internals the SDK must not depend on.
package sdkparity_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/stretchr/testify/require"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"

	sdk "github.com/synthetixio/synthetix-go/eip712"
)

// Asserts that two EIP-712 typed-data structures hash identically.
func digestsEqual(t *testing.T, label string, want, got sdk.TypedData) {
	t.Helper()

	wantHash, err := sdk.Digest(want)
	require.NoErrorf(t, err, "%s: hash canonical: %v", label, err)
	gotHash, err := sdk.Digest(got)
	require.NoErrorf(t, err, "%s: hash sdk: %v", label, err)

	require.Equalf(t, wantHash, gotHash, "%s: digest mismatch\n  canonical=0x%x\n  sdk      =0x%x", label, wantHash, gotHash)
}

// ---------------------------------------------------------------------
// Auth message
// ---------------------------------------------------------------------

func Test_Parity_AuthMessage(t *testing.T) {
	canonical := snx_lib_auth.CreateEIP712TypedData(
		snx_lib_core.SubAccountId(42),
		1700000000,
		snx_lib_auth.ActionWebSocketAuth,
		snx_lib_auth.DefaultDomainName,
		"1",
		1,
	)
	got := sdk.BuildAuthMessage(42, 1700000000, sdk.ActionWebSocketAuth)

	digestsEqual(t, "AuthMessage", canonical, got)
}

// ---------------------------------------------------------------------
// PlaceOrders
// ---------------------------------------------------------------------

func Test_Parity_PlaceOrders(t *testing.T) {
	payload := &snx_lib_api_validation.PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Grouping: snx_lib_api_validation.GroupingValues("na"),
		Orders: []snx_lib_api_json.PlaceOrderRequest{
			{
				Symbol:        "ETH-USDT",
				Side:          "buy",
				OrderType:     "limitGtc",
				Price:         snx_lib_api_types.Price("100.5"),
				Quantity:      snx_lib_api_types.Quantity("1.25"),
				ClientOrderId: "0xdeadbeef",
			},
		},
	}
	validated := &snx_lib_api_validation.ValidatedPlaceOrdersAction{Payload: payload}
	canonical, err := snx_lib_auth.CreateTradeTypedData("7", 42, 1700, "placeOrders", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildPlaceOrders(7, []sdk.PlaceOrderItem{
		{Symbol: "ETH-USDT", Side: "buy", OrderType: "limitGtc", Price: "100.5", Quantity: "1.25", ClientOrderID: "0xdeadbeef"},
	}, "na", 42, 1700)

	digestsEqual(t, "PlaceOrders", canonical, got)
}

// ---------------------------------------------------------------------
// CancelOrders / CancelOrdersByCloid / CancelAllOrders
// ---------------------------------------------------------------------

func Test_Parity_CancelOrders(t *testing.T) {
	payload := &snx_lib_api_validation.CancelOrdersActionPayload{Action: "cancelOrders"}
	validated := &snx_lib_api_validation.ValidatedCancelOrdersAction{
		Payload:       payload,
		VenueOrderIds: []snx_lib_api_types.VenueOrderId{"1", "2", "3"},
	}
	canonical, err := snx_lib_auth.CreateTradeTypedData("9", 11, 1700, "cancelOrders", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildCancelOrders(9, []uint64{1, 2, 3}, 11, 1700)
	digestsEqual(t, "CancelOrders", canonical, got)

	// Sanity: orderIds shape matches what HashStruct sees on both sides.
	gotIDs, ok := got.Message["orderIds"].([]any)
	require.True(t, ok)
	require.Len(t, gotIDs, 3)
	first, ok := gotIDs[0].(*math.HexOrDecimal256)
	require.True(t, ok)
	require.Equal(t, big.NewInt(1), (*big.Int)(first))
}

func Test_Parity_CancelOrdersByCloid(t *testing.T) {
	payload := &snx_lib_api_validation.CancelOrdersByCloidActionPayload{
		Action:         "cancelOrders",
		ClientOrderIds: []snx_lib_api_types.ClientOrderId{"cli-1", "cli-2"},
	}
	validated := &snx_lib_api_validation.ValidatedCancelOrdersByCloidAction{
		Payload:        payload,
		ClientOrderIds: []snx_lib_api_types.ClientOrderId{"cli-1", "cli-2"},
	}
	canonical, err := snx_lib_auth.CreateTradeTypedData("9", 11, 1700, "cancelOrders", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildCancelOrdersByCloid(9, []string{"cli-1", "cli-2"}, 11, 1700)
	digestsEqual(t, "CancelOrdersByCloid", canonical, got)
}

func Test_Parity_CancelAllOrders(t *testing.T) {
	payload := &snx_lib_api_validation.CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []snx_lib_api_types.Symbol{"ETH-USDT", "BTC-USDT"},
	}
	validated, err := snx_lib_api_validation.NewValidatedCancelAllOrdersAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("9", 11, 1700, "cancelAllOrders", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildCancelAllOrders(9, []string{"ETH-USDT", "BTC-USDT"}, 11, 1700)
	digestsEqual(t, "CancelAllOrders", canonical, got)
}

// ---------------------------------------------------------------------
// ModifyOrder / ModifyOrderByCloid
// ---------------------------------------------------------------------

func Test_Parity_ModifyOrder(t *testing.T) {
	price := snx_lib_api_types.Price("110")
	qty := snx_lib_api_types.Quantity("2")
	payload := &snx_lib_api_validation.ModifyOrderActionPayload{
		Action:       "modifyOrder",
		VenueOrderId: snx_lib_api_types.VenueOrderId("123"),
		Price:        &price,
		Quantity:     &qty,
	}
	validated, err := snx_lib_api_validation.NewValidatedModifyOrderAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("9", 11, 1700, "modifyOrder", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildModifyOrder(9, 123, "110", "2", "", 11, 1700)
	digestsEqual(t, "ModifyOrder", canonical, got)
}

func Test_Parity_ModifyOrderByCloid(t *testing.T) {
	price := snx_lib_api_types.Price("110")
	qty := snx_lib_api_types.Quantity("2")
	payload := &snx_lib_api_validation.ModifyOrderByCloidActionPayload{
		Action:        "modifyOrder",
		ClientOrderId: snx_lib_api_types.ClientOrderId("cli-x"),
		Price:         &price,
		Quantity:      &qty,
	}
	validated, err := snx_lib_api_validation.NewValidatedModifyOrderByCloidAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("9", 11, 1700, "modifyOrder", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildModifyOrderByCloid(9, "cli-x", "110", "2", "", 11, 1700)
	digestsEqual(t, "ModifyOrderByCloid", canonical, got)
}

// ---------------------------------------------------------------------
// UpdateLeverage
// ---------------------------------------------------------------------

func Test_Parity_UpdateLeverage(t *testing.T) {
	payload := &snx_lib_api_validation.UpdateLeverageActionPayload{
		Action:   "updateLeverage",
		Symbol:   "ETH-USDT",
		Leverage: "5",
	}
	validated, err := snx_lib_api_validation.NewValidatedUpdateLeverageAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("9", 11, 1700, "updateLeverage", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildUpdateLeverage(9, "ETH-USDT", "5", 11, 1700)
	digestsEqual(t, "UpdateLeverage", canonical, got)
}

// ---------------------------------------------------------------------
// WithdrawCollateral
// ---------------------------------------------------------------------

func Test_Parity_WithdrawCollateral(t *testing.T) {
	payload := &snx_lib_api_validation.WithdrawCollateralActionPayload{
		Action:      "withdrawCollateral",
		Symbol:      snx_lib_api_types.Asset("USDT"),
		Amount:      "100",
		Destination: snx_lib_api_types.WalletAddress("0xAbCdEf0123456789AbCdEf0123456789AbCdEf01"),
	}
	validated, err := snx_lib_api_validation.NewValidatedWithdrawCollateralAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("9", 11, 1700, "withdrawCollateral", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildWithdrawCollateral(9, "USDT", "100", "0xAbCdEf0123456789AbCdEf0123456789AbCdEf01", 11, 1700)
	digestsEqual(t, "WithdrawCollateral", canonical, got)
}

// ---------------------------------------------------------------------
// CreateSubaccount / TransferCollateral / UpdateSubAccountName
// ---------------------------------------------------------------------

func Test_Parity_CreateSubaccount(t *testing.T) {
	payload := &snx_lib_api_validation.CreateSubaccountActionPayload{
		Action: "createSubaccount",
		Name:   "TradingDesk",
	}
	validated, err := snx_lib_api_validation.NewValidatedCreateSubaccountAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "createSubaccount", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildCreateSubaccount(3, "TradingDesk", 11, 1700)
	digestsEqual(t, "CreateSubaccount", canonical, got)
}

func Test_Parity_TransferCollateral(t *testing.T) {
	payload := &snx_lib_api_validation.TransferCollateralActionPayload{
		Action: "transferCollateral",
		To:     "8",
		Symbol: snx_lib_api_types.Asset("USDT"),
		Amount: "250",
	}
	validated, err := snx_lib_api_validation.NewValidatedTransferCollateralAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "transferCollateral", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildTransferCollateral(3, 8, "USDT", "250", 11, 1700)
	digestsEqual(t, "TransferCollateral", canonical, got)
}

func Test_Parity_UpdateSubAccountName(t *testing.T) {
	payload := &snx_lib_api_validation.UpdateSubAccountNameActionPayload{
		Action: "updateSubAccountName",
		Name:   "NewName",
	}
	validated, err := snx_lib_api_validation.NewValidatedUpdateSubAccountNameAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "updateSubAccountName", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildUpdateSubAccountName(3, "NewName", 11, 1700)
	digestsEqual(t, "UpdateSubAccountName", canonical, got)
}

// ---------------------------------------------------------------------
// Delegated signer lifecycle
// ---------------------------------------------------------------------

func Test_Parity_AddDelegatedSigner(t *testing.T) {
	expiresAt := int64(1900000000)
	payload := &snx_lib_api_validation.AddDelegatedSignerActionPayload{
		Action:          "addDelegatedSigner",
		DelegateAddress: snx_lib_api_types.WalletAddress("0xAbCdEf0123456789AbCdEf0123456789AbCdEf01"),
		Permissions:     []string{"trading"},
		ExpiresAt:       &expiresAt,
	}
	validated, err := snx_lib_api_validation.NewValidatedAddDelegatedSignerAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "addDelegatedSigner", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildAddDelegatedSigner(3, "0xAbCdEf0123456789AbCdEf0123456789AbCdEf01", []string{"trading"}, 1900000000, 11, 1700)
	digestsEqual(t, "AddDelegatedSigner", canonical, got)
}

func Test_Parity_RemoveDelegatedSigner(t *testing.T) {
	payload := &snx_lib_api_validation.RemoveDelegatedSignerActionPayload{
		Action:          "removeDelegatedSigner",
		DelegateAddress: snx_lib_api_types.WalletAddress("0xAbCdEf0123456789AbCdEf0123456789AbCdEf01"),
	}
	validated, err := snx_lib_api_validation.NewValidatedRemoveDelegatedSignerAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "removeDelegatedSigner", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildRemoveDelegatedSigner(3, "0xAbCdEf0123456789AbCdEf0123456789AbCdEf01", 11, 1700)
	digestsEqual(t, "RemoveDelegatedSigner", canonical, got)
}

func Test_Parity_RemoveAllDelegatedSigners(t *testing.T) {
	payload := &snx_lib_api_validation.RemoveAllDelegatedSignersActionPayload{Action: "removeAllDelegatedSigners"}
	validated, err := snx_lib_api_validation.NewValidatedRemoveAllDelegatedSignersAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "removeAllDelegatedSigners", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildRemoveAllDelegatedSigners(3, 11, 1700)
	digestsEqual(t, "RemoveAllDelegatedSigners", canonical, got)
}

// ---------------------------------------------------------------------
// ScheduleCancel
// ---------------------------------------------------------------------

func Test_Parity_ScheduleCancel(t *testing.T) {
	timeoutSec := int64(60)
	payload := &snx_lib_api_validation.ScheduleCancelActionPayload{
		Action:         "scheduleCancel",
		TimeoutSeconds: &timeoutSec,
	}
	validated, err := snx_lib_api_validation.NewValidatedScheduleCancelAction(payload)
	require.NoError(t, err)

	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "scheduleCancel", validated, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildScheduleCancel(3, 60, 11, 1700)
	digestsEqual(t, "ScheduleCancel", canonical, got)
}

// ---------------------------------------------------------------------
// SubAccountAction (generic GET-style read)
// ---------------------------------------------------------------------

func Test_Parity_SubAccountAction_WithNonce(t *testing.T) {
	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 11, 1700, "getPositions", nil, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildSubAccountAction(3, "getPositions", 11, 1700)
	digestsEqual(t, "SubAccountAction(getPositions, nonce>0)", canonical, got)
}

func Test_Parity_SubAccountAction_NoNonce(t *testing.T) {
	canonical, err := snx_lib_auth.CreateTradeTypedData("3", 0, 1700, "getOpenOrders", nil, snx_lib_auth.DefaultDomainName, "1", 1)
	require.NoError(t, err)

	got := sdk.BuildSubAccountAction(3, "getOpenOrders", 0, 1700)
	digestsEqual(t, "SubAccountAction(getOpenOrders, nonce=0)", canonical, got)
}
