package validation

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewValidatedTransferCollateralAction_Success(t *testing.T) {
	payload := &TransferCollateralActionPayload{
		Action: "transferCollateral",
		To:     "1867542890123456790",
		Symbol: "USDT",
		Amount: "1000.50",
	}

	validated, err := NewValidatedTransferCollateralAction(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	assert.Equal(t, SubAccountId(1867542890123456790), validated.To)
	assert.Equal(t, Asset("USDT"), validated.Symbol)
	assert.True(t, validated.Amount.Equal(shopspring_decimal.RequireFromString("1000.50")))
}

func Test_NewValidatedTransferCollateralAction_Errors(t *testing.T) {
	tests := []struct {
		name    string
		payload TransferCollateralActionPayload
		wantErr string
	}{
		{
			name: "wrong action type",
			payload: TransferCollateralActionPayload{
				Action: "placeOrders",
				To:     "1867542890123456790",
				Symbol: "USDT",
				Amount: "100",
			},
			wantErr: "action type must be 'transferCollateral'",
		},
		{
			name: "missing to",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "",
				Symbol: "USDT",
				Amount: "100",
			},
			wantErr: "to must be a valid positive integer",
		},
		{
			name: "invalid to format",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "not-a-number",
				Symbol: "USDT",
				Amount: "100",
			},
			wantErr: "to must be a valid positive integer",
		},
		{
			name: "missing symbol",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "1867542890123456790",
				Symbol: "",
				Amount: "100",
			},
			wantErr: "asset name empty",
		},
		{
			name: "invalid symbol format",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "1867542890123456790",
				Symbol: "INVALID-SYMBOL-THAT-IS-TOO-LONG-AND-CONTAINS-INVALID-CHARS-!@#",
				Amount: "100",
			},
			wantErr: "asset name invalid",
		},
		{
			name: "missing amount",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "1867542890123456790",
				Symbol: "USDT",
				Amount: "",
			},
			wantErr: "amount must be a valid decimal number",
		},
		{
			name: "invalid amount format",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "1867542890123456790",
				Symbol: "USDT",
				Amount: "not-a-number",
			},
			wantErr: "amount must be a valid decimal number",
		},
		{
			name: "zero amount",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "1867542890123456790",
				Symbol: "USDT",
				Amount: "0",
			},
			wantErr: "amount must be a positive value",
		},
		{
			name: "negative amount",
			payload: TransferCollateralActionPayload{
				Action: "transferCollateral",
				To:     "1867542890123456790",
				Symbol: "USDT",
				Amount: "-100",
			},
			wantErr: "amount must be a positive value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewValidatedTransferCollateralAction(&tc.payload)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func Test_NewValidatedTransferCollateralAction_NilPayload(t *testing.T) {
	validated, err := NewValidatedTransferCollateralAction(nil)
	assert.Error(t, err)
	assert.Nil(t, validated)
	assert.Equal(t, "action payload is required", err.Error())
}

func Test_DecodeTransferCollateralAction_Success(t *testing.T) {
	input := map[string]any{
		"action": "transferCollateral",
		"to":     "1867542890123456790",
		"symbol": "USDT",
		"amount": "1000.50",
	}

	payload, err := DecodeTransferCollateralAction(input)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, payload)
	assert.Equal(t, "transferCollateral", payload.Action)
	assert.Equal(t, "1867542890123456790", payload.To)
	assert.Equal(t, Asset("USDT"), payload.Symbol)
	assert.Equal(t, "1000.50", payload.Amount)
}

func Test_DecodeTransferCollateralAction_NilInput(t *testing.T) {
	payload, err := DecodeTransferCollateralAction(nil)
	assert.Error(t, err)
	assert.Nil(t, payload)
	assert.Equal(t, "action payload is required", err.Error())
}

func Test_DecodeTransferCollateralAction_InvalidMapType(t *testing.T) {
	// Test with map containing values that can't be decoded to strings
	input := map[string]any{
		"action": []int{1, 2, 3}, // cannot decode slice to string
		"to":     "1867542890123456790",
		"symbol": "USDT",
		"amount": "100",
	}

	payload, err := DecodeTransferCollateralAction(input)
	assert.Error(t, err)
	assert.Nil(t, payload)
	assert.Contains(t, err.Error(), "invalid transferCollateral payload")
}

func Test_DecodeTransferCollateralAction_PartialFields(t *testing.T) {
	// Test with only some fields present - should decode successfully
	// (validation happens in NewValidatedTransferCollateralAction)
	input := map[string]any{
		"to": "1867542890123456790",
	}

	payload, err := DecodeTransferCollateralAction(input)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, payload)
	assert.Equal(t, "1867542890123456790", payload.To)
	assert.Equal(t, "", payload.Action)
	assert.Equal(t, AssetName_None, payload.Symbol)
	assert.Equal(t, "", payload.Amount)
}
