package tools

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// Shared card helpers — same as in other card PRs. See
// pr/place-order-card for the full rationale on why these
// duplicate across branches during the staged rollout.

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

func unixMillisToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func formatQuantity(q decimal.Decimal) string {
	s := q.StringFixed(6)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

func baseAsset(symbol string) string {
	for _, sep := range []string{"-", "/", "_"} {
		if idx := strings.Index(symbol, sep); idx > 0 {
			return symbol[:idx]
		}
	}
	return "contracts"
}

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

func firstNonEmptyLocal(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func truncateForRow(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

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

var _ = cards.StatusNeutral
