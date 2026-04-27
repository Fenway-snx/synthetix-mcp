package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

func Test_ValidateModifyOrderAction_SuccessWithPrice(t *testing.T) {

	voids := []string{
		"123",
		" 123",
		"+123 ",
		" +123",
	}

	for _, void := range voids {

		price := Price("10.5")
		action := &ModifyOrderActionPayload{
			Action:       "modifyOrder",
			VenueOrderId: VenueOrderId(void),
			Price:        &price,
			Quantity:     nil,
		}

		orderID, err := ValidateModifyOrderAction(action)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if orderID != VenueOrderId("123") {
			t.Fatalf("expected orderID 123, got '%s'", orderID)
		}
	}
}

func Test_ValidateModifyOrderAction_SuccessWithQuantity(t *testing.T) {
	qty := Quantity("1.25")
	action := &ModifyOrderActionPayload{
		Action:       "modifyOrder",
		VenueOrderId: "42",
		Price:        nil,
		Quantity:     &qty,
	}

	orderID, err := ValidateModifyOrderAction(action)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if orderID != VenueOrderId("42") {
		t.Fatalf("expected orderID 42, got '%s'", orderID)
	}
}

func Test_ValidateModifyOrderAction_SuccessWithTriggerPrice(t *testing.T) {
	triggerPrice := Price("50.5")
	action := &ModifyOrderActionPayload{
		Action:       "modifyOrder",
		VenueOrderId: "99",
		Price:        nil,
		Quantity:     nil,
		TriggerPrice: &triggerPrice,
	}

	orderID, err := ValidateModifyOrderAction(action)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if orderID != VenueOrderId("99") {
		t.Fatalf("expected orderID 99, got '%s'", orderID)
	}
}

func Test_ValidateModifyOrderAction_SuccessWithMultipleFields(t *testing.T) {
	price := Price("100.0")
	qty := Quantity("2.0")
	triggerPrice := Price("95.0")
	action := &ModifyOrderActionPayload{
		Action:       "modifyOrder",
		VenueOrderId: "888",
		Price:        &price,
		Quantity:     &qty,
		TriggerPrice: &triggerPrice,
	}

	orderID, err := ValidateModifyOrderAction(action)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if orderID != VenueOrderId("888") {
		t.Fatalf("expected orderID 888, got '%s'", orderID)
	}
}

func Test_ValidateModifyOrderAction_NilAction(t *testing.T) {
	_, err := ValidateModifyOrderAction(nil)
	assert.Error(t, err)
	assert.Equal(t, snx_lib_core.ErrActionPayloadRequired.Error(), err.Error())
}

