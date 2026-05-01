package validation

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
)

// Returns a minimal valid order so tests can focus on client IDs.
func minimalValidOrder() snx_lib_api_json.PlaceOrderRequest {
	return snx_lib_api_json.PlaceOrderRequest{
		Symbol:    "BTC-USDT",
		Side:      "buy",
		OrderType: "limitGtc",
		Quantity:  snx_lib_api_types.QuantityFromStringUnvalidated("1.0"),
		Price:     snx_lib_api_types.PriceFromStringUnvalidated("50000"),
	}
}

func Test_ValidatePlaceOrdersAction_ClientOrderId_RejectsWhitespacePaddedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		inputId string
	}{
		{
			name:    "leading whitespace",
			inputId: "  my-cloid-123",
		},
		{
			name:    "trailing whitespace",
			inputId: "my-cloid-123   ",
		},
		{
			name:    "leading and trailing whitespace",
			inputId: "  my-cloid-123  ",
		},
		{
			name:    "whitespace only",
			inputId: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.ClientOrderId = snx_lib_api_json.ClientOrderId(tt.inputId)

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "clientOrderId")
			assert.Contains(t, err.Error(), "leading or trailing whitespace")
		})
	}
}

func Test_ValidatePlaceOrdersAction_ClientOrderId_AllowsCanonicalValues(t *testing.T) {
	t.Parallel()

	order0 := minimalValidOrder()
	order0.ClientOrderId = snx_lib_api_json.ClientOrderId("alpha")

	order1 := minimalValidOrder()
	order1.ClientOrderId = snx_lib_api_json.ClientOrderId("bravo")

	order2 := minimalValidOrder()
	order2.ClientOrderId = snx_lib_api_json.ClientOrderId("charlie")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order0, order1, order2},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, snx_lib_api_json.ClientOrderId("alpha"), action.Orders[0].ClientOrderId)
	assert.Equal(t, snx_lib_api_json.ClientOrderId("bravo"), action.Orders[1].ClientOrderId)
	assert.Equal(t, snx_lib_api_json.ClientOrderId("charlie"), action.Orders[2].ClientOrderId)
}

func Test_ValidatePlaceOrdersAction_ClientOrderId_EmptyIsSkipped(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	// leave ClientOrderId at its zero value ("")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, snx_lib_api_json.ClientOrderId(""), action.Orders[0].ClientOrderId,
		"empty ClientOrderId should remain empty (validation skipped)",
	)
}

func Test_ValidatePlaceOrdersAction_ClientOrderId_TooLong(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.ClientOrderId = snx_lib_api_json.ClientOrderId(strings.Repeat("a", 256))

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientOrderId")
}

func Test_ValidatePlaceOrdersAction_ClientOrderId_InvalidChars(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.ClientOrderId = snx_lib_api_json.ClientOrderId("invalid!@#$%^&*()")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientOrderId")
}

func Test_NewValidatedPlaceOrdersAction_ClientOrderId_RejectsWhitespacePaddedValue(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.ClientOrderId = snx_lib_api_json.ClientOrderId("  normalise-me  ")

	payload := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	validated, err := NewValidatedPlaceOrdersAction(payload)
	require.Error(t, err)
	assert.Nil(t, validated)
	assert.Contains(t, err.Error(), "clientOrderId")
	assert.Contains(t, err.Error(), "leading or trailing whitespace")
}

func Test_ValidatePlaceOrdersAction_RejectsTooManyOrders(t *testing.T) {
	t.Parallel()

	orders := make([]snx_lib_api_json.PlaceOrderRequest, MaxOrdersPerBatch+1)
	for i := range orders {
		orders[i] = minimalValidOrder()
	}

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   orders,
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "orders cannot exceed")
}

func Test_ValidatePlaceOrdersAction_RejectsNonCanonicalSymbols(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.Symbol = "  btc-usdt  "

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbol must use canonical uppercase format")
}

