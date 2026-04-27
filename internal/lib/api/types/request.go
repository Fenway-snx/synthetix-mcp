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

// NonceRequired returns true if the action requires nonce validation.
// Read-only operations (get* actions) do not require nonce for replay protection.
func (a RequestAction) NonceRequired() bool {
	return !strings.HasPrefix(string(a), "get")
}

// RequiresOwner returns true if the action can only be performed by the
// account owner, not by a delegated signer.
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

// Represents a client-provided request ID from an API or WebSocket request.
// This is an untrusted value from the client, used only for echoing back in responses.
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
