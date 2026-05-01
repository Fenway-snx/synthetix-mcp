package types

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// =========================================================================
// Constants
// =========================================================================

const (
	// NOTE: these values explicitly specified as an _untyped_ integer

	VenueOrderIdMaximumValidValue = 0x8000_0000_0000_0000 - 1
)

var (
	errVenueOrderIdEmpty            = errors.New("venue order id cannot be empty")
	errVenueOrderIdCannotBeNegative = errors.New("venue order id cannot be negative")
	errVenueOrderIdCannotBeZero     = errors.New("venue order id cannot be zero")
	errVenueOrderIdInvalidFormat    = errors.New("venue order id has invalid format")
	errVenueOrderIdTooLarge         = errors.New("venue order id too large")
)

// =========================================================================
// Types
// =========================================================================

// API representation of a venue order Id.
type VenueOrderId string

const (
	VenueOrderId_None VenueOrderId = ""
)

// Marshals the raw venue order ID without validation.
func (void VenueOrderId) MarshalJSON() (bytes []byte, err error) {
	s := string(void)

	bytes, err = json.Marshal(s)

	return
}

// Unmarshals and validates a non-negative decimal venue order ID.
func (void *VenueOrderId) UnmarshalJSON(data []byte) error {

	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	s = strings.TrimSpace(s)

	if s == "" {

		return errVenueOrderIdEmpty
	}

	if i, err := strconv.ParseInt(s, 10, 64); err != nil {

		// NOTE: we deliberately do not bother to qualify the "invalid format"
		// with the specific failure, because it's perfectly fine to avoid doing
		// such extra expense and pushing responsibility back to the API client.

		return errVenueOrderIdInvalidFormat
	} else {

		switch {
		case i < 0:

			return errVenueOrderIdCannotBeNegative
		case i == 0:

			return errVenueOrderIdCannotBeZero
		default:

			*void = VenueOrderId(strconv.FormatInt(i, 10))

			return nil
		}
	}
}

// =========================================================================
// Utility functions
// =========================================================================

// ===========================
// `VenueOrderId`
// ===========================

// from/to `uint64`

func VenueOrderIdFromUint(
	v uint64,
) (VenueOrderId, error) {
	switch {
	case v == 0:

		return "", errVenueOrderIdCannotBeZero
	case v > VenueOrderIdMaximumValidValue:

		return "", errVenueOrderIdTooLarge
	default:

		return VenueOrderIdFromUintUnvalidated(v), nil
	}
}

func VenueOrderIdFromUintRaw(
	v uint64,
) string {
	return uitoa(v)
}

func VenueOrderIdFromUintUnvalidated(
	v uint64,
) VenueOrderId {
	return VenueOrderId(VenueOrderIdFromUintRaw(v))
}

func VenueOrderIdToUintUnvalidated(
	void VenueOrderId,
) (r uint64) {
	// NOTE: we do not check for failure because the `VenueOrderId`` must be valid
	// from its unmarshaling

	s := string(void)
	r, _ = strconv.ParseUint(s, 10, 64)

	return
}

func VenueOrderIdArrayFromUintArrayUnvalidated(
	ar []uint64,
) []VenueOrderId {
	r := make([]VenueOrderId, len(ar))

	for i, u := range ar {
		r[i] = VenueOrderIdFromUintUnvalidated(u)
	}

	return r
}

func VenueOrderIdArrayToUintArrayUnvalidated(
	ar []VenueOrderId,
) []uint64 {
	r := make([]uint64, len(ar))

	for i, u := range ar {
		r[i] = VenueOrderIdToUintUnvalidated(u)
	}

	return r
}

// from/to `core.VenueOrderId`

// Converts a Core `VenueOrderId` value into an API `VenueOrderId` value,
// without any validation.
func VenueOrderIdFromCoreVenueOrderIdUnvalidated(
	v snx_lib_core.VenueOrderId,
) VenueOrderId {
	u := snx_lib_core.VenueOrderIdToUint64Unvalidated(v)

	return VenueOrderIdFromUintUnvalidated(u)
}
