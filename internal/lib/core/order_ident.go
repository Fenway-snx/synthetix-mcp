package core

import (
	"math"
	"strings"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// -------------------------------------------------------------------------
// types
// -------------------------------------------------------------------------

type ClientOrderId string

const (
	ClientOrderId_Empty         ClientOrderId = ""              // A sentinel to represent an empty ClientOrderId value
	InternalClientOrderIdPrefix               = "snx-internal/" // Reserved for system-generated CLOIDs only
)

type VenueOrderId uint64

const (
	VenueOrderId_Zero    VenueOrderId = 0              // The 0-initialised value
	VenueOrderId_Invalid VenueOrderId = math.MaxUint64 // A sentinel to represent an invalid VenueOrderId value
)

func (void VenueOrderId) VenueOrderId() VenueOrderId {
	return void
}

type VenueOrderIdProvider interface {
	VenueOrderId() VenueOrderId
}

// A composite order identifier, comprising the system-generated venue
// identifier and the optional client order identifier.
//
// NOTE: once all the CLOID work has been propagated throughout the
// codebase, this will be renamed to `OrderId`.
type OrderId struct {
	VenueId  VenueOrderId  `json:"void"`            // The internal order identifier, which is entirely definitive.
	ClientId ClientOrderId `json:"cloid,omitempty"` // An optional client identifier, which accompanies the venue identifier but has zero functional impact on order handling.
}

func (o OrderId) VenueOrderId() VenueOrderId {
	return o.VenueId
}

// -------------------------------------------------------------------------
// Utility functions
// -------------------------------------------------------------------------

// ---------------------------
// `ClientOrderId`
// ---------------------------

// from/to `string`

// Obtains a `ClientOrderId` from a string, without validating the string
// contents.
func ClientOrderIdFromStringUnvalidated(
	s string,
) ClientOrderId {
	return ClientOrderId(s)
}

// Obtains a `ClientOrderId` from a string pointer, including validating the
// string contents.
func ClientOrderIdFromStringPtr(
	p *string,
) (
	clientOrderId ClientOrderId,
	err error,
) {
	if p != nil {
		if *p != "" {

			clientOrderId = ClientOrderId(*p)
		}
	}

	return
}

// Obtains a `ClientOrderId` from a string pointer, without validating the
// string contents.
func ClientOrderIdFromStringPtrUnvalidated(
	p *string,
) (
	clientOrderId ClientOrderId,
) {
	if p != nil {
		if *p != "" {

			clientOrderId = ClientOrderId(*p)
		}
	}

	return
}

// Converts a `ClientOrderId` into a string, without validating the
// instance contents.
func ClientOrderIdToStringUnvalidated(
	cloid ClientOrderId,
) string {
	return string(cloid)
}

func IsInternalClientOrderId(cloid string) bool {
	return strings.HasPrefix(cloid, InternalClientOrderIdPrefix)
}

// ---------------------------
// `OrderId`
// ---------------------------

// from/to `*v4grpc.OrderId`

// Obtains an `OrderId` from a `*v4grpc.OrderId`, without validating any
// aspects of the source.
//
// Note:
// As a TEMPORARY measure we actually perform partial validation out of
// caution due to the scope of the CLOID changes. This will likely be
// removed in the future.
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

// Obtains an `OrderId` pointer from a `*v4grpc.OrderId`, without validating
// any aspects of the source.
//
// Note:
// As a TEMPORARY measure we actually perform partial validation out of
// caution due to the scope of the CLOID changes. This will likely be
// removed in the future.
func OrderIdPtrFromGRPCOrderIdUnvalidated(
	orderId *v4grpc.OrderId,
) (r *OrderId) {

	// NOTE: ideally we should not need this defensive programming action, but
	// given the scope of CLOID changes we are taking this caution.
	if orderId == nil || orderId.VenueId == 0 {
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

// Obtains an `OrderId` pointer or `nil` from a `*v4grpc.OrderId`, without
// validating any aspects of the source other that it is not `nil`.
//
// Note:
// As a TEMPORARY measure we actually perform partial validation - insofar
// as verifying that it the source pointer is not to a default-initialised
// instance - out of caution due to the scope of the CLOID changes. This
// will likely be removed in the future.
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

// Converts an `OrderId` instance into a `*v4grpc.OrderId`, without
// validating the contents.
func OrderIdToGRPCOrderIdPtrUnvalidated(
	orderId OrderId,
) (r *v4grpc.OrderId) {

	if orderId.VenueId == 0 {
		return
	}

	venueOrderId := VenueOrderIdToUint64Unvalidated(orderId.VenueId)
	clientOrderId := ClientOrderIdToStringUnvalidated(orderId.ClientId)

	r = &v4grpc.OrderId{
		VenueId:  venueOrderId,
		ClientId: clientOrderId,
	}

	return
}

func OrderIdToGRPCOrderIdPtrOrNilUnvalidated(
	orderId OrderId,
) *v4grpc.OrderId {

	return &v4grpc.OrderId{
		VenueId:  uint64(orderId.VenueId),
		ClientId: string(orderId.ClientId),
	}
}

// ---------------------------
// `VenueOrderId`
// ---------------------------

// from/to `uint64`

// Obtains a `VenueOrderId` from an integer, without validating the
// integer's value.
func VenueOrderIdFromUintUnvalidated(
	v uint64,
) VenueOrderId {
	return VenueOrderId(v)
}

// Converts a `VenueOrderId` into an integer, without validating the
// instance contents.
func VenueOrderIdToUint64Unvalidated(
	void VenueOrderId,
) uint64 {
	return uint64(void)
}

// from/to `*uint64`

// Obtains a `VenueOrderId` pointer or `nil` from an integer pointer or nil,
// without validating the integer's value.
func VenueOrderIdPtrFromUint64PtrUnvalidated(
	p *uint64,
) *VenueOrderId {
	if p == nil {
		return nil
	} else {
		r := VenueOrderIdFromUintUnvalidated(*p)

		return &r
	}
}

// Converts a `VenueOrderId` pointer or `nil` to an integer pointer or nil,
// without validating the instance contents.
func VenueOrderIdPtrToUint64PtrUnvalidated(
	p *VenueOrderId,
) *uint64 {
	if p == nil {
		return nil
	} else {
		r := VenueOrderIdToUint64Unvalidated(*p)

		return &r
	}
}

// from/to `[]uint64`

// Obtains a slice of `VenueOrderId` from a slice of integers, without
// validating the source integers' values.
func VenueOrderIdArrayFromUint64ArrayUnvalidated(
	ar []uint64,
) []VenueOrderId {
	r := make([]VenueOrderId, len(ar))

	for i, u := range ar {
		r[i] = VenueOrderIdFromUintUnvalidated(u)
	}

	return r
}

// Converts a slice of `VenueOrderId` into a slice of integers, without
// validating the instances' values.
func VenueOrderIdArrayToUint64ArrayUnvalidated(
	ar []VenueOrderId,
) []uint64 {
	r := make([]uint64, len(ar))

	for i, u := range ar {
		r[i] = VenueOrderIdToUint64Unvalidated(u)
	}

	return r
}
