package core

import (
	"fmt"
	"slices"

	shopspring_decimal "github.com/shopspring/decimal"
)

type ConditionalOrders []*Order

// We probably won't have too many conditional orders
func (p *ConditionalOrders) InsertConditionalOrder(order *Order) error {
	if order.TriggerSide == nil {
		side, err := PendingSideFor(order)
		if err != nil {
			return err
		}
		order.TriggerSide = &side
	}

	// TODO: We need to check if this order already exists in the slice by ID
	*p = append(*p, order)
	return nil
}

func (p ConditionalOrders) FindOrderByVenueId(venueOrderId VenueOrderId) (*Order, bool) {

	for _, order := range p {
		if order.OrderId.VenueId == venueOrderId {
			return order, true
		}
	}

	return nil, false
}

// Returns true if an order was removed, false if not found
func (p *ConditionalOrders) RemoveOrderByVenueId(venueOrderId VenueOrderId) bool {
	for i, order := range *p {
		if order.OrderId.VenueId == venueOrderId {
			*p = slices.Delete(*p, i, i+1)
			return true
		}
	}
	return false
}

func (o *Order) ShouldTrigger(currentPrice shopspring_decimal.Decimal) bool {
	if o.TriggerSide == nil {
		return false
	}

	switch *o.TriggerSide {
	case ConditionalTriggerAbove:
		return currentPrice.GreaterThanOrEqual(o.TriggerPrice)
	case ConditionalTriggerBelow:
		return currentPrice.LessThanOrEqual(o.TriggerPrice)
	default:
		return false
	}
}

type ConditionalTriggerSide int

const (
	ConditionalTriggerAbove ConditionalTriggerSide = iota
	ConditionalTriggerBelow
)

// Determines which side of the current price triggers a conditional order.
// Sell-side directions (CloseLong, Short) and buy-side directions
// (CloseShort, Long) share trigger semantics regardless of whether the
// order is reducing an existing position or opening a new one.
func PendingSideFor(o *Order) (ConditionalTriggerSide, error) {
	isSellSide := o.Direction == Direction_CloseLong || o.Direction == Direction_Short

	switch o.Type {
	case OrderTypeStopMarket, OrderTypeStopLimit:
		if isSellSide {
			return ConditionalTriggerBelow, nil
		}
		return ConditionalTriggerAbove, nil

	case OrderTypeTakeProfitMarket, OrderTypeTakeProfitLimit:
		if isSellSide {
			return ConditionalTriggerAbove, nil
		}
		return ConditionalTriggerBelow, nil
	}
	return -1, fmt.Errorf("invalid order type %d", o.Type)
}
