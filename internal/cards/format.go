package cards

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Chooses card status from a signed numeric value.
// A small epsilon keeps dust-level amounts visually flat.
func SignedStatus(value float64) Status {
	switch {
	case math.IsNaN(value):
		return StatusNeutral
	case value > 0.00005:
		return StatusPositive
	case value < -0.00005:
		return StatusNegative
	default:
		return StatusFlat
	}
}

// Formats a signed number with a leading direction arrow.
// The emitted string is not padded.
func SignedWithArrow(value float64, decimals int) string {
	arrow := glyphsFor(SignedStatus(value)).Inline
	body := SignedNumber(value, decimals)
	return arrow + " " + body
}

// Formats signed floats with an explicit positive prefix.
// A lone zero has no sign.
func SignedNumber(value float64, decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	abs := math.Abs(value)
	body := strconv.FormatFloat(abs, 'f', decimals, 64)
	body = insertThousandsSeparator(body)
	switch {
	case math.IsNaN(value):
		return "—"
	case value > 0:
		return "+" + body
	case value < 0:
		return "-" + body
	default:
		return body
	}
}

// Formats signed dollar amounts with the sign before the dollar symbol.
func SignedUSD(value float64, decimals int) string {
	if math.IsNaN(value) {
		return "—"
	}
	abs := math.Abs(value)
	body := strconv.FormatFloat(abs, 'f', decimals, 64)
	body = insertThousandsSeparator(body)
	switch {
	case value > 0:
		return "+$" + body
	case value < 0:
		return "-$" + body
	default:
		return "$" + body
	}
}

// Formats non-negative dollar amounts where sign is meaningless.
func USD(value float64, decimals int) string {
	if math.IsNaN(value) {
		return "—"
	}
	body := strconv.FormatFloat(math.Abs(value), 'f', decimals, 64)
	return "$" + insertThousandsSeparator(body)
}

// Formats signed percent deltas from percent-unit values.
func Percent(value float64, decimals int) string {
	return SignedNumber(value, decimals) + "%"
}

// Formats fraction-unit values as signed percent deltas.
func PercentFraction(fraction float64, decimals int) string {
	return Percent(fraction*100, decimals)
}

// Humanizes an approximate positive duration for card display.
// The leading tilde signals that the value is not millisecond-precise.
func Duration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("~%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("~%dmin", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d / time.Hour)
		m := int((d - time.Duration(h)*time.Hour) / time.Minute)
		if m == 0 {
			return fmt.Sprintf("~%dh", h)
		}
		return fmt.Sprintf("~%dh %dmin", h, m)
	default:
		days := int(d / (24 * time.Hour))
		h := int((d - time.Duration(days)*24*time.Hour) / time.Hour)
		if h == 0 {
			return fmt.Sprintf("~%dd", days)
		}
		return fmt.Sprintf("~%dd %dh", days, h)
	}
}

// Formats a Unix time in compact UTC form for card columns.
func TimestampUTC(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

// insertThousandsSeparator adds commas to the integer portion of
// a stringified number. The decimal tail is preserved verbatim so
// precision does not change.
func insertThousandsSeparator(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	dot := strings.IndexByte(s, '.')
	var intPart, fracPart string
	if dot < 0 {
		intPart = s
	} else {
		intPart = s[:dot]
		fracPart = s[dot:]
	}
	if len(intPart) <= 3 {
		if neg {
			return "-" + intPart + fracPart
		}
		return intPart + fracPart
	}
	out := strings.Builder{}
	lead := len(intPart) % 3
	if lead > 0 {
		out.WriteString(intPart[:lead])
		if len(intPart) > lead {
			out.WriteByte(',')
		}
	}
	for i := lead; i < len(intPart); i += 3 {
		out.WriteString(intPart[i : i+3])
		if i+3 < len(intPart) {
			out.WriteByte(',')
		}
	}
	out.WriteString(fracPart)
	if neg {
		return "-" + out.String()
	}
	return out.String()
}
