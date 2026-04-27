package types

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// =========================================================================
// Types
// =========================================================================

// A composite order identifier, comprising the venue identifier and the
// optional client order identifier.
//
// NOTE: once all the CLOID work has been propagated throughout the
// codebase, this will be renamed to `OrderId`.
type OrderId struct {
	VenueId  VenueOrderId  `json:"venueId"`            // The venue order identifier.
	ClientId ClientOrderId `json:"clientId,omitempty"` // An optional client identifier, which accompanies the venue identifier but has zero functional impact on order handling.
}

// =========================================================================
// Utility functions
// =========================================================================

func OrderIdFromCoreOrderIdUnvalidated(
	orderId snx_lib_core.OrderId,
) OrderId {

	venueOrderId := VenueOrderIdFromCoreVenueOrderIdUnvalidated(orderId.VenueId)
	clientOrderId := ClientOrderIdFromCoreClientOrderIdUnvalidated(orderId.ClientId)

	return OrderId{
		VenueId:  venueOrderId,
		ClientId: clientOrderId,
	}
}

