package core

// cloidEntry tracks one or more active orders that share a CLOID.
// When exactly one order owns the CLOID (the overwhelmingly common case),
// venueOrderId is valid and refCount == 1, so LookupByCloid can return it
// directly with no heap allocation.
// When the mapping is ambiguous or degraded, venueOrderId is set to
// VenueOrderId_Zero as a sentinel and LookupByCloid returns
// (VenueOrderId_Zero, false). This happens when multiple active orders share
// a CLOID (refCount > 1), and also after removals that reduce refCount back to
// 1 before a rebuild can determine which concrete order remains.
type cloidEntry struct {
	venueOrderId VenueOrderId
	refCount     int
}

// Adds or updates a cloid-to-venue mapping for active orders.
// Normal runtime mutations should maintain the index incrementally via this
// helper and `RemoveCloidMapping()`.
func (s *SubAccount) AddCloidMapping(cloid ClientOrderId, venueOrderId VenueOrderId) {
	if cloid == ClientOrderId_Empty || venueOrderId == VenueOrderId_Zero {
		return
	}

	entry, exists := s.cloidIndex[cloid]
	if !exists {
		s.cloidIndex[cloid] = cloidEntry{venueOrderId: venueOrderId, refCount: 1}
		return
	}

	if entry.refCount == 1 && entry.venueOrderId == venueOrderId {
		return
	}

	entry.refCount++
	entry.venueOrderId = 0
	s.cloidIndex[cloid] = entry
}

// Clears a cloid mapping when an order leaves the active set.
func (s *SubAccount) RemoveCloidMapping(cloid ClientOrderId) {
	if cloid == ClientOrderId_Empty {
		return
	}

	entry, exists := s.cloidIndex[cloid]
	if !exists {
		return
	}

	if entry.refCount <= 1 {
		delete(s.cloidIndex, cloid)
		return
	}

	entry.refCount--
	if entry.refCount == 1 {
		// Cannot determine the remaining VenueOrderId without scanning;
		// mark as needing a rebuild to restore the fast-path lookup.
		entry.venueOrderId = 0
	}
	s.cloidIndex[cloid] = entry
}

// Reports whether any active order currently owns the CLOID.
func (s *SubAccount) HasActiveCloid(cloid ClientOrderId) bool {
	if cloid == ClientOrderId_Empty {
		return false
	}

	entry, exists := s.cloidIndex[cloid]
	return exists && entry.refCount > 0
}

// Resolves an active order by cloid when the mapping is unambiguous.
func (s *SubAccount) LookupByCloid(cloid ClientOrderId) (VenueOrderId, bool) {
	if cloid == ClientOrderId_Empty {
		return VenueOrderId_Zero, false
	}

	entry, ok := s.cloidIndex[cloid]
	if !ok || entry.refCount != 1 || entry.venueOrderId == VenueOrderId_Zero {
		return VenueOrderId_Zero, false
	}

	return entry.venueOrderId, true
}

// Rebuilds the active-order cloid index from the current subaccount state.
// Use this only after bulk state construction or replacement, such as
// subaccount hydration from an existing snapshot or full in-memory resets.
// Ordinary order lifecycle mutations should keep the index in sync with
// `AddCloidMapping()` / `RemoveCloidMapping()` instead of relying on rebuilds.
func (s *SubAccount) RebuildCloidIndex() {
	s.cloidIndex = make(map[ClientOrderId]cloidEntry)

	for venueOrderId, order := range s.OpenOrders {
		if order == nil {
			continue
		}

		s.AddCloidMapping(order.OrderId.ClientId, venueOrderId)
	}

	for _, orders := range s.ConditionalOrders {
		for _, order := range orders {
			if order == nil {
				continue
			}

			s.AddCloidMapping(order.OrderId.ClientId, order.OrderId.VenueId)
		}
	}
}
