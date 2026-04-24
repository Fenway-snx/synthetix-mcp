package types

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// =========================================================================
// Types
// =========================================================================

// A composite order identifier, comprising the system-generated venue
// identifier and the optional client order identifier.
//
// NOTE: once all the CLOID work has been propagated throughout the
// codebase, this will be renamed to `OrderId`.
type OrderId struct {
	VenueId  VenueOrderId  `json:"venueId"`            // The internal order identifier, which is entirely definitive.
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

func OrderIdFromGRPCOrderIdUnvalidated(
	orderId *v4grpc.OrderId,
) (r OrderId) {

	// NOTE: ideally we should not need this defensive programming action, but
	// given the scope of CLOID changes we are taking this caution.
	if orderId == nil || orderId.VenueId == 0 {
		return
	}

	venueOrderId := VenueOrderIdFromUintUnvalidated(orderId.VenueId)
	clientOrderId := ClientOrderIdFromStringUnvalidated(orderId.ClientId)

	r = OrderId{
		VenueId:  venueOrderId,
		ClientId: clientOrderId,
	}

	return
}

func OrderIdPtrOrNilFromGRPCOrderIdUnvalidated(
	orderId *v4grpc.OrderId,
) (r *OrderId) {

	if orderId == nil {
		return
	}

	// TODO: determine whether we can dispense with this check
	if orderId.VenueId == 0 {
		return
	}

	venueOrderId := VenueOrderIdFromUintUnvalidated(orderId.VenueId)
	clientOrderId := ClientOrderIdFromStringUnvalidated(orderId.ClientId)

	r = &OrderId{
		VenueId:  venueOrderId,
		ClientId: clientOrderId,
	}

	return
}
