package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ConditionalOrders(t *testing.T) {
	t.Run("Basic created, update, and remove", func(t *testing.T) {
		var cOrders ConditionalOrders

		for i := range 5 {
			order := Order{
				OrderId: OrderId{
					VenueId:  VenueOrderId(i + 1),
					ClientId: ClientOrderId(fmt.Sprintf("cli-%d", i+1)),
				},
				Type:      OrderTypeStopMarket,
				Direction: Direction_CloseLong,
			}
			err := cOrders.InsertConditionalOrder(&order)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		}

		assert.Equal(t, 5, len(cOrders))

		order, found := cOrders.FindOrderByVenueId(1)
		assert.True(t, found)
		assert.EqualValues(t, 1, order.OrderId.VenueId)

		// invalid order id
		wrong, found := cOrders.FindOrderByVenueId(1000)
		assert.False(t, found)
		assert.Nil(t, wrong)

		// remove order 1
		removed := cOrders.RemoveOrderByVenueId(1)
		assert.True(t, removed)

		// try to get it
		order, found = cOrders.FindOrderByVenueId(1)
		assert.False(t, found)
		assert.Nil(t, order)

		// try to remove the same order again
		removed_t2 := cOrders.RemoveOrderByVenueId(1)
		assert.False(t, removed_t2)

		// get an order and update it
		anotherOne, found := cOrders.FindOrderByVenueId(2)
		assert.True(t, found)
		assert.EqualValues(t, 2, anotherOne.OrderId.VenueId)

		anotherOne.OrderId.VenueId = 1002

		// try to get the order again but it should fail
		order, found = cOrders.FindOrderByVenueId(2)
		assert.False(t, found)
		assert.Nil(t, order)

		// try to get the order with the new id
		order, found = cOrders.FindOrderByVenueId(1002)
		assert.True(t, found)
		assert.EqualValues(t, 1002, order.OrderId.VenueId)
	})
}

func Test_PendingSideFor(t *testing.T) {
	tests := []struct {
		name      string
		orderType OrderType
		direction Direction
		expected  ConditionalTriggerSide
	}{
		// Stop orders — sell-side directions trigger below
		{"stop market CloseLong", OrderTypeStopMarket, Direction_CloseLong, ConditionalTriggerBelow},
		{"stop limit CloseLong", OrderTypeStopLimit, Direction_CloseLong, ConditionalTriggerBelow},
		{"stop market Short (standalone)", OrderTypeStopMarket, Direction_Short, ConditionalTriggerBelow},
		{"stop limit Short (standalone)", OrderTypeStopLimit, Direction_Short, ConditionalTriggerBelow},

		// Stop orders — buy-side directions trigger above
		{"stop market CloseShort", OrderTypeStopMarket, Direction_CloseShort, ConditionalTriggerAbove},
		{"stop limit CloseShort", OrderTypeStopLimit, Direction_CloseShort, ConditionalTriggerAbove},
		{"stop market Long (standalone)", OrderTypeStopMarket, Direction_Long, ConditionalTriggerAbove},
		{"stop limit Long (standalone)", OrderTypeStopLimit, Direction_Long, ConditionalTriggerAbove},

		// Take profit orders — sell-side directions trigger above
		{"tp market CloseLong", OrderTypeTakeProfitMarket, Direction_CloseLong, ConditionalTriggerAbove},
		{"tp limit CloseLong", OrderTypeTakeProfitLimit, Direction_CloseLong, ConditionalTriggerAbove},
		{"tp market Short (standalone)", OrderTypeTakeProfitMarket, Direction_Short, ConditionalTriggerAbove},
		{"tp limit Short (standalone)", OrderTypeTakeProfitLimit, Direction_Short, ConditionalTriggerAbove},

		// Take profit orders — buy-side directions trigger below
		{"tp market CloseShort", OrderTypeTakeProfitMarket, Direction_CloseShort, ConditionalTriggerBelow},
		{"tp limit CloseShort", OrderTypeTakeProfitLimit, Direction_CloseShort, ConditionalTriggerBelow},
		{"tp market Long (standalone)", OrderTypeTakeProfitMarket, Direction_Long, ConditionalTriggerBelow},
		{"tp limit Long (standalone)", OrderTypeTakeProfitLimit, Direction_Long, ConditionalTriggerBelow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{Type: tt.orderType, Direction: tt.direction}
			side, err := PendingSideFor(order)
			require.NoError(t, err, "PendingSideFor should not error for %s", tt.name)
			assert.Equal(t, tt.expected, side)
		})
	}

	t.Run("invalid order type returns error", func(t *testing.T) {
		order := &Order{Type: OrderTypeMarket, Direction: Direction_Long}
		_, err := PendingSideFor(order)
		assert.Error(t, err)
	})
}
