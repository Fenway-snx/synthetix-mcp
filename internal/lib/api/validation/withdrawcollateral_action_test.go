package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateWithdrawCollateralAction_Success(t *testing.T) {
	action := &WithdrawCollateralActionPayload{
		Action:      "withdrawCollateral",
		Symbol:      "USDT",
		Amount:      "1000.50",
		Destination: "0x1234567890123456789012345678901234567890",
	}

	err := ValidateWithdrawCollateralAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_ValidateWithdrawCollateralAction_EmptyAction(t *testing.T) {
	action := &WithdrawCollateralActionPayload{
		Action:      "",
		Symbol:      "USDT",
		Amount:      "100",
		Destination: "0x1234567890123456789012345678901234567890",
	}

	err := ValidateWithdrawCollateralAction(action)
	require.NoError(t, err, "empty action should be allowed")
}

func Test_ValidateWithdrawCollateralAction_Errors(t *testing.T) {
	tests := []struct {
		name    string
		action  WithdrawCollateralActionPayload
		wantErr string
	}{
		{
			name: "wrong action type",
			action: WithdrawCollateralActionPayload{
				Action:      "placeOrders",
				Symbol:      "USDT",
				Amount:      "100",
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "action type must be 'withdrawCollateral'",
		},
		{
			name: "missing symbol",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "",
				Amount:      "100",
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "symbol is required",
		},
		{
			name: "non-canonical symbol whitespace",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      " USDT ",
				Amount:      "100",
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "symbol must use canonical uppercase format",
		},
		{
			name: "missing amount",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "",
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "amount must be a positive value",
		},
		{
			name: "invalid amount format",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "not-a-number",
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "amount must be a valid decimal number",
		},
		{
			name: "amount too long",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      strings.Repeat("9", MaxDecimalStringLength+1),
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "amount exceeds maximum length of 128 characters",
		},
		{
			name: "zero amount",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "0",
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "amount must be a positive value",
		},
		{
			name: "negative amount",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "-100",
				Destination: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "amount must be a positive value",
		},
		{
			name: "missing destination",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "100",
				Destination: "",
			},
			wantErr: "destination address is required",
		},
		{
			name: "destination with surrounding whitespace",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "100",
				Destination: " 0x1234567890123456789012345678901234567890 ",
			},
			wantErr: "destination must not include leading or trailing whitespace",
		},
		{
			name: "invalid destination format - no 0x prefix",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "100",
				Destination: "1234567890123456789012345678901234567890",
			},
			wantErr: "destination must be a valid Ethereum address",
		},
		{
			name: "invalid destination format - wrong length",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "100",
				Destination: "0x1234",
			},
			wantErr: "destination must be a valid Ethereum address",
		},
		{
			name: "invalid destination format - invalid hex",
			action: WithdrawCollateralActionPayload{
				Action:      "withdrawCollateral",
				Symbol:      "USDT",
				Amount:      "100",
				Destination: "0xGGGG567890123456789012345678901234567890",
			},
			wantErr: "destination must be a valid Ethereum address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWithdrawCollateralAction(&tc.action)
			assert.Error(t, err)
			assert.Equal(t, tc.wantErr, err.Error(), "expected error %q, got %v", tc.wantErr, err)
		})
	}
}

func Test_DecodeWithdrawCollateralAction_Success(t *testing.T) {
	input := map[string]any{
		"action":      "withdrawCollateral",
		"symbol":      "USDT",
		"amount":      "1000.50",
		"destination": "0x1234567890123456789012345678901234567890",
	}

	payload, err := DecodeWithdrawCollateralAction(input)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, payload)
	assert.Equal(t, "withdrawCollateral", string(payload.Action))
	assert.Equal(t, Asset("USDT"), payload.Symbol)
	assert.Equal(t, "1000.50", payload.Amount)
	assert.Equal(t, WalletAddress("0x1234567890123456789012345678901234567890"), payload.Destination)
}

func Test_DecodeWithdrawCollateralAction_NilInput(t *testing.T) {
	payload, err := DecodeWithdrawCollateralAction(nil)
	assert.Error(t, err)
	assert.Nil(t, payload)
	assert.Equal(t, "action payload is required", err.Error())
}

func Test_NewValidatedWithdrawCollateralAction_Success(t *testing.T) {
	payload := &WithdrawCollateralActionPayload{
		Action:      "withdrawCollateral",
		Symbol:      "USDT",
		Amount:      "1000.50",
		Destination: "0x1234567890123456789012345678901234567890",
	}

	validated, err := NewValidatedWithdrawCollateralAction(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	assert.Equal(t, payload, validated.Payload)
}

func Test_NewValidatedWithdrawCollateralAction_ValidationError(t *testing.T) {
	payload := &WithdrawCollateralActionPayload{
		Action:      "withdrawCollateral",
		Symbol:      "",
		Amount:      "1000.50",
		Destination: "0x1234567890123456789012345678901234567890",
	}

	validated, err := NewValidatedWithdrawCollateralAction(payload)
	assert.Error(t, err)
	assert.Nil(t, validated)
	assert.Equal(t, "symbol is required", err.Error())
}
