package types

import (
	"encoding/json"
	"errors"
	"strings"

	snx_lib_utils_string "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/string"
)

// =========================================================================
// Constants
// =========================================================================

var (
	errSymbolEmpty   = errors.New("symbol name empty")
	errSymbolInvalid = errors.New("symbol name invalid")
)

// =========================================================================
// Types
// =========================================================================

type Symbol string

const (
	Symbol_None Symbol = ""
)

// Marshal the string value of the symbol.
func (sym Symbol) MarshalJSON() (bytes []byte, err error) {
	s := string(sym)

	bytes, err = json.Marshal(s)

	return
}

// Unmarshals and validates a symbol string.
// It trims space and requires word-character endpoints.
func (sym *Symbol) UnmarshalJSON(data []byte) error {

	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	s = strings.TrimSpace(s)

	if s == "" {

		return errSymbolEmpty
	}

	// NOTE: if the validation requirements get more complex, consider using a
	// regular expression or shwild

	if len(s) < 3 {
		return errSymbolInvalid
	}

	if !snx_lib_utils_string.ByteIsASCIIWordChar(s[0]) || !snx_lib_utils_string.ByteIsASCIIWordChar(s[len(s)-1]) {
		return errSymbolInvalid
	}

	// TODO: require a pair separator once symbol-pair semantics are defined.

	s = strings.ToUpper(s)

	*sym = Symbol(s)

	return nil
}

// =========================================================================
// Utility functions
// =========================================================================

// ===========================
// `Symbol`
// ===========================

// Converts a symbol from a string obtained from a trusted source, without
// any validation.
func SymbolFromStringUnvalidated(
	s string,
) Symbol {
	return Symbol(s)
}

func SymbolPtrFromStringUnvalidated(
	s string,
) *Symbol {
	m := Symbol(s)

	return &m
}

func SymbolPtrFromStringPtrUnvalidated(
	s *string,
) *Symbol {
	if s == nil {
		return nil
	} else {
		m := Symbol(*s)

		return &m
	}
}

func SymbolPtrToStringPtr(
	p *Symbol,
) *string {
	if p == nil {
		return nil
	} else {
		s := string(*p)

		return &s
	}
}

func SymbolToString(
	v Symbol,
) string {
	return string(v)
}

// Converts a slice of symbols into a slice of strings, eliding any that are
// [Symbol_None].
func SymbolsToStringsElidingNones(
	sl []Symbol,
) []string {
	r := make([]string, 0, len(sl))

	for _, sym := range sl {
		if Symbol_None != sym {
			r = append(r, SymbolToString(sym))
		}
	}

	return r
}

// Converts a slice of symbols into a slice of strings, without further
// consideration the validity of the specific symbols.
func SymbolsToStringsUnfiltered(
	sl []Symbol,
) []string {
	r := make([]string, len(sl))

	for i, sym := range sl {
		r[i] = SymbolToString(sym)
	}

	return r
}
