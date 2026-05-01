package auth

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
)

func Test_CreateTradeTypedData_ModifyOrderOptionalFieldsNormalization(t *testing.T) {
	payload := &snx_lib_api_validation.ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: snx_lib_api_types.VenueOrderId("1")}
	validated := &snx_lib_api_validation.ValidatedModifyOrderAction{Payload: payload, VenueOrderId: snx_lib_api_types.VenueOrderId("1")}

	typed, err := CreateTradeTypedData("1", 1, 100, "modifyOrder", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	price, ok := typed.Message["price"].(string)
	require.True(t, ok, "price should be a string")
	require.Equal(t, "", price)

	quantity, ok := typed.Message["quantity"].(string)
	require.True(t, ok, "quantity should be a string")
	require.Equal(t, "", quantity)

	triggerPrice, ok := typed.Message["triggerPrice"].(string)
	require.True(t, ok, "triggerPrice should be a string")
	require.Equal(t, "", triggerPrice)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_ModifyOrderOptionalFieldsExplicitNil(t *testing.T) {
	payload := &snx_lib_api_validation.ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Price: nil, Quantity: nil, TriggerPrice: nil}
	validated := &snx_lib_api_validation.ValidatedModifyOrderAction{Payload: payload, VenueOrderId: snx_lib_api_types.VenueOrderId("1")}

	typed, err := CreateTradeTypedData("1", 1, 100, "modifyOrder", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Equal(t, "", typed.Message["price"])
	require.Equal(t, "", typed.Message["quantity"])
	require.Equal(t, "", typed.Message["triggerPrice"])

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_ModifyOrderOptionalFieldsUsesProvidedValues(t *testing.T) {
	price := snx_lib_api_types.Price("10")
	quantity := snx_lib_api_types.Quantity("5")
	triggerPrice := snx_lib_api_types.Price("15")
	payload := &snx_lib_api_validation.ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Price: &price, Quantity: &quantity, TriggerPrice: &triggerPrice}
	validated := &snx_lib_api_validation.ValidatedModifyOrderAction{Payload: payload, VenueOrderId: snx_lib_api_types.VenueOrderId("1")}

	typed, err := CreateTradeTypedData("1", 1, 100, "modifyOrder", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Equal(t, "10", typed.Message["price"])
	require.Equal(t, "5", typed.Message["quantity"])
	require.Equal(t, "15", typed.Message["triggerPrice"])

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_ModifyOrderByCloid(t *testing.T) {
	price := snx_lib_api_types.Price("10")
	payload := &snx_lib_api_validation.ModifyOrderByCloidActionPayload{
		Action:        "modifyOrder",
		ClientOrderId: "cli-123",
		Price:         &price,
	}
	validated := &snx_lib_api_validation.ValidatedModifyOrderByCloidAction{
		ClientOrderId: "cli-123",
		Payload:       payload,
	}

	typed, err := CreateTradeTypedData("1", 1, 100, "modifyOrder", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "ModifyOrderByCloid", typed.PrimaryType)
	require.Equal(t, "cli-123", typed.Message["clientOrderId"])
	require.Equal(t, "10", typed.Message["price"])
	require.Equal(t, "", typed.Message["quantity"])
	require.Equal(t, "", typed.Message["triggerPrice"])

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_PlaceOrdersNormalization(t *testing.T) {
	payload := &snx_lib_api_validation.PlaceOrdersActionPayload{
		Action: "placeOrders",
		Orders: []snx_lib_api_json.PlaceOrderRequest{
			{
				Symbol:          "ETH-USDT",
				Side:            "buy",
				OrderType:       "limitGtc",
				Price:           Price_None,
				TriggerPrice:    Price_None,
				Quantity:        snx_lib_api_types.Quantity("1"),
				ReduceOnly:      false,
				IsTriggerMarket: false,
				ClientOrderId:   "",
			},
		},
	}
	validated := &snx_lib_api_validation.ValidatedPlaceOrdersAction{Payload: payload}

	typed, err := CreateTradeTypedData("1", 10, 0, "placeOrders", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	grouping, ok := typed.Message["grouping"].(string)
	require.True(t, ok)
	require.Equal(t, "", grouping)

	orders, ok := typed.Message["orders"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, orders, 1)
	order := orders[0]
	require.Equal(t, "ETH-USDT", order["symbol"])
	require.Equal(t, "buy", order["side"])
	require.Equal(t, "limitGtc", order["orderType"])
	require.Equal(t, "", order["price"])
	require.Equal(t, "", order["triggerPrice"])
	require.Equal(t, "1", order["quantity"])
	require.Equal(t, false, order["reduceOnly"])
	require.Equal(t, false, order["isTriggerMarket"])
	require.Equal(t, "", order["clientOrderId"])

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_CancelOrdersNormalization(t *testing.T) {
	payload := &snx_lib_api_validation.CancelOrdersActionPayload{Action: "cancelOrders", VenueOrderIds: []snx_lib_api_types.VenueOrderId{
		snx_lib_api_types.VenueOrderId("1"),
		snx_lib_api_types.VenueOrderId("2"),
		snx_lib_api_types.VenueOrderId("3"),
	}}
	validated := &snx_lib_api_validation.ValidatedCancelOrdersAction{Payload: payload, VenueOrderIds: []snx_lib_api_types.VenueOrderId{
		snx_lib_api_types.VenueOrderId("1"),
		snx_lib_api_types.VenueOrderId("2"),
		snx_lib_api_types.VenueOrderId("3"),
	}}

	typed, err := CreateTradeTypedData("7", 42, 0, "cancelOrders", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "CancelOrders", typed.PrimaryType)

	orderIds, ok := typed.Message["orderIds"].([]any)
	require.True(t, ok)
	require.Len(t, orderIds, 3)

	for idx, expected := range []int64{1, 2, 3} {
		id, ok := orderIds[idx].(*math.HexOrDecimal256)
		require.True(t, ok)
		require.Equal(t, big.NewInt(expected), (*big.Int)(id))
	}

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_CancelOrders_NoFallbackOrderId_Error(t *testing.T) {
	payload := &snx_lib_api_validation.CancelOrdersActionPayload{Action: "cancelOrders"}

	_, err := CreateTradeTypedData("2", 11, 0, "cancelOrders", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
}

func Test_CreateTradeTypedData_CancelOrdersByCloid(t *testing.T) {
	payload := &snx_lib_api_validation.CancelOrdersByCloidActionPayload{
		Action: "cancelOrders",
		ClientOrderIds: []snx_lib_api_types.ClientOrderId{
			"cli-1",
			"cli-2",
		},
	}
	validated := &snx_lib_api_validation.ValidatedCancelOrdersByCloidAction{
		ClientOrderIds: []snx_lib_api_types.ClientOrderId{
			"cli-1",
			"cli-2",
		},
		Payload: payload,
	}

	typed, err := CreateTradeTypedData("7", 42, 0, "cancelOrders", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "CancelOrdersByCloid", typed.PrimaryType)
	require.Equal(t, []string{"cli-1", "cli-2"}, typed.Message["clientOrderIds"])

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_ScheduleCancel(t *testing.T) {
	timeoutSeconds := int64(60)
	payload := &snx_lib_api_validation.ScheduleCancelActionPayload{
		Action:         "scheduleCancel",
		TimeoutSeconds: &timeoutSeconds,
	}
	validated := &snx_lib_api_validation.ValidatedScheduleCancelAction{
		Payload:        payload,
		TimeoutSeconds: timeoutSeconds,
	}

	typed, err := CreateTradeTypedData("7", 42, 0, "scheduleCancel", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "ScheduleCancel", typed.PrimaryType)
	require.Equal(t, "60", typed.Message["timeoutSeconds"])

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_ScheduleCancel_Clear(t *testing.T) {
	timeoutSeconds := int64(0)
	payload := &snx_lib_api_validation.ScheduleCancelActionPayload{
		Action:         "scheduleCancel",
		TimeoutSeconds: &timeoutSeconds,
	}
	validated := &snx_lib_api_validation.ValidatedScheduleCancelAction{
		Payload:        payload,
		TimeoutSeconds: timeoutSeconds,
	}

	typed, err := CreateTradeTypedData("7", 42, 0, "scheduleCancel", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "ScheduleCancel", typed.PrimaryType)
	require.Equal(t, "0", typed.Message["timeoutSeconds"])

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_CancelOrders_RejectsAmbiguousIdentifiers(t *testing.T) {
	payload := map[string]any{
		"action":         "cancelOrders",
		"orderIds":       []any{"1"},
		"clientOrderIds": []any{"cli-1"},
	}

	_, err := CreateTradeTypedData("7", 42, 0, "cancelOrders", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "either orderIds or clientOrderIds")
}

func Test_CreateTradeTypedData_ModifyOrder_RejectsAmbiguousIdentifiers(t *testing.T) {
	payload := map[string]any{
		"action":        "modifyOrder",
		"orderId":       "1",
		"clientOrderId": "cli-1",
		"price":         "10",
	}

	_, err := CreateTradeTypedData("7", 42, 0, "modifyOrder", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "either orderId or clientOrderId")
}

func Test_CreateTradeTypedData_SubAccountAction(t *testing.T) {
	payload := map[string]any{"foo": "bar"}

	// Test new client behavior: nonce=0 -> no nonce in message/types
	typed, err := CreateTradeTypedData("5", 0, 99, "getPortfolio", payload, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "SubAccountAction", typed.PrimaryType)

	// Ensure message contains expected fields
	assertAction, ok := typed.Message["action"].(string)
	require.True(t, ok)
	require.Equal(t, "getPortfolio", assertAction)

	// SubAccountAction should NOT have nonce in message when nonce=0
	_, hasNonce := typed.Message["nonce"]
	require.False(t, hasNonce, "SubAccountAction should not have nonce in message when nonce=0")

	expiresStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "99", expiresStr)

	fields := typed.Types["SubAccountAction"]
	require.Len(t, fields, 3)
	require.Equal(t, "subAccountId", fields[0].Name)
	require.Equal(t, "action", fields[1].Name)
	require.Equal(t, "expiresAfter", fields[2].Name)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

// Backwards compatibility: old clients send nonce with SubAccountAction.
func Test_CreateTradeTypedData_SubAccountAction_WithNonce(t *testing.T) {
	payload := map[string]any{"foo": "bar"}

	// Test old client behavior: nonce > 0 -> nonce IS in message/types
	typed, err := CreateTradeTypedData("5", 12345, 99, "getPortfolio", payload, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "SubAccountAction", typed.PrimaryType)

	// Ensure message contains expected fields
	assertAction, ok := typed.Message["action"].(string)
	require.True(t, ok)
	require.Equal(t, "getPortfolio", assertAction)

	// SubAccountAction SHOULD have nonce in message when nonce > 0 (backwards compatibility)
	nonceStr, hasNonce := typed.Message["nonce"].(string)
	require.True(t, hasNonce, "SubAccountAction should have nonce in message when nonce > 0")
	require.Equal(t, "12345", nonceStr)

	expiresStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "99", expiresStr)

	// Types should include nonce field when nonce > 0
	fields := typed.Types["SubAccountAction"]
	require.Len(t, fields, 4)
	require.Equal(t, "subAccountId", fields[0].Name)
	require.Equal(t, "action", fields[1].Name)
	require.Equal(t, "nonce", fields[2].Name)
	require.Equal(t, "expiresAfter", fields[3].Name)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_WithdrawCollateral(t *testing.T) {
	payload := &snx_lib_api_validation.WithdrawCollateralActionPayload{
		Action:      "withdrawCollateral",
		Symbol:      "USDT",
		Amount:      "1000.50",
		Destination: "0x1234567890123456789012345678901234567890",
	}
	validated := &snx_lib_api_validation.ValidatedWithdrawCollateralAction{Payload: payload}

	typed, err := CreateTradeTypedData("123", 42, 1234567890, "withdrawCollateral", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "WithdrawCollateral", typed.PrimaryType)

	// Ensure message contains expected fields
	subAccountId, ok := typed.Message["subAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "123", subAccountId)

	symbol, ok := typed.Message["symbol"].(string)
	require.True(t, ok)
	require.Equal(t, "USDT", symbol)

	amount, ok := typed.Message["amount"].(string)
	require.True(t, ok)
	require.Equal(t, "1000.50", amount)

	destination, ok := typed.Message["destination"].(string)
	require.True(t, ok)
	require.Equal(t, "0x1234567890123456789012345678901234567890", destination)

	nonceStr, ok := typed.Message["nonce"].(string)
	require.True(t, ok)
	require.Equal(t, "42", nonceStr)

	expiresStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "1234567890", expiresStr)

	// Verify the types are correct
	fields := typed.Types["WithdrawCollateral"]
	require.Len(t, fields, 6)
	require.Equal(t, "subAccountId", fields[0].Name)
	require.Equal(t, "uint256", fields[0].Type)
	require.Equal(t, "symbol", fields[1].Name)
	require.Equal(t, "string", fields[1].Type)
	require.Equal(t, "amount", fields[2].Name)
	require.Equal(t, "string", fields[2].Type)
	require.Equal(t, "destination", fields[3].Name)
	require.Equal(t, "address", fields[3].Type)
	require.Equal(t, "nonce", fields[4].Name)
	require.Equal(t, "uint256", fields[4].Type)
	require.Equal(t, "expiresAfter", fields[5].Name)
	require.Equal(t, "uint256", fields[5].Type)

	// Ensure the message can be hashed
	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_WithdrawCollateral_InvalidPayload(t *testing.T) {
	// Test with invalid payload type
	payload := map[string]any{"foo": "bar"}

	_, err := CreateTradeTypedData("123", 42, 1234567890, "withdrawCollateral", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "withdrawCollateral payload")
}

func Test_CreateTradeTypedData_AddDelegatedSigner(t *testing.T) {
	expiresAt := int64(1735689600000) // Example timestamp in milliseconds
	payload := &snx_lib_api_validation.AddDelegatedSignerActionPayload{
		Action:          "addDelegatedSigner",
		DelegateAddress: "0x1234567890123456789012345678901234567890",
		Permissions:     []string{"trade", "withdraw"},
		ExpiresAt:       &expiresAt,
	}
	validated := &snx_lib_api_validation.ValidatedAddDelegatedSignerAction{Payload: payload}

	typed, err := CreateTradeTypedData("123", 42, 1234567890, "addDelegatedSigner", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "AddDelegatedSigner", typed.PrimaryType)

	// Ensure message contains expected fields
	subAccountId, ok := typed.Message["subAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "123", subAccountId)

	delegateAddress, ok := typed.Message["delegateAddress"].(string)
	require.True(t, ok)
	require.Equal(t, "0x1234567890123456789012345678901234567890", delegateAddress)

	permissions, ok := typed.Message["permissions"].([]string)
	require.True(t, ok)
	require.Equal(t, []string{"trade", "withdraw"}, permissions)

	expiresAtStr, ok := typed.Message["expiresAt"].(string)
	require.True(t, ok)
	require.Equal(t, "1735689600000", expiresAtStr)

	nonceStr, ok := typed.Message["nonce"].(string)
	require.True(t, ok)
	require.Equal(t, "42", nonceStr)

	expiresAfterStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "1234567890", expiresAfterStr)

	// Verify the types are correct
	fields := typed.Types["AddDelegatedSigner"]
	require.Len(t, fields, 6)
	require.Equal(t, "delegateAddress", fields[0].Name)
	require.Equal(t, "address", fields[0].Type)
	require.Equal(t, "subAccountId", fields[1].Name)
	require.Equal(t, "uint256", fields[1].Type)
	require.Equal(t, "nonce", fields[2].Name)
	require.Equal(t, "uint256", fields[2].Type)
	require.Equal(t, "expiresAfter", fields[3].Name)
	require.Equal(t, "uint256", fields[3].Type)
	require.Equal(t, "expiresAt", fields[4].Name)
	require.Equal(t, "uint256", fields[4].Type)
	require.Equal(t, "permissions", fields[5].Name)
	require.Equal(t, "string[]", fields[5].Type)

	// Ensure the message can be hashed
	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_AddDelegatedSigner_NoExpiresAt(t *testing.T) {
	// Test with no expiresAt (should default to 0)
	payload := &snx_lib_api_validation.AddDelegatedSignerActionPayload{
		Action:          "addDelegatedSigner",
		DelegateAddress: "0x1234567890123456789012345678901234567890",
		Permissions:     []string{"trade"},
		ExpiresAt:       nil,
	}
	validated := &snx_lib_api_validation.ValidatedAddDelegatedSignerAction{Payload: payload}

	typed, err := CreateTradeTypedData("123", 42, 1234567890, "addDelegatedSigner", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "AddDelegatedSigner", typed.PrimaryType)

	// expiresAt should default to "0" when not provided
	expiresAtStr, ok := typed.Message["expiresAt"].(string)
	require.True(t, ok)
	require.Equal(t, "0", expiresAtStr)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_AddDelegatedSigner_InvalidPayload(t *testing.T) {
	// Test with completely wrong payload type (not a map)
	payload := "invalid"

	_, err := CreateTradeTypedData("123", 42, 1234567890, "addDelegatedSigner", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "addDelegatedSigner payload")
}

func Test_CreateTradeTypedData_AddDelegatedSigner_EmptyPermissions(t *testing.T) {
	// Test with empty permissions array - should fail validation
	payload := map[string]any{
		"action":        "addDelegatedSigner",
		"walletAddress": "0x1234567890123456789012345678901234567890",
		"permissions":   []string{},
	}

	_, err := CreateTradeTypedData("123", 42, 1234567890, "addDelegatedSigner", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one permission must be specified")
}

func Test_CreateTradeTypedData_AddDelegatedSigner_FromMap(t *testing.T) {
	// Test with map[string]any payload (as it comes from the API)
	expiresAt := int64(1735689600000)
	payload := map[string]any{
		"action":        "addDelegatedSigner",
		"walletAddress": "0x1234567890123456789012345678901234567890",
		"permissions":   []string{"delegate"},
		"expiresAt":     expiresAt,
	}

	typed, err := CreateTradeTypedData("123", 42, 1234567890, "addDelegatedSigner", payload, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "AddDelegatedSigner", typed.PrimaryType)

	delegateAddress, ok := typed.Message["delegateAddress"].(string)
	require.True(t, ok)
	require.Equal(t, "0x1234567890123456789012345678901234567890", delegateAddress)

	permissions, ok := typed.Message["permissions"].([]string)
	require.True(t, ok)
	require.Equal(t, []string{"delegate"}, permissions)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_UpdateLeverage(t *testing.T) {
	payload := &snx_lib_api_validation.UpdateLeverageActionPayload{
		Action:   "updateLeverage",
		Symbol:   "BTC-USDT",
		Leverage: "10",
	}
	validated := &snx_lib_api_validation.ValidatedUpdateLeverageAction{Payload: payload}

	typed, err := CreateTradeTypedData("123", 42, 1234567890, "updateLeverage", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "UpdateLeverage", typed.PrimaryType)

	// Ensure message contains expected fields
	subAccountId, ok := typed.Message["subAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "123", subAccountId)

	symbol, ok := typed.Message["symbol"].(string)
	require.True(t, ok)
	require.Equal(t, "BTC-USDT", symbol)

	leverage, ok := typed.Message["leverage"].(string)
	require.True(t, ok)
	require.Equal(t, "10", leverage)

	nonceStr, ok := typed.Message["nonce"].(string)
	require.True(t, ok)
	require.Equal(t, "42", nonceStr)

	expiresStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "1234567890", expiresStr)

	// Verify the types are correct
	fields := typed.Types["UpdateLeverage"]
	require.Len(t, fields, 5)
	require.Equal(t, "subAccountId", fields[0].Name)
	require.Equal(t, "uint256", fields[0].Type)
	require.Equal(t, "symbol", fields[1].Name)
	require.Equal(t, "string", fields[1].Type)
	require.Equal(t, "leverage", fields[2].Name)
	require.Equal(t, "string", fields[2].Type)
	require.Equal(t, "nonce", fields[3].Name)
	require.Equal(t, "uint256", fields[3].Type)
	require.Equal(t, "expiresAfter", fields[4].Name)
	require.Equal(t, "uint256", fields[4].Type)

	// Ensure the message can be hashed
	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_UpdateLeverage_InvalidPayload(t *testing.T) {
	// Test with invalid payload type
	payload := "invalid"

	_, err := CreateTradeTypedData("123", 42, 1234567890, "updateLeverage", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "updateLeverage payload")
}

func Test_CreateTradeTypedData_UpdateLeverage_EmptySymbol(t *testing.T) {
	// Test with empty symbol - should fail validation
	payload := map[string]any{
		"action":   "updateLeverage",
		"symbol":   "",
		"leverage": "10",
	}

	_, err := CreateTradeTypedData("123", 42, 1234567890, "updateLeverage", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "symbol is required")
}

func Test_CreateTradeTypedData_UpdateLeverage_InvalidLeverage(t *testing.T) {
	// Test with invalid leverage - should fail validation
	payload := map[string]any{
		"action":   "updateLeverage",
		"symbol":   "BTC-USDT",
		"leverage": "0",
	}

	_, err := CreateTradeTypedData("123", 42, 1234567890, "updateLeverage", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "leverage must be a positive integer")
}

func Test_CreateTradeTypedData_UpdateLeverage_FromMap(t *testing.T) {
	// Test with map[string]any payload (as it comes from the API)
	payload := map[string]any{
		"action":   "updateLeverage",
		"symbol":   "ETH-USDT",
		"leverage": "25",
	}

	typed, err := CreateTradeTypedData("456", 100, 9999999999, "updateLeverage", payload, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "UpdateLeverage", typed.PrimaryType)

	symbol, ok := typed.Message["symbol"].(string)
	require.True(t, ok)
	require.Equal(t, "ETH-USDT", symbol)

	leverage, ok := typed.Message["leverage"].(string)
	require.True(t, ok)
	require.Equal(t, "25", leverage)

	subAccountId, ok := typed.Message["subAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "456", subAccountId)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_RemoveDelegatedSigner(t *testing.T) {
	payload := &snx_lib_api_validation.RemoveDelegatedSignerActionPayload{
		Action:          "removeDelegatedSigner",
		DelegateAddress: "0x1234567890123456789012345678901234567890",
	}
	validated := &snx_lib_api_validation.ValidatedRemoveDelegatedSignerAction{Payload: payload}

	typed, err := CreateTradeTypedData("123", 42, 1234567890, "removeDelegatedSigner", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "RemoveDelegatedSigner", typed.PrimaryType)

	// Ensure message contains expected fields
	subAccountId, ok := typed.Message["subAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "123", subAccountId)

	delegateAddress, ok := typed.Message["delegateAddress"].(string)
	require.True(t, ok)
	require.Equal(t, "0x1234567890123456789012345678901234567890", delegateAddress)

	nonceStr, ok := typed.Message["nonce"].(string)
	require.True(t, ok)
	require.Equal(t, "42", nonceStr)

	expiresAfterStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "1234567890", expiresAfterStr)

	// Verify the types are correct
	fields := typed.Types["RemoveDelegatedSigner"]
	require.Len(t, fields, 4)
	require.Equal(t, "delegateAddress", fields[0].Name)
	require.Equal(t, "address", fields[0].Type)
	require.Equal(t, "subAccountId", fields[1].Name)
	require.Equal(t, "uint256", fields[1].Type)
	require.Equal(t, "nonce", fields[2].Name)
	require.Equal(t, "uint256", fields[2].Type)
	require.Equal(t, "expiresAfter", fields[3].Name)
	require.Equal(t, "uint256", fields[3].Type)

	// Ensure the message can be hashed
	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_RemoveDelegatedSigner_InvalidPayload(t *testing.T) {
	// Test with completely wrong payload type (not a map)
	payload := "invalid"

	_, err := CreateTradeTypedData("123", 42, 1234567890, "removeDelegatedSigner", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "removeDelegatedSigner payload")
}

func Test_CreateTradeTypedData_RemoveDelegatedSigner_EmptyDelegate(t *testing.T) {
	// Test with empty delegate address - should fail validation
	payload := map[string]any{
		"action":        "removeDelegatedSigner",
		"walletAddress": "",
	}

	_, err := CreateTradeTypedData("123", 42, 1234567890, "removeDelegatedSigner", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "delegate is required")
}

func Test_CreateTradeTypedData_RemoveDelegatedSigner_InvalidDelegate(t *testing.T) {
	// Test with invalid delegate address - should fail validation
	payload := map[string]any{
		"action":        "removeDelegatedSigner",
		"walletAddress": "not-an-address",
	}

	_, err := CreateTradeTypedData("123", 42, 1234567890, "removeDelegatedSigner", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "delegate address must be a valid Ethereum address")
}

func Test_CreateTradeTypedData_RemoveDelegatedSigner_FromMap(t *testing.T) {
	// Test with map[string]any payload (as it comes from the API)
	payload := map[string]any{
		"action":        "removeDelegatedSigner",
		"walletAddress": "0x1234567890123456789012345678901234567890",
	}

	typed, err := CreateTradeTypedData("123", 42, 1234567890, "removeDelegatedSigner", payload, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "RemoveDelegatedSigner", typed.PrimaryType)

	delegateAddress, ok := typed.Message["delegateAddress"].(string)
	require.True(t, ok)
	require.Equal(t, "0x1234567890123456789012345678901234567890", delegateAddress)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_CreateSubaccount(t *testing.T) {
	payload := &snx_lib_api_validation.CreateSubaccountActionPayload{
		Action: "createSubaccount",
		Name:   "My Trading Account",
	}
	validated := &snx_lib_api_validation.ValidatedCreateSubaccountAction{Payload: payload}

	// Ownership verification uses the master subaccount field.
	typed, err := CreateTradeTypedData("12345", 42, 1234567890, "createSubaccount", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "CreateSubaccount", typed.PrimaryType)

	// Ensure message contains masterSubAccountId (proving ownership)
	masterSubAccountIdStr, ok := typed.Message["masterSubAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "12345", masterSubAccountIdStr)

	// Ensure message contains expected fields
	name, ok := typed.Message["name"].(string)
	require.True(t, ok)
	require.Equal(t, "My Trading Account", name)

	nonceStr, ok := typed.Message["nonce"].(string)
	require.True(t, ok)
	require.Equal(t, "42", nonceStr)

	expiresStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "1234567890", expiresStr)

	// Verify subAccountId is NOT in the message (we use masterSubAccountId instead)
	_, hasSubAccountId := typed.Message["subAccountId"]
	require.False(t, hasSubAccountId, "CreateSubaccount should not have subAccountId in message")

	// Verify the types are correct
	fields := typed.Types["CreateSubaccount"]
	require.Len(t, fields, 4)
	require.Equal(t, "masterSubAccountId", fields[0].Name)
	require.Equal(t, "uint256", fields[0].Type)
	require.Equal(t, "name", fields[1].Name)
	require.Equal(t, "string", fields[1].Type)
	require.Equal(t, "nonce", fields[2].Name)
	require.Equal(t, "uint256", fields[2].Type)
	require.Equal(t, "expiresAfter", fields[3].Name)
	require.Equal(t, "uint256", fields[3].Type)

	// Ensure the message can be hashed
	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_CreateSubaccount_EmptyName(t *testing.T) {
	// Test with empty name - should be valid (name is optional)
	payload := &snx_lib_api_validation.CreateSubaccountActionPayload{
		Action: "createSubaccount",
		Name:   "",
	}
	validated := &snx_lib_api_validation.ValidatedCreateSubaccountAction{Payload: payload}

	typed, err := CreateTradeTypedData("99999", 42, 1234567890, "createSubaccount", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "CreateSubaccount", typed.PrimaryType)

	// Verify masterSubAccountId is present
	masterSubAccountIdStr, ok := typed.Message["masterSubAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "99999", masterSubAccountIdStr)

	name, ok := typed.Message["name"].(string)
	require.True(t, ok)
	require.Equal(t, "", name)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_CreateSubaccount_InvalidPayload(t *testing.T) {
	// Test with completely wrong payload type (not a map)
	payload := "invalid"

	_, err := CreateTradeTypedData("12345", 42, 1234567890, "createSubaccount", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "createSubaccount payload")
}

func Test_CreateTradeTypedData_CreateSubaccount_FromMap(t *testing.T) {
	// Test with map[string]any payload (as it comes from the API)
	// Note: name must be alphanumeric with allowed special chars (no spaces in middle)
	payload := map[string]any{
		"action": "createSubaccount",
		"name":   "API-Trading-Account",
	}

	typed, err := CreateTradeTypedData("77777", 100, 9999999999, "createSubaccount", payload, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "CreateSubaccount", typed.PrimaryType)

	// Verify masterSubAccountId is present
	masterSubAccountIdStr, ok := typed.Message["masterSubAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "77777", masterSubAccountIdStr)

	name, ok := typed.Message["name"].(string)
	require.True(t, ok)
	require.Equal(t, "API-Trading-Account", name)

	// Verify subAccountId is NOT in the message
	_, hasSubAccountId := typed.Message["subAccountId"]
	require.False(t, hasSubAccountId, "CreateSubaccount should not have subAccountId in message")

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}
func Test_CreateTradeTypedData_TransferCollateral(t *testing.T) {
	validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
		To:     1867542890123456790,
		Symbol: "USDT",
		Amount: shopspring_decimal.RequireFromString("1000.50"),
	}

	// subAccountId "1867542890123456789" is the "from" account
	typed, err := CreateTradeTypedData("1867542890123456789", 42, 1234567890, "transferCollateral", validated, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "TransferCollateral", typed.PrimaryType)

	// Ensure message contains expected fields
	subAccountId, ok := typed.Message["subAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "1867542890123456789", subAccountId)

	to, ok := typed.Message["to"].(string)
	require.True(t, ok)
	require.Equal(t, "1867542890123456790", to)

	symbol, ok := typed.Message["symbol"].(string)
	require.True(t, ok)
	require.Equal(t, "USDT", symbol)

	amount, ok := typed.Message["amount"].(string)
	require.True(t, ok)
	require.Equal(t, "1000.5", amount)

	nonceStr, ok := typed.Message["nonce"].(string)
	require.True(t, ok)
	require.Equal(t, "42", nonceStr)

	expiresStr, ok := typed.Message["expiresAfter"].(string)
	require.True(t, ok)
	require.Equal(t, "1234567890", expiresStr)

	// Verify the types are correct (lexicographical order for EIP-712)
	fields := typed.Types["TransferCollateral"]
	require.Len(t, fields, 6)
	require.Equal(t, "amount", fields[0].Name)
	require.Equal(t, "string", fields[0].Type)
	require.Equal(t, "expiresAfter", fields[1].Name)
	require.Equal(t, "uint256", fields[1].Type)
	require.Equal(t, "nonce", fields[2].Name)
	require.Equal(t, "uint256", fields[2].Type)
	require.Equal(t, "subAccountId", fields[3].Name)
	require.Equal(t, "uint256", fields[3].Type)
	require.Equal(t, "symbol", fields[4].Name)
	require.Equal(t, "string", fields[4].Type)
	require.Equal(t, "to", fields[5].Name)
	require.Equal(t, "uint256", fields[5].Type)

	// Ensure the message can be hashed
	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_TransferCollateral_InvalidPayload(t *testing.T) {
	// Test with invalid payload type
	payload := map[string]any{"foo": "bar"}

	_, err := CreateTradeTypedData("123", 42, 1234567890, "transferCollateral", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transferCollateral payload")
}

func Test_CreateTradeTypedData_TransferCollateral_FromMap(t *testing.T) {
	// Test with map[string]any payload (as it comes from the API)
	payload := map[string]any{
		"action": "transferCollateral",
		"to":     "1867542890123456790",
		"symbol": "USDT",
		"amount": "500.25",
	}

	// subAccountId "1867542890123456789" is the "from" account
	typed, err := CreateTradeTypedData("1867542890123456789", 42, 1234567890, "transferCollateral", payload, DefaultDomainName, "1", 1)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, "TransferCollateral", typed.PrimaryType)

	subAccountId, ok := typed.Message["subAccountId"].(string)
	require.True(t, ok)
	require.Equal(t, "1867542890123456789", subAccountId)

	to, ok := typed.Message["to"].(string)
	require.True(t, ok)
	require.Equal(t, "1867542890123456790", to)

	symbol, ok := typed.Message["symbol"].(string)
	require.True(t, ok)
	require.Equal(t, "USDT", symbol)

	amount, ok := typed.Message["amount"].(string)
	require.True(t, ok)
	require.Equal(t, "500.25", amount)

	_, err = GetEIP712MessageHash(typed)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateTradeTypedData_TransferCollateral_InvalidAmount(t *testing.T) {
	// Test with invalid amount - should fail validation
	payload := map[string]any{
		"action": "transferCollateral",
		"to":     "1867542890123456790",
		"symbol": "USDT",
		"amount": "0",
	}

	_, err := CreateTradeTypedData("1867542890123456789", 42, 1234567890, "transferCollateral", payload, DefaultDomainName, "1", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "amount must be a positive value")
}

func Test_ensureValidatedTransferCollateral_Success(t *testing.T) {
	payload := &snx_lib_api_validation.TransferCollateralActionPayload{
		Action: "transferCollateral",
		To:     "1867542890123456790",
		Symbol: "USDT",
		Amount: "1000",
	}

	validated, err := ensureValidatedTransferCollateral(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	require.Equal(t, uint64(1867542890123456790), uint64(validated.To))
	require.Equal(t, "USDT", string(validated.Symbol))
	require.True(t, validated.Amount.Equal(shopspring_decimal.RequireFromString("1000")))
}

func Test_ensureValidatedTransferCollateral_AlreadyValidated(t *testing.T) {
	validated := &snx_lib_api_validation.ValidatedTransferCollateralAction{
		To:     1867542890123456790,
		Symbol: "USDT",
		Amount: shopspring_decimal.RequireFromString("1000"),
	}

	result, err := ensureValidatedTransferCollateral(validated)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Equal(t, validated, result)
}

func Test_ensureValidatedTransferCollateral_FromMap(t *testing.T) {
	payload := map[string]any{
		"action": "transferCollateral",
		"to":     "1867542890123456790",
		"symbol": "USDT",
		"amount": "1000",
	}

	validated, err := ensureValidatedTransferCollateral(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	require.Equal(t, "USDT", string(validated.Symbol))
}

func Test_ensureValidatedTransferCollateral_InvalidType(t *testing.T) {
	payload := "invalid"

	_, err := ensureValidatedTransferCollateral(payload)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported transferCollateral payload type")
}

func Test_ensureValidatedTransferCollateral_NonPointerPayload(t *testing.T) {
	// Test with non-pointer TransferCollateralActionPayload (value type)
	payload := snx_lib_api_validation.TransferCollateralActionPayload{
		Action: "transferCollateral",
		To:     "1867542890123456790",
		Symbol: "USDT",
		Amount: "1000",
	}

	validated, err := ensureValidatedTransferCollateral(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	require.Equal(t, uint64(1867542890123456790), uint64(validated.To))
	require.Equal(t, "USDT", string(validated.Symbol))
	require.True(t, validated.Amount.Equal(shopspring_decimal.RequireFromString("1000")))
}

func Test_ensureValidatedTransferCollateral_NonPointerValidated(t *testing.T) {
	// Test with non-pointer ValidatedTransferCollateralAction (value type)
	validated := snx_lib_api_validation.ValidatedTransferCollateralAction{
		To:     1867542890123456790,
		Symbol: "USDT",
		Amount: shopspring_decimal.RequireFromString("1000"),
	}

	result, err := ensureValidatedTransferCollateral(validated)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, result)
	require.Equal(t, uint64(1867542890123456790), uint64(result.To))
	require.Equal(t, "USDT", string(result.Symbol))
}

func Test_ensureValidatedTransferCollateral_FromMapDecodeError(t *testing.T) {
	// Test with map that will fail validation after decode (missing required fields)
	payload := map[string]any{
		"action": "transferCollateral",
		// missing "to" field
		"symbol": "USDT",
		"amount": "1000",
	}

	_, err := ensureValidatedTransferCollateral(payload)
	require.Error(t, err)
}

func Test_ensureValidatedTransferCollateral_FromMapValidationError(t *testing.T) {
	// Test with map that decodes but fails validation (invalid amount)
	payload := map[string]any{
		"action": "transferCollateral",
		"to":     "1867542890123456790",
		"symbol": "USDT",
		"amount": "-100", // negative amount should fail
	}

	_, err := ensureValidatedTransferCollateral(payload)
	require.Error(t, err)
	require.Contains(t, err.Error(), "amount must be a positive value")
}

func Test_ensureValidatedTransferCollateral_PointerPayloadValidationError(t *testing.T) {
	// Test with pointer payload that fails validation
	payload := &snx_lib_api_validation.TransferCollateralActionPayload{
		Action: "transferCollateral",
		To:     "1867542890123456790",
		Symbol: "USDT",
		Amount: "", // empty amount should fail
	}

	_, err := ensureValidatedTransferCollateral(payload)
	require.Error(t, err)
}

func Test_ensureValidatedTransferCollateral_NonPointerPayloadValidationError(t *testing.T) {
	// Test with non-pointer payload that fails validation
	payload := snx_lib_api_validation.TransferCollateralActionPayload{
		Action: "transferCollateral",
		To:     "", // empty to should fail
		Symbol: "USDT",
		Amount: "100",
	}

	_, err := ensureValidatedTransferCollateral(payload)
	require.Error(t, err)
}

func Test_ensureValidatedTransferCollateral_MapDecodeFailure(t *testing.T) {
	// Test with map that causes mapstructure.Decode to fail
	// This triggers the error return path in DecodeTransferCollateralAction
	payload := map[string]any{
		"action": []int{1, 2, 3}, // slice cannot be decoded to string
		"to":     "1867542890123456790",
		"symbol": "USDT",
		"amount": "1000",
	}

	_, err := ensureValidatedTransferCollateral(payload)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid transferCollateral payload")
}

// --- BuildSignatureHex tests ---

func Test_BuildSignatureHex_WithLowercasePrefix(t *testing.T) {
	sig := TradeSignature{
		R: "0xaabbccdd",
		S: "0x11223344",
		V: 27,
	}
	result := BuildSignatureHex(sig)
	require.Equal(t, "0xaabbccdd112233441b", result)
}

func Test_BuildSignatureHex_WithUppercasePrefix(t *testing.T) {
	sig := TradeSignature{
		R: "0Xaabbccdd",
		S: "0X11223344",
		V: 28,
	}
	result := BuildSignatureHex(sig)
	require.Equal(t, "0xaabbccdd112233441c", result)
}

func Test_BuildSignatureHex_NoPrefix(t *testing.T) {
	sig := TradeSignature{
		R: "aabbccdd",
		S: "11223344",
		V: 27,
	}
	result := BuildSignatureHex(sig)
	require.Equal(t, "0xaabbccdd112233441b", result)
}

func Test_BuildSignatureHex_MixedPrefix(t *testing.T) {
	sig := TradeSignature{
		R: "0xaabb",
		S: "ccdd",
		V: 0,
	}
	result := BuildSignatureHex(sig)
	require.Equal(t, "0xaabbccdd00", result)
}

func Test_BuildSignatureHex_V28(t *testing.T) {
	sig := TradeSignature{
		R: "0xff",
		S: "0xee",
		V: 28,
	}
	result := BuildSignatureHex(sig)
	require.Equal(t, "0xffee1c", result)
}

func Test_BuildSignatureHex_SingleCharR(t *testing.T) {
	// Edge: short R/S values, V=1
	sig := TradeSignature{
		R: "0xa",
		S: "0xb",
		V: 1,
	}
	result := BuildSignatureHex(sig)
	require.Equal(t, "0xab01", result)
}
