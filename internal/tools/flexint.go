package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Signed 64-bit integer that accepts JSON numbers or digit strings.
// Output always uses strings to preserve precision for JavaScript clients.
type FlexInt64 int64

// Returns the underlying signed value. Having an accessor (rather than
// relying on implicit conversion at every call site) makes it easy to
// grep for call sites that must stay int64.
func (v FlexInt64) Int64() int64 { return int64(v) }

// Accepts JSON numbers or strings without passing through float64.
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
