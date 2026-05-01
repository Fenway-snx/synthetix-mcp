package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

func Test_ValidateCancelOrdersByCloidAction_Success(t *testing.T) {
	action := &CancelOrdersByCloidActionPayload{
		Action: "cancelOrders",
		ClientOrderIds: []ClientOrderId{
			"alpha-1",
			"beta-2",
		},
	}

	clientOrderIds, err := ValidateCancelOrdersByCloidAction(action)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, []ClientOrderId{"alpha-1", "beta-2"}, clientOrderIds)
}

func Test_ValidateCancelOrdersByCloidAction_Error(t *testing.T) {
	tests := []struct {
		name    string
		action  *CancelOrdersByCloidActionPayload
		wantErr string
	}{
		{
			name:    "nil payload",
			action:  nil,
			wantErr: "action payload is required",
		},
		{
			name: "empty client order ids",
			action: &CancelOrdersByCloidActionPayload{
				Action:         "cancelOrders",
				ClientOrderIds: nil,
			},
			wantErr: "orderIds must be nonempty",
		},
		{
			name: "empty client order id element",
			action: &CancelOrdersByCloidActionPayload{
				Action: "cancelOrders",
				ClientOrderIds: []ClientOrderId{
					"",
				},
			},
			wantErr: "clientOrderIds[0]",
		},
		{
			name: "empty client order id after valid id",
			action: &CancelOrdersByCloidActionPayload{
				Action: "cancelOrders",
				ClientOrderIds: []ClientOrderId{
					"alpha-1",
					"",
				},
			},
			wantErr: "clientOrderIds[1]",
		},
		{
			name: "invalid client order id",
			action: &CancelOrdersByCloidActionPayload{
				Action: "cancelOrders",
				ClientOrderIds: []ClientOrderId{
					"invalid cloid",
				},
			},
			wantErr: "clientOrderIds[0]",
		},
		{
			name: "whitespace padded client order id",
			action: &CancelOrdersByCloidActionPayload{
				Action: "cancelOrders",
				ClientOrderIds: []ClientOrderId{
					"  alpha-1  ",
				},
			},
			wantErr: "leading or trailing whitespace",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateCancelOrdersByCloidAction(tc.action)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func Test_ValidateModifyOrderByCloidAction_Success(t *testing.T) {
	action := &ModifyOrderByCloidActionPayload{
		Action:        "modifyOrder",
		ClientOrderId: "alpha-1",
		Price:         snx_lib_utils_test.MakePointerOf(Price("10.5")),
	}

	clientOrderId, err := ValidateModifyOrderByCloidAction(action)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, ClientOrderId("alpha-1"), clientOrderId)
}

func Test_ValidateModifyOrderByCloidAction_Error(t *testing.T) {
	tests := []struct {
		name    string
		action  *ModifyOrderByCloidActionPayload
		wantErr string
	}{
		{
			name:    "nil payload",
			action:  nil,
			wantErr: "action payload is required",
		},
		{
			name: "missing client order id",
			action: &ModifyOrderByCloidActionPayload{
				Action: "modifyOrder",
				Price:  snx_lib_utils_test.MakePointerOf(Price("10")),
			},
			wantErr: "clientOrderId",
		},
		{
			name: "invalid client order id",
			action: &ModifyOrderByCloidActionPayload{
				Action:        "modifyOrder",
				ClientOrderId: "invalid cloid",
			},
			wantErr: "clientOrderId",
		},
		{
			name: "whitespace padded client order id",
			action: &ModifyOrderByCloidActionPayload{
				Action:        "modifyOrder",
				ClientOrderId: "  alpha-1  ",
				Price:         snx_lib_utils_test.MakePointerOf(Price("10")),
			},
			wantErr: "leading or trailing whitespace",
		},
		{
			name: "missing modifications",
			action: &ModifyOrderByCloidActionPayload{
				Action:        "modifyOrder",
				ClientOrderId: "alpha-1",
			},
			wantErr: "please provide at least one of the following: price, quantity, trigger price",
		},
		{
			name: "overlong price",
			action: &ModifyOrderByCloidActionPayload{
				Action:        "modifyOrder",
				ClientOrderId: "alpha-1",
				Price: snx_lib_utils_test.MakePointerOf(Price(
					strings.Repeat("9", MaxDecimalStringLength+1),
				)),
			},
			wantErr: "price exceeds maximum length of 128 characters",
		},
		{
			name: "overlong trigger price",
			action: &ModifyOrderByCloidActionPayload{
				Action:        "modifyOrder",
				ClientOrderId: "alpha-1",
				TriggerPrice: snx_lib_utils_test.MakePointerOf(Price(
					strings.Repeat("9", MaxDecimalStringLength+1),
				)),
			},
			wantErr: "triggerPrice exceeds maximum length of 128 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateModifyOrderByCloidAction(tc.action)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func Test_NewValidatedCancelOrdersByCloidAction_WritesBackSanitizedPayload(t *testing.T) {
	payload := &CancelOrdersByCloidActionPayload{
		Action: "cancelOrders",
		ClientOrderIds: []ClientOrderId{
			"alpha-1",
			"beta-2",
		},
	}

	validated, err := NewValidatedCancelOrdersByCloidAction(payload)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	assert.Equal(t, validated.ClientOrderIds, validated.Payload.ClientOrderIds)
}
