package validation

import (
	"fmt"
	"testing"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

// Returns a full batch of valid orders for benchmarks.
func benchPlaceOrdersFullBatch() []snx_lib_api_json.PlaceOrderRequest {
	base := minimalValidOrder()
	orders := make([]snx_lib_api_json.PlaceOrderRequest, MaxOrdersPerBatch)
	for i := range orders {
		orders[i] = base
	}
	return orders
}

// Benchmarks eager labels, deferred labels, and static labels.
func Benchmark_ValidatePlaceOrdersAction_orderLoopStringMaxLength(b *testing.B) {
	orders := benchPlaceOrdersFullBatch()

	b.Run("DYNAMIC_SPRINTF", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for j, order := range orders {
				if err := ValidateStringMaxLength(order.Side, MaxEnumFieldLength, fmt.Sprintf("order %d: %s", j, API_WKS_side)); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.OrderType, MaxEnumFieldLength, fmt.Sprintf("order %d: orderType", j)); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.Price, MaxDecimalStringLength, fmt.Sprintf("order %d: price", j)); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.Quantity, MaxDecimalStringLength, fmt.Sprintf("order %d: quantity", j)); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.TriggerPrice, MaxDecimalStringLength, fmt.Sprintf("order %d: triggerPrice", j)); err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("DEFERRED_FUNC_THUNK", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for j, order := range orders {
				if err := ValidateStringMaxLength(order.Side, MaxEnumFieldLength, func() string {
					return fmt.Sprintf("order %d: %s", j, API_WKS_side)
				}); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.OrderType, MaxEnumFieldLength, func() string {
					return fmt.Sprintf("order %d: orderType", j)
				}); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.Price, MaxDecimalStringLength, func() string {
					return fmt.Sprintf("order %d: price", j)
				}); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.Quantity, MaxDecimalStringLength, func() string {
					return fmt.Sprintf("order %d: quantity", j)
				}); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.TriggerPrice, MaxDecimalStringLength, func() string {
					return fmt.Sprintf("order %d: triggerPrice", j)
				}); err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("STATIC_FIELD_NAMES", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, order := range orders {
				if err := ValidateStringMaxLength(order.Side, MaxEnumFieldLength, "side"); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.OrderType, MaxEnumFieldLength, "orderType"); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.Price, MaxDecimalStringLength, "price"); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.Quantity, MaxDecimalStringLength, "quantity"); err != nil {
					b.Fatal(err)
				}
				if err := ValidateStringMaxLength(order.TriggerPrice, MaxDecimalStringLength, "triggerPrice"); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}
