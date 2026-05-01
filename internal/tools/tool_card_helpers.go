package tools

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// This file collects the small pure helpers that multiple agent-
// trader cards share (quantity formatting, price-decimal heuristics,
// decimal parsing). They don't live inside internal/cards because
// the cards package is deliberately trading-agnostic — it knows
// how to draw a box, not how a 6-decimal crypto quantity should
// display.
//
// Duplicate definitions across card PRs are expected during the
// staged rollout: each PR branches off pr/cards-foundation. When
// these PRs merge the identical helpers collapse into one copy.

// decimalOrZero parses a REST string decimal, returning zero for
// empty / malformed inputs. REST payloads use strings for all
// numeric fields so floats don't lose precision on the wire;
// upstream has been observed to send "" for fields that haven't
// settled yet.
func decimalOrZero(s string) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// unixMillisToTime converts a REST timestamp (unix millis) into a
// time.Time. Zero-valued inputs stay zero so callers can
// recognise "no timestamp known".
func unixMillisToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// formatQuantity renders a decimal quantity with up to 6 decimal
// places, trimming trailing zeros. Decimal.String() would emit
// "0.1000000000" which is visually heavy and hides the actual
// precision the order carries.
func formatQuantity(q decimal.Decimal) string {
	s := q.StringFixed(6)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

// baseAsset extracts the base from a symbol like "BTC-USDT" ->
// "BTC". Used as the unit label on quantity rows. Falls back to
// "contracts" when the symbol isn't in the expected form.
func baseAsset(symbol string) string {
	for _, sep := range []string{"-", "/", "_"} {
		if idx := strings.Index(symbol, sep); idx > 0 {
			return symbol[:idx]
		}
	}
	return "contracts"
}

// priceDecimals picks a reasonable decimal count for a USD price:
// $76,300 reads as 0 decimals; $76.30 reads as 2 decimals; $0.0042
// reads as 4 decimals. Prevents sub-dollar crypto prices from
// looking like zeros in a card.
func priceDecimals(price float64) int {
	abs := price
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1000:
		return 0
	case abs >= 1:
		return 2
	case abs >= 0.01:
		return 4
	default:
		return 6
	}
}

// pnlDecimals picks decimals for dollar-denominated PnL amounts.
// Small crypto positions generate cents-and-fractions of cents of
// PnL, so we keep 3 decimals below $1 to make dust visible.
func pnlDecimals(pnl float64) int {
	abs := pnl
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 100:
		return 2
	case abs >= 1:
		return 2
	default:
		return 3
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// firstNonEmptyLocal returns the first non-blank string. The
// "Local" suffix avoids collision with a similarly named utility
// that may exist elsewhere in the tree under a different
// semantics.
func firstNonEmptyLocal(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// truncateForRow clamps a single-line string to max display chars
// with an ellipsis. Used for error messages and remediation hints
// that would otherwise break the card's column layout.
func truncateForRow(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// glyphsForDelta is a tool-side mirror of the cards package's
// internal arrow selector, for callers that only need the inline
// arrow without a whole status/glyphs lookup.
func glyphsForDelta(v float64) string {
	switch {
	case v > 0:
		return "▲"
	case v < 0:
		return "▼"
	default:
		return "◆"
	}
}

// Ensures the cards package import stays live even when only a
// subset of the above helpers are used in a specific PR.
var _ = cards.StatusNeutral