func Test_ValidatePlaceOrdersAction_RejectsReservedInternalClientOrderIdPrefix(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.ClientOrderId = snx_lib_api_json.ClientOrderId("snx-system/chunk-123")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved internal clientOrderId prefix")
}

func Test_ValidatePlaceOrdersAction_RejectsOverlongDecimalString(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.Price = Price(strings.Repeat("9", MaxDecimalStringLength+1))

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "price exceeds maximum length")
}

func Test_ValidatePlaceOrdersAction_RejectsPastExpiresAt(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "limitGtd"
	expiresAt := time.Now().Add(-time.Second).Unix()
	order.ExpiresAt = &expiresAt

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiresAt must be in the future")
}

func Test_ValidatePlaceOrdersAction_AcceptsPositiveQuantity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		quantity string
	}{
		{name: "integer", quantity: "1"},
		{name: "decimal", quantity: "0.5"},
		{name: "small decimal", quantity: "0.001"},
		{name: "large integer", quantity: "10000"},
		{name: "large decimal", quantity: "999.999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.Quantity = Quantity(tt.quantity)

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.NoError(t, err, "quantity %q should be accepted", tt.quantity)
		})
	}
}

func Test_ValidatePlaceOrdersAction_ClosePosition_AllowsZeroQuantity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		quantity string
	}{
		{name: "empty", quantity: ""},
		{name: "literal zero", quantity: "0"},
		{name: "zero point zero", quantity: "0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.OrderType = "triggerTp"
			order.Price = Price("50000")
			order.TriggerPrice = Price("49000")
			order.Quantity = Quantity(tt.quantity)
			order.ClosePosition = true

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.NoError(t, err, "closePosition=true should allow quantity %q", tt.quantity)
		})
	}
}

func Test_ValidatePlaceOrdersAction_RejectsEmptyQuantity(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.Quantity = Quantity_None

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quantity is missing")
}

func Test_ValidatePlaceOrdersAction_RejectsNegativeQuantity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		quantity string
	}{
		{name: "negative integer", quantity: "-1"},
		{name: "negative decimal", quantity: "-0.5"},
		{name: "large negative", quantity: "-100.123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.Quantity = Quantity(tt.quantity)

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err, "quantity %q should be rejected", tt.quantity)
			assert.Contains(t, err.Error(), "quantity is negative")
		})
	}
}

func Test_ValidatePlaceOrdersAction_RejectsNonNumericQuantity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		quantity string
	}{
		{name: "alphabetic", quantity: "abc"},
		{name: "Inf", quantity: "Inf"},
		{name: "mixed alphanumeric", quantity: "12abc"},
		{name: "NaN", quantity: "NaN"},
		{name: "negative infinity", quantity: "-Inf"},
		{name: "positive infinity", quantity: "+Inf"},
		{name: "special characters", quantity: "1.0$"},
		{name: "whitespace", quantity: " 1.0 "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.Quantity = Quantity(tt.quantity)

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err, "quantity %q should be rejected", tt.quantity)
			assert.Contains(t, err.Error(), "quantity is invalid")
		})
	}
}

func Test_ValidatePlaceOrdersAction_RejectsZeroQuantityVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		quantity string
	}{
		{name: "literal zero", quantity: "0"},
		{name: "zero point zero", quantity: "0.0"},
		{name: "zero point double zero", quantity: "0.00"},
		{name: "zero point triple zero", quantity: "0.000"},
		{name: "leading-zero decimal", quantity: "00.0"},
		{name: "point zero", quantity: ".0"},
		{name: "zero with exponent", quantity: "0e0"},
		{name: "zero with positive exponent", quantity: "0E+1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.Quantity = Quantity(tt.quantity)

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err, "quantity %q should be rejected", tt.quantity)
			assert.Contains(t, err.Error(), "quantity is zero")
		})
	}
}

