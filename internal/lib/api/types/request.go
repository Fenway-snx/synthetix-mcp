package types

import (
	"strconv"
	"strings"
)

type Nonce int64

func (n Nonce) String() string {
	return strconv.FormatInt(int64(n), 10)
}

// Represents the "action" of a WS request.
type RequestAction string

// Reports whether the action requires nonce validation.
// Read-only operations (get* actions) do not require nonce for replay protection.
func (a RequestAction) NonceRequired() bool {
	return !strings.HasPrefix(string(a), "get")
}

// Reports whether the action must be performed by the account owner.
func (a RequestAction) RequiresOwner() bool {
	switch string(a) {
	case "createSubaccount",
		"removeAllDelegatedSigners",
		"transferCollateral",
		"updateSubAccountName",
		"withdrawCollateral":
		return true
	}
	return false
}

// Client-provided request ID from an API or WebSocket request.
// Untrusted and used only for echoing back in responses.
type ClientRequestId string

// =========================================================================
// Utility functions
// =========================================================================

// ===========================
// `ClientRequestId`
// ===========================

func ClientRequestIdToString(
	v ClientRequestId,
) (r string, err error) {

	r = string(v)

	return
}

func ClientRequestIdToStringUnvalidated(
	v ClientRequestId,
) (r string) {

	r, _ = ClientRequestIdToString(v)

	return
}

// ===========================
// `Nonce`
// ===========================

// ===========================
// `RequestAction`
// ===========================
