package tools

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// Shared pure helpers for trader cards.
// Kept outside internal/cards so rendering stays trading-agnostic.

// Parses a REST decimal string, returning zero for empty or malformed input.
// REST uses strings so numeric fields preserve precision on the wire.
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

// Converts REST Unix milliseconds into UTC time.
// Zero-valued inputs stay zero.
func unixMillisToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// Renders a decimal quantity with up to 6 places.
// Trailing zeros are trimmed for compact card rows.
func formatQuantity(q decimal.Decimal) string {
	s := q.StringFixed(6)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

// Extracts the base from symbols such as "BTC-USDT".
// Falls back to "contracts" for unknown forms.
func baseAsset(symbol string) string {
	for _, sep := range []string{"-", "/", "_"} {
		if idx := strings.Index(symbol, sep); idx > 0 {
			return symbol[:idx]
		}
	}
	return "contracts"
}

// Picks a compact decimal count for USD prices.
// Sub-dollar crypto prices keep enough precision to avoid rendering as zero.
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

// Picks decimals for dollar-denominated PnL amounts.
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

// Returns the first non-blank string.
func firstNonEmptyLocal(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// Clamps a single-line string with an ellipsis for card rows.
func truncateForRow(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// Mirrors the cards package's inline delta arrow selector.
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