// ---------------------------------------------------------------------------
// Action-level validation
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_AcceptsEmptyActionType(t *testing.T) {
	t.Parallel()

	action := &PlaceOrdersActionPayload{
		Action:   "",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{minimalValidOrder()},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_ValidatePlaceOrdersAction_AcceptsMaxBatchSize(t *testing.T) {
	t.Parallel()

	orders := make([]snx_lib_api_json.PlaceOrderRequest, MaxOrdersPerBatch)
	for i := range orders {
		orders[i] = minimalValidOrder()
	}

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   orders,
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_ValidatePlaceOrdersAction_RejectsEmptyOrdersArray(t *testing.T) {
	t.Parallel()

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Equal(t, ErrOrdersArrayEmpty, err)
}

func Test_ValidatePlaceOrdersAction_RejectsNilPayload(t *testing.T) {
	t.Parallel()

	err := ValidatePlaceOrdersAction(nil)
	require.Error(t, err)
	assert.Equal(t, ErrActionPayloadRequired, err)
}

func Test_ValidatePlaceOrdersAction_RejectsWrongActionType(t *testing.T) {
	t.Parallel()

	action := &PlaceOrdersActionPayload{
		Action:   "cancelOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{minimalValidOrder()},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "placeOrders")
}

// ---------------------------------------------------------------------------
// Side validation
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_AcceptsValidSides(t *testing.T) {
	t.Parallel()

	for _, side := range []string{"buy", "sell"} {
		t.Run(side, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.Side = side

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		})
	}
}

func Test_ValidatePlaceOrdersAction_RejectsInvalidSide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		side string
	}{
		{name: "long", side: "long"},
		{name: "short", side: "short"},
		{name: "uppercase", side: "Buy"},
		{name: "empty", side: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.Side = tt.side

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "side")
		})
	}
}

// ---------------------------------------------------------------------------
// Order type validation
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_AcceptsAllValidOrderTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		orderType        string
		price            Price
		triggerPrice     Price
		isTriggerMarket  bool
		triggerPriceType snx_lib_api_types.TriggerPriceType
	}{
		{name: "limitGtc", orderType: "limitGtc", price: Price("50000")},
		{name: "limitIoc", orderType: "limitIoc", price: Price("50000")},
		{name: "limitAlo", orderType: "limitAlo", price: Price("50000")},
		{name: "market", orderType: "market"},
		{name: "triggerTp limit", orderType: "triggerTp", price: Price("50000"), triggerPrice: Price("49000"), triggerPriceType: "mark"},
		{name: "triggerSl limit", orderType: "triggerSl", price: Price("48000"), triggerPrice: Price("49000"), triggerPriceType: "last"},
		{name: "triggerTp market", orderType: "triggerTp", triggerPrice: Price("49000"), isTriggerMarket: true, triggerPriceType: "mark"},
		{name: "triggerSl market", orderType: "triggerSl", triggerPrice: Price("49000"), isTriggerMarket: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.OrderType = tt.orderType
			order.Price = tt.price
			order.TriggerPrice = tt.triggerPrice
			order.IsTriggerMarket = tt.isTriggerMarket
			order.TriggerPriceType = tt.triggerPriceType

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.NoError(t, err, "orderType %q should be accepted", tt.orderType)
		})
	}
}

func Test_ValidatePlaceOrdersAction_RejectsInvalidOrderType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		orderType string
	}{
		{name: "unknown", orderType: "fooBar"},
		{name: "empty", orderType: ""},
		{name: "uppercase limit", orderType: "LimitGtc"},
		{name: "stop", orderType: "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.OrderType = tt.orderType

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err, "orderType %q should be rejected", tt.orderType)
			assert.Contains(t, err.Error(), "orderType")
		})
	}
}

// ---------------------------------------------------------------------------
// Order type field constraints
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_LimitOrder_RejectsIsTriggerMarket(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "limitGtc"
	order.IsTriggerMarket = true

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "isTriggerMarket must be false")
}

