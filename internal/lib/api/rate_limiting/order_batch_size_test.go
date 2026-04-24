package ratelimiting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_OrderBatchSize(t *testing.T) {
	t.Parallel()

	t.Run("non_place_orders_returns_one", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 1, OrderBatchSize("cancelOrders", map[string]any{
			"orders": []any{"a"},
		}))
	})

	t.Run("place_orders_missing_orders_returns_one", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 1, OrderBatchSize("placeOrders", map[string]any{}))
	})

	t.Run("place_orders_empty_orders_returns_one", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 1, OrderBatchSize("placeOrders", map[string]any{
			"orders": []any{},
		}))
	})

	t.Run("place_orders_wrong_type_returns_one", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 1, OrderBatchSize("placeOrders", map[string]any{
			"orders": "not-a-slice",
		}))
	})

	t.Run("place_orders_counts_entries", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 3, OrderBatchSize("placeOrders", map[string]any{
			"orders": []any{map[string]any{}, map[string]any{}, map[string]any{}},
		}))
	})
}
