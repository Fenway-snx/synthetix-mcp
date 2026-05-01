// Market name strong type in the core domain.

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
	errMarketNameEmpty   = errors.New("market name empty")
	errMarketNameInvalid = errors.New("market name invalid")
)

// Strong type representing the market name.
type MarketName string

// ===========================
// Creation functions
// ===========================

// Attempt to convert a string into a market name.
func MarketNameFromString(symbol string) (MarketName, error) {

	if s, err := ValidateStringForMarketName(symbol); err == nil {
		return MarketName(s), nil
	} else {
		return "", err
	}
}

// Attempt to convert a symbol into a market name.
func MarketNameFromSymbol(symbol Symbol) (MarketName, error) {

	if s, err := ValidateStringForMarketName(string(symbol)); err == nil {
		return MarketName(s), nil
	} else {
		return "", err
	}
}

// ===========================
// Methods
// ===========================

// Expresses a market name as a symbol.
func (mn MarketName) Symbol() Symbol {
	return Symbol(string(mn))
}

// ===========================
// JSON Marshaling
// ===========================

// Marshals the raw market-name string without validation.
func (mn MarketName) MarshalJSON() (bytes []byte, err error) {
	s := string(mn)

	bytes, err = json.Marshal(s)

	return
}

// Unmarshals and validates a market-name string.
// It trims space, requires word-character endpoints, and requires a hyphen.
func (mn *MarketName) UnmarshalJSON(data []byte) error {

	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	s, err = ValidateStringForMarketName(s)

	if err == nil {
		*mn = MarketName(s)
	}

	return err
}

// ===========================
// Helper functions
// ===========================

// Validates a string as whether it may be suitable for a market name,
// returning a sanitised form if so, or an empty string and an error
// otherwise.
func ValidateStringForMarketName(s string) (string, error) {

	s = strings.TrimSpace(s)

	if s == "" {
		return "", errMarketNameEmpty
	}

	// NOTE: if the validation requirements get more complex, consider using a
	// regular expression or shwild, or a state-machine (for maximum
	// performance).

	if len(s) < 3 {
		return "", errMarketNameInvalid
	}

	if !snx_lib_utils_string.ByteIsASCIIWordChar(s[0]) || !snx_lib_utils_string.ByteIsASCIIWordChar(s[len(s)-1]) {
		return "", errMarketNameInvalid
	}

	if !strings.Contains(s, "-") {
		return "", errMarketNameInvalid
	}

	s = strings.ToUpper(s)

	return s, nil
}

// ===========================
// API functions
// ===========================