func Test_ValidatePlaceOrdersAction_LimitOrder_RejectsTriggerPrice(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "limitGtc"
	order.TriggerPrice = Price("49000")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "triggerPrice must be empty")
}

func Test_ValidatePlaceOrdersAction_LimitOrder_RequiresPrice(t *testing.T) {
	t.Parallel()

	for _, ot := range []string{"limitGtc", "limitIoc", "limitAlo"} {
		t.Run(ot, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.OrderType = ot
			order.Price = Price_None

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "price is required")
		})
	}
}

func Test_ValidatePlaceOrdersAction_MarketOrder_RejectsPrice(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "market"
	order.Price = Price("50000")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "price must be empty for market orders")
}

func Test_ValidatePlaceOrdersAction_MarketOrder_RejectsTriggerPrice(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "market"
	order.Price = Price_None
	order.TriggerPrice = Price("49000")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "triggerPrice must be empty for market orders")
}

func Test_ValidatePlaceOrdersAction_TriggerLimitOrder_RequiresPrice(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "triggerSl"
	order.Price = Price_None
	order.TriggerPrice = Price("49000")
	order.IsTriggerMarket = false

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "price is required when isTriggerMarket is false")
}

func Test_ValidatePlaceOrdersAction_TriggerMarketOrder_RejectsPrice(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "triggerTp"
	order.Price = Price("50000")
	order.TriggerPrice = Price("49000")
	order.IsTriggerMarket = true

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "price must be empty when isTriggerMarket is true")
}

func Test_ValidatePlaceOrdersAction_TriggerOrder_RequiresTriggerPrice(t *testing.T) {
	t.Parallel()

	for _, ot := range []string{"triggerTp", "triggerSl"} {
		t.Run(ot, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.OrderType = ot
			order.Price = Price("50000")
			order.TriggerPrice = Price_None

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "triggerPrice is required")
		})
	}
}

// ---------------------------------------------------------------------------
// Overlong field rejection (beyond price, which is already tested)
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_RejectsOverlongOrderType(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = strings.Repeat("x", MaxEnumFieldLength+1)

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "orderType exceeds maximum length")
}

func Test_ValidatePlaceOrdersAction_RejectsOverlongQuantity(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.Quantity = Quantity(strings.Repeat("9", MaxDecimalStringLength+1))

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quantity exceeds maximum length")
}

func Test_ValidatePlaceOrdersAction_RejectsOverlongSide(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.Side = strings.Repeat("x", MaxEnumFieldLength+1)

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "side exceeds maximum length")
}

func Test_ValidatePlaceOrdersAction_RejectsOverlongTriggerPrice(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.OrderType = "triggerTp"
	order.TriggerPrice = Price(strings.Repeat("9", MaxDecimalStringLength+1))

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "triggerPrice exceeds maximum length")
}

// ---------------------------------------------------------------------------
// Grouping validation
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_AcceptsValidGrouping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		grouping GroupingValues
	}{
		{name: "na", grouping: GroupingValues_na},
		{name: "normalTpsl", grouping: GroupingValues_normalTpsl},
		{name: "positionTpsl", grouping: GroupingValues_positionsTpsl},
		{name: "empty", grouping: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{minimalValidOrder()},
				Grouping: tt.grouping,
			}

			err := ValidatePlaceOrdersAction(action)
			require.NoError(t, err, "grouping %q should be accepted", tt.grouping)
		})
	}
}

func Test_ValidatePlaceOrdersAction_RejectsInvalidGrouping(t *testing.T) {
	t.Parallel()

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{minimalValidOrder()},
		Grouping: "invalidGroup",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Equal(t, ErrInvalidGrouping, err)
}

