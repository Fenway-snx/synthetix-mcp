package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DecodeVoluntaryAutoExchangeAction_Success(t *testing.T) {
	input := map[string]any{
		"action":           "voluntaryAutoExchange",
		"sourceAsset":      "ETH",
		"targetUSDTAmount": "500.25",
	}

	payload, err := DecodeVoluntaryAutoExchangeAction(input)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, payload)
	assert.Equal(t, "voluntaryAutoExchange", payload.Action)
	assert.Equal(t, "ETH", payload.SourceAsset)
	assert.Equal(t, "500.25", payload.TargetUSDTAmount)
}

func Test_DecodeVoluntaryAutoExchangeAction_NilInput(t *testing.T) {
	payload, err := DecodeVoluntaryAutoExchangeAction(nil)
	assert.Error(t, err)
	assert.Nil(t, payload)
	assert.Equal(t, "action payload is required", err.Error())
}

func Test_DecodeVoluntaryAutoExchangeAction_DecodeError(t *testing.T) {
	input := map[string]any{
		"action":           123,           // wrong type
		"sourceAsset":      []int{1, 2},   // wrong type
		"targetUSDTAmount": map[int]int{}, // wrong type
	}

	payload, err := DecodeVoluntaryAutoExchangeAction(input)
	assert.Error(t, err)
	assert.Nil(t, payload)
	assert.Contains(t, err.Error(), "invalid voluntaryAutoExchange payload")
}

func Test_NewValidatedVoluntaryAutoExchangeAction_Success(t *testing.T) {
	payload := &VoluntaryAutoExchangeActionPayload{
		Action:           "voluntaryAutoExchange",
		SourceAsset:      "ETH",
		TargetUSDTAmount: "1000.50",
	}

	validated, err := NewValidatedVoluntaryAutoExchangeAction(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	assert.Equal(t, payload, validated.Payload)
	assert.Equal(t, "ETH", validated.SourceAsset)
	assert.Equal(t, "1000.50", validated.TargetUSDTAmount)
}

func Test_NewValidatedVoluntaryAutoExchangeAction_AllTarget(t *testing.T) {
	payload := &VoluntaryAutoExchangeActionPayload{
		Action:           "voluntaryAutoExchange",
		SourceAsset:      "ETH",
		TargetUSDTAmount: "all",
	}

	validated, err := NewValidatedVoluntaryAutoExchangeAction(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	assert.Equal(t, "all", validated.TargetUSDTAmount)
}

func Test_NewValidatedVoluntaryAutoExchangeAction_Errors(t *testing.T) {
	tests := []struct {
		name    string
		payload *VoluntaryAutoExchangeActionPayload
		wantErr string
	}{
		{
			name:    "nil payload",
			payload: nil,
			wantErr: "action payload is required",
		},
		{
			name: "wrong action type",
			payload: &VoluntaryAutoExchangeActionPayload{
				Action:           "placeOrders",
				SourceAsset:      "ETH",
				TargetUSDTAmount: "100",
			},
			wantErr: "action type must be 'voluntaryAutoExchange'",
		},
		{
			name: "empty source asset",
			payload: &VoluntaryAutoExchangeActionPayload{
				Action:           "voluntaryAutoExchange",
				SourceAsset:      "",
				TargetUSDTAmount: "100",
			},
			wantErr: "sourceAsset is required",
		},
		{
			name: "source asset is USDT",
			payload: &VoluntaryAutoExchangeActionPayload{
				Action:           "voluntaryAutoExchange",
				SourceAsset:      "USDT",
				TargetUSDTAmount: "100",
			},
			wantErr: "sourceAsset cannot be USDT",
		},
		{
			name: "empty target USDT amount",
			payload: &VoluntaryAutoExchangeActionPayload{
				Action:           "voluntaryAutoExchange",
				SourceAsset:      "ETH",
				TargetUSDTAmount: "",
			},
			wantErr: "targetUSDTAmount is required",
		},
		{
			name: "invalid decimal target amount",
			payload: &VoluntaryAutoExchangeActionPayload{
				Action:           "voluntaryAutoExchange",
				SourceAsset:      "ETH",
				TargetUSDTAmount: "not-a-number",
			},
			wantErr: "targetUSDTAmount must be 'all' or a valid positive decimal",
		},
		{
			name: "zero target amount",
			payload: &VoluntaryAutoExchangeActionPayload{
				Action:           "voluntaryAutoExchange",
				SourceAsset:      "ETH",
				TargetUSDTAmount: "0",
			},
			wantErr: "targetUSDTAmount must be 'all' or a valid positive decimal",
		},
		{
			name: "negative target amount",
			payload: &VoluntaryAutoExchangeActionPayload{
				Action:           "voluntaryAutoExchange",
				SourceAsset:      "ETH",
				TargetUSDTAmount: "-100",
			},
			wantErr: "targetUSDTAmount must be 'all' or a valid positive decimal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			validated, err := NewValidatedVoluntaryAutoExchangeAction(tc.payload)
			assert.Error(t, err)
			assert.Nil(t, validated)
			assert.Equal(t, tc.wantErr, err.Error(), "expected error %q, got %v", tc.wantErr, err)
		})
	}
}
