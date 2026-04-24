// Asset name strong type in the core domain.

package core

import (
	"encoding/json"
	"errors"
	"strings"

	snx_lib_utils_string "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/string"
)

// TODO: move move/all of these into a common area and make public, once we
// have separated lib into lib, corelib, apilib.
var (
	errAssetNameEmpty   = errors.New("asset name empty")
	errAssetNameInvalid = errors.New("asset name invalid")
)

// Strong type representing the asset name.
type AssetName string

// ===========================
// Creation functions
// ===========================

// Attempt to convert a string into an asset name.
func AssetNameFromString(symbol string) (AssetName, error) {

	if s, err := ValidateStringForAssetName(symbol); err == nil {
		return AssetName(s), nil
	} else {
		return "", err
	}
}

// Attempt to convert a symbol into an asset name.
func AssetNameFromSymbol(symbol Symbol) (AssetName, error) {

	if s, err := ValidateStringForAssetName(string(symbol)); err == nil {
		return AssetName(s), nil
	} else {
		return "", err
	}
}

// ===========================
// Methods
// ===========================

// Expresses an asset name as a symbol.
func (an AssetName) Symbol() Symbol {
	return Symbol(string(an))
}

// ===========================
// JSON Marshaling
// ===========================

// Marshal the string value of the asset name.
//
// Note:
// Unlike `AssetName#UnmarshalJSON()`, which applies correctness rules to
// an incoming asset name, this function will marshal out, in string form,
// any `AssetName` instance, even when such an instance violates the naming
// rules (e.g. by creating an instance around an empty string).
func (an AssetName) MarshalJSON() (bytes []byte, err error) {
	s := string(an)

	bytes, err = json.Marshal(s)

	return
}

// Attempt to unmarshal an asset name from a string representation.
//
// Note:
// Unlike `AssetName#MarshalJSON()`, which marshals any asset name (even
// if empty or otherwise invalid), this function is particular to produce
// only a valid name, subject to the following rules:
// - leading/trailing space is trimmed before processing;
// - empty value is not allowed;
// - must begin with an ASCII word-character (aka \w in regex terms);
// - must end with an ASCII word-character (aka \w in regex terms);
// - must not contain a hyphen character;
func (an *AssetName) UnmarshalJSON(data []byte) error {

	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	s, err = ValidateStringForAssetName(s)

	if err == nil {
		*an = AssetName(s)
	}

	return err
}

// ===========================
// Helper functions
// ===========================

// Validates a string as whether it may be suitable for an asset name,
// returning a sanitised form if so, or an empty string and an error
// otherwise.
func ValidateStringForAssetName(s string) (string, error) {

	s = strings.TrimSpace(s)

	if s == "" {
		return "", errAssetNameEmpty
	}

	// NOTE: if the validation requirements get more complex, consider using a
	// regular expression or shwild, or a state-machine (for maximum
	// performance).

	if !snx_lib_utils_string.ByteIsASCIIWordChar(s[0]) || !snx_lib_utils_string.ByteIsASCIIWordChar(s[len(s)-1]) {
		return "", errAssetNameInvalid
	}

	if strings.Contains(s, "-") {
		return "", errAssetNameInvalid
	}

	return s, nil
}

// ===========================
// API functions
// ===========================
