package core

import (
	"math"
	"strings"
)

// -------------------------------------------------------------------------
// types
// -------------------------------------------------------------------------

type ClientOrderId string

const (
	ClientOrderId_Empty         ClientOrderId = ""           // A sentinel to represent an empty ClientOrderId value
	InternalClientOrderIdPrefix               = "snx-system/" // Reserved for service-generated CLOIDs only
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

// A composite order identifier, comprising the venue identifier and the
// optional client order identifier.
//
// NOTE: once all the CLOID work has been propagated throughout the
// codebase, this will be renamed to `OrderId`.
type OrderId struct {
	VenueId  VenueOrderId  `json:"void"`            // The venue order identifier.
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
