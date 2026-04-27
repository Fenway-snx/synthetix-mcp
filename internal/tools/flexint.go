package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FlexInt64 is a signed 64-bit integer that tolerates either a JSON number
// or a JSON string of digits on the wire. It is the canonical input type
// for 64-bit Synthetix identifiers (subaccount IDs, order IDs, ...) whose
// values routinely exceed JavaScript's Number.MAX_SAFE_INTEGER (2^53) and
// cannot safely round-trip through an IEEE-754 double.
//
// Marshaling always produces a JSON string. This keeps the wire format
// self-consistent: any client that echoes a FlexInt64 back (e.g. in a
// signed EIP-712 payload, a subsequent request, or a log) preserves the
// exact integer verbatim. Clients that still want to send a raw number
// can do so on input.
//
// A zero FlexInt64 marshals as the string "0"; callers that want to omit
// zero values should pair it with the ",omitempty" JSON tag, which works
// because the underlying type has a meaningful zero value.
type FlexInt64 int64

// Returns the underlying signed value. Having an accessor (rather than
// relying on implicit conversion at every call site) makes it easy to
// grep for call sites that must stay int64.
func (v FlexInt64) Int64() int64 { return int64(v) }

// UnmarshalJSON accepts either a JSON number or a JSON string. The JSON
// string branch parses with strconv.ParseInt so we get an explicit range
// check; the JSON number branch uses a json.Decoder with UseNumber to
// avoid an intermediate float64 that would silently lose precision for
// values > 2^53.
func (v *FlexInt64) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*v = 0
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return fmt.Errorf("flex int64: %w", err)
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*v = 0
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("flex int64: parse %q: %w", s, err)
		}
		*v = FlexInt64(n)
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var num json.Number
	if err := dec.Decode(&num); err != nil {
		return fmt.Errorf("flex int64: decode number: %w", err)
	}
	n, err := strconv.ParseInt(num.String(), 10, 64)
	if err != nil {
		return fmt.Errorf("flex int64: parse number %q: %w", num.String(), err)
	}
	*v = FlexInt64(n)
	return nil
}

// MarshalJSON emits the value as a JSON string so downstream JSON parsers
// that bucket numbers as IEEE-754 doubles (notably browsers and Node.js)
// cannot round the low-order digits.
func (v FlexInt64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatInt(int64(v), 10) + `"`), nil
}
