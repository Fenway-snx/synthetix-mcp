package cards

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// SignedStatus chooses a Status from a signed numeric value. Used
// by downstream cards to turn a PnL, a percent change, or a fee
// delta into a header glyph without duplicating the sign-check
// everywhere.
//
// An explicit epsilon avoids flipping a dust-level number into a
// positive/negative status — anything within ±0.005 cents renders
// as flat. Callers with domain-specific thresholds can pick their
// own status directly.
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

// SignedWithArrow formats a signed float with an explicit +/- sign
// and the appropriate inline arrow (▲/▼/◆) for the Status implied
// by the sign. The arrow sits to the LEFT of the number so the
// direction is visible before the reader parses the digits — which
// matters when a trader is glancing at a column.
//
// decimals controls precision. The emitted string is NOT padded;
// callers use padLeft / padRight when placing it in a column.
func SignedWithArrow(value float64, decimals int) string {
	arrow := glyphsFor(SignedStatus(value)).Inline
	body := SignedNumber(value, decimals)
	return arrow + " " + body
}

// SignedNumber formats a signed float with an explicit + prefix
// for non-negative values. A lone "0" gets no sign, matching how
// traders read a flat PnL ("0.00" not "+0.00").
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

// SignedUSD formats a dollar-denominated signed float as
// `+$76,300.00` / `-$540.00` / `$0.00`. Use for realized PnL, fees,
// notionals, funding. The dollar sign sits inside the sign so it
// reads as "positive 76k dollars" not "dollars, positive 76k".
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

// USD formats a non-negative dollar amount as `$76,300.00`. Used
// for entry/exit/mark prices where sign is meaningless.
func USD(value float64, decimals int) string {
	if math.IsNaN(value) {
		return "—"
	}
	body := strconv.FormatFloat(math.Abs(value), 'f', decimals, 64)
	return "$" + insertThousandsSeparator(body)
}

// Percent formats a signed percent delta like `+0.82%` / `-0.71%`.
// The value is already in percent units (0.82 means 0.82%), not
// fraction units.
func Percent(value float64, decimals int) string {
	return SignedNumber(value, decimals) + "%"
}

// PercentFraction formats a fraction-units value like 0.0082 as
// `+0.82%`. Convenience wrapper for REST fields that return
// unscaled fractions.
func PercentFraction(fraction float64, decimals int) string {
	return Percent(fraction*100, decimals)
}

// Duration humanizes a positive duration as "~15h 47min",
// "~2h 13min", "~8min", or "~45s". The `~` is intentional: every
// card that shows a held-for or age is approximate and the tilde
// signals that to the reader without them having to check
// whether the number is millisecond-precise.
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

// TimestampUTC formats a unix time as `2026-04-30 06:24 UTC` —
// the compact form the reference PnL card uses. Short enough to
// fit in a card's right-hand column without wrapping.
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