func Test_ValidateModifyOrderAction_Errors(t *testing.T) {
	tests := []struct {
		name    string
		action  ModifyOrderActionPayload
		wantErr string
	}{
		{
			name:    "wrong type",
			action:  ModifyOrderActionPayload{Action: "cancelOrder", VenueOrderId: "1"},
			wantErr: "action type must be 'modifyOrder'",
		},
		{
			name:    "empty orderId",
			action:  ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: ""},
			wantErr: "orderId is required",
		},
		{
			name:    "non-numeric orderId",
			action:  ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "abc"},
			wantErr: "orderId must be a valid integer",
		},
		{
			name:    "negative orderId",
			action:  ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "-5"},
			wantErr: "orderId must be a positive integer",
		},
		{
			name:    "no fields to modify",
			action:  ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1"},
			wantErr: snx_lib_core.ErrInvalidModifyOrderPayload.Error(),
		},
		{
			name: "empty price",
			action: func() ModifyOrderActionPayload {
				price := Price_None
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Price: &price}
			}(),
			wantErr: "price must be a valid decimal number",
		},
		{
			name: "invalid price",
			action: func() ModifyOrderActionPayload {
				price := Price("NaN")
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Price: &price}
			}(),
			wantErr: "price must be a valid decimal number",
		},
		{
			name: "non-positive price",
			action: func() ModifyOrderActionPayload {
				price := Price("0")
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Price: &price}
			}(),
			wantErr: "price must be a positive value",
		},
		{
			name: "overlong price",
			action: func() ModifyOrderActionPayload {
				price := Price(strings.Repeat("9", MaxDecimalStringLength+1))
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Price: &price}
			}(),
			wantErr: "price exceeds maximum length of 128 characters",
		},
		{
			name: "empty triggerPrice",
			action: func() ModifyOrderActionPayload {
				triggerPrice := Price_None
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", TriggerPrice: &triggerPrice}
			}(),
			wantErr: "price must be a valid decimal number",
		},
		{
			name: "invalid triggerPrice",
			action: func() ModifyOrderActionPayload {
				triggerPrice := Price("invalid")
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", TriggerPrice: &triggerPrice}
			}(),
			wantErr: "price must be a valid decimal number",
		},
		{
			name: "non-positive triggerPrice",
			action: func() ModifyOrderActionPayload {
				triggerPrice := Price("-100")
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", TriggerPrice: &triggerPrice}
			}(),
			wantErr: "price must be a positive value",
		},
		{
			name: "overlong triggerPrice",
			action: func() ModifyOrderActionPayload {
				triggerPrice := Price(strings.Repeat("9", MaxDecimalStringLength+1))
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", TriggerPrice: &triggerPrice}
			}(),
			wantErr: "triggerPrice exceeds maximum length of 128 characters",
		},
		{
			name: "invalid quantity",
			action: func() ModifyOrderActionPayload {
				qty := Quantity("not-a-number")
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Quantity: &qty}
			}(),
			wantErr: "quantity must be a valid decimal number",
		},
		{
			name: "non-positive quantity",
			action: func() ModifyOrderActionPayload {
				qty := Quantity("0")
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Quantity: &qty}
			}(),
			wantErr: "quantity must be a positive value",
		},
		{
			name: "overlong quantity",
			action: func() ModifyOrderActionPayload {
				qty := Quantity(strings.Repeat("9", MaxDecimalStringLength+1))
				return ModifyOrderActionPayload{Action: "modifyOrder", VenueOrderId: "1", Quantity: &qty}
			}(),
			wantErr: "quantity exceeds maximum length of 128 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateModifyOrderAction(&tc.action)
			assert.Error(t, err)
			assert.Equal(t, tc.wantErr, err.Error(), "expected error containing %q, got %v", tc.wantErr, err)

		})
	}
}

func Test_ValidateModifyOrderBatchAction_SUCCESS_MULTIPLE_ORDERS(t *testing.T) {
	t.Parallel()

	priceOrder1 := Price("10")
	priceOrder2 := Price("20")
	items := []ModifyOrderBatchItem{
		{VenueOrderId: "1", Price: &priceOrder1},
		{VenueOrderId: "2", Price: &priceOrder2},
	}

	got, err := ValidateModifyOrderBatchAction(&ModifyOrderBatchActionPayload{
		Action: "modifyOrderBatch",
		Orders: items,
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 validated items, got %d", len(got))
	}
}

func Test_ValidateModifyOrderBatchAction_LengthValidation(t *testing.T) {
	tests := []struct {
		name    string
		order   ModifyOrderBatchItem
		wantErr string
	}{
		{
			name: "overlong price",
			order: func() ModifyOrderBatchItem {
				price := Price(strings.Repeat("9", MaxDecimalStringLength+1))
				return ModifyOrderBatchItem{VenueOrderId: "1", Price: &price}
			}(),
			wantErr: "order 0: price exceeds maximum length of 128 characters",
		},
		{
			name: "overlong quantity",
			order: func() ModifyOrderBatchItem {
				qty := Quantity(strings.Repeat("9", MaxDecimalStringLength+1))
				return ModifyOrderBatchItem{VenueOrderId: "1", Quantity: &qty}
			}(),
			wantErr: "order 0: quantity exceeds maximum length of 128 characters",
		},
		{
			name: "overlong triggerPrice",
			order: func() ModifyOrderBatchItem {
				triggerPrice := Price(strings.Repeat("9", MaxDecimalStringLength+1))
				return ModifyOrderBatchItem{VenueOrderId: "1", TriggerPrice: &triggerPrice}
			}(),
			wantErr: "order 0: triggerPrice exceeds maximum length of 128 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateModifyOrderBatchAction(&ModifyOrderBatchActionPayload{
				Action: "modifyOrderBatch",
				Orders: []ModifyOrderBatchItem{tc.order},
			})
			assert.Error(t, err)
			assert.Equal(t, tc.wantErr, err.Error())
		})
	}
}
