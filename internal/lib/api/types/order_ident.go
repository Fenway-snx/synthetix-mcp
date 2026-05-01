package types

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// =========================================================================
// Types
// =========================================================================

// Composite venue and optional client order identifier.
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