// ---------------------------------------------------------------------------
// TriggerPriceType validation (for trigger orders)
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_TriggerOrder_NormalizesValidTriggerPriceType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    snx_lib_api_types.TriggerPriceType
		expected snx_lib_api_types.TriggerPriceType
	}{
		{name: "mark", input: "mark", expected: "mark"},
		{name: "last", input: "last", expected: "last"},
		{name: "empty defaults to mark", input: "", expected: "mark"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.OrderType = "triggerTp"
			order.TriggerPrice = Price("49000")
			order.TriggerPriceType = tt.input

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expected, action.Orders[0].TriggerPriceType)
		})
	}
}

func Test_ValidatePlaceOrdersAction_TriggerOrder_RejectsInvalidTriggerPriceType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input snx_lib_api_types.TriggerPriceType
	}{
		{name: "index", input: "index"},
		{name: "uppercase Mark", input: "Mark"},
		{name: "oracle", input: "oracle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.OrderType = "triggerSl"
			order.TriggerPrice = Price("49000")
			order.TriggerPriceType = tt.input

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unrecognised")
		})
	}
}

// ---------------------------------------------------------------------------
// Fractional values are accepted at this layer
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_AcceptsFractionalPrices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		price Price
	}{
		{name: "one decimal", price: Price("50000.5")},
		{name: "two decimals", price: Price("25000.50")},
		{name: "many decimals", price: Price("1234.56789012")},
		{name: "sub-dollar", price: Price("0.99")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := minimalValidOrder()
			order.Price = tt.price

			action := &PlaceOrdersActionPayload{
				Action:   "placeOrders",
				Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
				Grouping: "na",
			}

			err := ValidatePlaceOrdersAction(action)
			require.NoError(t, err, "fractional price %q should be accepted at API layer", tt.price)
		})
	}
}

// ---------------------------------------------------------------------------
// Multi-order batch happy path
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_AcceptsMultipleValidOrders(t *testing.T) {
	t.Parallel()

	limit := minimalValidOrder()

	market := minimalValidOrder()
	market.OrderType = "market"
	market.Price = Price_None

	trigger := minimalValidOrder()
	trigger.OrderType = "triggerTp"
	trigger.TriggerPrice = Price("49000")

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{limit, market, trigger},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

// ---------------------------------------------------------------------------
// NewValidatedPlaceOrdersAction wrapper
// ---------------------------------------------------------------------------

func Test_NewValidatedPlaceOrdersAction_HappyPath(t *testing.T) {
	t.Parallel()

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{minimalValidOrder()},
		Grouping: "na",
	}

	validated, err := NewValidatedPlaceOrdersAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, validated)
	assert.Equal(t, action, validated.Payload)
}

func Test_NewValidatedPlaceOrdersAction_PropagatesValidationError(t *testing.T) {
	t.Parallel()

	validated, err := NewValidatedPlaceOrdersAction(nil)
	require.Error(t, err)
	assert.Nil(t, validated)
	assert.Equal(t, ErrActionPayloadRequired, err)
}

// ---------------------------------------------------------------------------
// Symbol validation beyond non-canonical
// ---------------------------------------------------------------------------

func Test_ValidatePlaceOrdersAction_RejectsEmptySymbol(t *testing.T) {
	t.Parallel()

	order := minimalValidOrder()
	order.Symbol = ""

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{order},
		Grouping: "na",
	}

	err := ValidatePlaceOrdersAction(action)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symbol")
}

func Test_ValidatePlaceOrdersAction_PayloadSymbolFillsOrdersMissingSymbol(t *testing.T) {
	t.Parallel()

	action := &PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Grouping: GroupingValues_twap,
		Symbol:   "BTC-USDT",
		Orders: []snx_lib_api_json.PlaceOrderRequest{{
			Side:            "buy",
			OrderType:       "twap",
			Quantity:        Quantity("1.0"),
			IntervalSeconds: 30,
			DurationSeconds: 3600,
		}},
	}

	err := ValidatePlaceOrdersAction(action)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, Symbol("BTC-USDT"), action.Orders[0].Symbol)
}
