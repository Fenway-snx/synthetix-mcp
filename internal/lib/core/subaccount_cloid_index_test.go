package core

import "testing"

func Test_NewSubAccount_REBUILDS_CLOID_INDEX(t *testing.T) {
	t.Parallel()

	openOrder := &Order{
		OrderId: OrderId{
			VenueId:  VenueOrderId(101),
			ClientId: ClientOrderId("open-101"),
		},
	}
	conditionalOrder := &Order{
		OrderId: OrderId{
			VenueId:  VenueOrderId(202),
			ClientId: ClientOrderId("conditional-202"),
		},
	}
	emptyCloidOrder := &Order{
		OrderId: OrderId{
			VenueId: VenueOrderId(303),
		},
	}

	subAccount := NewSubAccount(NewSubAccountParams{
		SubAccountId: 7,
		OpenOrders:   map[VenueOrderId]*Order{openOrder.OrderId.VenueId: openOrder},
		ConditionalOrders: map[string]ConditionalOrders{
			"BTC-USD": {conditionalOrder, emptyCloidOrder},
		},
	})

	if venueOrderId, ok := subAccount.LookupByCloid(openOrder.OrderId.ClientId); !ok || venueOrderId != openOrder.OrderId.VenueId {
		t.Fatalf("expected open order cloid to resolve to %d, got %d, found=%t", openOrder.OrderId.VenueId, venueOrderId, ok)
	}
	if venueOrderId, ok := subAccount.LookupByCloid(conditionalOrder.OrderId.ClientId); !ok || venueOrderId != conditionalOrder.OrderId.VenueId {
		t.Fatalf("expected conditional order cloid to resolve to %d, got %d, found=%t", conditionalOrder.OrderId.VenueId, venueOrderId, ok)
	}
	if _, ok := subAccount.LookupByCloid(ClientOrderId_Empty); ok {
		t.Fatal("expected empty cloid to be ignored by the index")
	}
}

func Test_CloidIndexAddRemoveAndRebuild(t *testing.T) {
	t.Parallel()

	subAccount := NewSubAccount(NewSubAccountParams{SubAccountId: 9})

	subAccount.AddCloidMapping(ClientOrderId("cli-1"), VenueOrderId(1))
	if venueOrderId, ok := subAccount.LookupByCloid(ClientOrderId("cli-1")); !ok || venueOrderId != VenueOrderId(1) {
		t.Fatalf("expected added cloid to resolve to 1, got %d, found=%t", venueOrderId, ok)
	}

	subAccount.RemoveCloidMapping(ClientOrderId("cli-1"))
	if _, ok := subAccount.LookupByCloid(ClientOrderId("cli-1")); ok {
		t.Fatal("expected removed cloid mapping to disappear")
	}

	subAccount.OpenOrders[VenueOrderId(11)] = &Order{
		OrderId: OrderId{
			VenueId:  VenueOrderId(11),
			ClientId: ClientOrderId("cli-11"),
		},
	}
	subAccount.ConditionalOrders["ETH-USD"] = ConditionalOrders{
		&Order{
			OrderId: OrderId{
				VenueId:  VenueOrderId(12),
				ClientId: ClientOrderId("cli-12"),
			},
		},
	}

	subAccount.RebuildCloidIndex()

	if venueOrderId, ok := subAccount.LookupByCloid(ClientOrderId("cli-11")); !ok || venueOrderId != VenueOrderId(11) {
		t.Fatalf("expected rebuilt open order cloid to resolve to 11, got %d, found=%t", venueOrderId, ok)
	}
	if venueOrderId, ok := subAccount.LookupByCloid(ClientOrderId("cli-12")); !ok || venueOrderId != VenueOrderId(12) {
		t.Fatalf("expected rebuilt conditional order cloid to resolve to 12, got %d, found=%t", venueOrderId, ok)
	}
}

func Test_CloidIndexRejectsZeroVenueOrderId(t *testing.T) {
	t.Parallel()

	subAccount := NewSubAccount(NewSubAccountParams{SubAccountId: 10})

	subAccount.AddCloidMapping(ClientOrderId("cli-zero"), VenueOrderId_Zero)

	_, ok := subAccount.LookupByCloid(ClientOrderId("cli-zero"))
	if ok {
		t.Fatal("expected zero venue order id mapping to be silently rejected")
	}

	if subAccount.HasActiveCloid(ClientOrderId("cli-zero")) {
		t.Fatal("expected no active cloid for zero venue order id")
	}
}

func Test_CloidIndexRebuildLeavesHistoricalDuplicatesUnambiguous(t *testing.T) {
	t.Parallel()

	subAccount := NewSubAccount(NewSubAccountParams{
		SubAccountId: 11,
		OpenOrders: map[VenueOrderId]*Order{
			VenueOrderId(21): {
				OrderId: OrderId{
					VenueId:  VenueOrderId(21),
					ClientId: ClientOrderId("dup-cli"),
				},
			},
		},
		ConditionalOrders: map[string]ConditionalOrders{
			"BTC-USD": {
				&Order{
					OrderId: OrderId{
						VenueId:  VenueOrderId(22),
						ClientId: ClientOrderId("dup-cli"),
					},
				},
			},
		},
	})

	if !subAccount.HasActiveCloid(ClientOrderId("dup-cli")) {
		t.Fatal("expected duplicate cloid to still count as active")
	}
	if _, ok := subAccount.LookupByCloid(ClientOrderId("dup-cli")); ok {
		t.Fatal("expected duplicate cloid lookup to remain unresolved")
	}

	delete(subAccount.OpenOrders, VenueOrderId(21))
	subAccount.RemoveCloidMapping(ClientOrderId("dup-cli"))

	if !subAccount.HasActiveCloid(ClientOrderId("dup-cli")) {
		t.Fatal("expected remaining duplicate to still count as active")
	}

	// In the degraded state (refCount==1, venueOrderId==0), LookupByCloid
	// must return false rather than returning VenueOrderId_Zero as a match.
	if _, ok := subAccount.LookupByCloid(ClientOrderId("dup-cli")); ok {
		t.Fatal("expected degraded-state cloid lookup to return false before rebuild")
	}

	subAccount.RebuildCloidIndex()

	venueOrderId, ok := subAccount.LookupByCloid(ClientOrderId("dup-cli"))
	if !ok || venueOrderId != VenueOrderId(22) {
		t.Fatalf("expected remaining duplicate to resolve to 22 after rebuild, got %d, found=%t", venueOrderId, ok)
	}
}
