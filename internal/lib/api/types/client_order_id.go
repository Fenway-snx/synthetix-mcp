package types

import (
	"encoding/json"
	"errors"

	snx_lib_api_validation_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation/utils"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_utils_string "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/string"
)

// =========================================================================
// Constants
// =========================================================================

const (
	ClientOrderIdMaxLen = 255
)

var (
	errInputMustNotIncludeLeadingOrTrailingWhitespace = errors.New("input must not include leading or trailing whitespace")
	errInputUsesReservedInternalClientOrderIdPrefix   = errors.New("input uses reserved internal clientOrderId prefix")
)

// =========================================================================
// Types
// =========================================================================

// API representation of a client order Id.
type ClientOrderId string

const (
	ClientOrderId_Empty ClientOrderId = ""
)

// Marshals the raw client order ID without validation.
func (cloid ClientOrderId) MarshalJSON() (bytes []byte, err error) {
	s := string(cloid)

	bytes, err = json.Marshal(s)

	return
}

// Unmarshals and validates a client order ID string.
func (cloid *ClientOrderId) UnmarshalJSON(data []byte) error {

	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	if validatedForm, err := ClientOrderIdFromString(s); err != nil {
		*cloid = ClientOrderId_Empty

		return err
	} else {
		*cloid = ClientOrderId(validatedForm)

		return nil
	}
}

// =========================================================================
// Utility functions
// =========================================================================

// ===========================
// `ClientOrderId`
// ===========================

// from/to `core.ClientOrderId`

// Converts a Core `ClientOrderId` value into an API `ClientOrderId` value,
// without any validation.
func ClientOrderIdFromCoreClientOrderIdUnvalidated(
	v snx_lib_core.ClientOrderId,
) ClientOrderId {
	s := snx_lib_core.ClientOrderIdToStringUnvalidated(v)

	return ClientOrderIdFromStringUnvalidated(s)
}

// from/to `string`

// Parses a CLOID from a string, ensuring that it meets the format
// requirements of length, no-whitespace, and meeting the character
// restrictions.
func ClientOrderIdFromString(
	s string,
) (r ClientOrderId, err error) {

	if snx_lib_utils_string.HasLeadingOrTrailingWhitespace(s) {
		return "", errInputMustNotIncludeLeadingOrTrailingWhitespace
	}

	var validated string
	validated, err = snx_lib_api_validation_utils.ValidateClientSideString(
		s,
		ClientOrderIdMaxLen,
		// NOTE: we do _NOT_ specify snx_lib_api_validation_utils.ClientSideOption_Trim as we reject at ingestion
	)
	if err != nil {
		r = ClientOrderId_Empty

		return
	}

	if snx_lib_core.IsInternalClientOrderId(validated) {
		r = ClientOrderId_Empty
		err = errInputUsesReservedInternalClientOrderIdPrefix

		return
	}

	r = ClientOrderId(validated)

	return
}

func ClientOrderIdFromStringUnvalidated(
	v string,
) ClientOrderId {
	return ClientOrderId(v)
}

func ClientOrderIdToStringUnvalidated(
	cloid ClientOrderId,
) string {
	return string(cloid)
}

func ClientOrderIdToStringPtrUnvalidated(
	cloid ClientOrderId,
) (r *string) {
	s := string(cloid)
	r = &s

	return
}

func ClientOrderIdPtrFromStringPtrUnvalidated(
	p *string,
) (r *ClientOrderId) {
	if p != nil {
		cloid := ClientOrderId(*p)

		r = &cloid
	}

	return
}

func ClientOrderIdPtrToStringPtrUnvalidated(
	p *ClientOrderId,
) (r *string) {
	if p != nil {
		s := string(*p)

		r = &s
	}

	return
}

func ValidateClientOrderId(
	cloid ClientOrderId,
) (validatedForm ClientOrderId, err error) {

	validatedForm, err = ClientOrderIdFromString(string(cloid))

	return
}
