// Package cards renders fixed-width agent cards beside structured JSON.
// Visual language is centralized so downstream cards share one theme.
package cards

import (
	"os"
	"strings"
)

// Classifies a card outcome so renderers can choose consistent glyphs.
type Status int

const (
	// Informational outcome without positive or negative bias.
	StatusNeutral Status = iota
	// Positive outcome such as a win, fill, or healthy margin state.
	StatusPositive
	// Negative outcome such as a loss, rejection, or deteriorating account.
	StatusNegative
	// Break-even outcome rendered differently from neutral market data.
	StatusFlat
	// Cautionary outcome that should be read before continuing.
	StatusWarning
	// Critical outcome requiring immediate attention.
	StatusCritical
)

// Visual vocabulary for one card status.
// Inline glyphs avoid double-width emoji in dense rows.
type Glyphs struct {
	Header string
	Inline string
	ANSIFg string
}

// Returns the glyph triple for a card status.
func glyphsFor(s Status) Glyphs {
	switch s {
	case StatusPositive:
		return Glyphs{Header: "🟢", Inline: "▲", ANSIFg: ansiGreen}
	case StatusNegative:
		return Glyphs{Header: "🔴", Inline: "▼", ANSIFg: ansiRed}
	case StatusFlat:
		return Glyphs{Header: "⚪", Inline: "◆", ANSIFg: ansiDim}
	case StatusWarning:
		return Glyphs{Header: "🟡", Inline: "⚠", ANSIFg: ansiYellow}
	case StatusCritical:
		return Glyphs{Header: "🔥", Inline: "✖", ANSIFg: ansiBoldRed}
	case StatusNeutral:
		fallthrough
	default:
		return Glyphs{Header: "◆", Inline: "·", ANSIFg: ""}
	}
}

// ANSI escape codes. Only applied when ANSIEnabled() is true.
// Kept as a small, explicit palette — the goal is semantic color,
// not a theme system.
const (
	ansiReset   = "\x1b[0m"
	ansiGreen   = "\x1b[32m"
	ansiRed     = "\x1b[31m"
	ansiBoldRed = "\x1b[1;31m"
	ansiYellow  = "\x1b[33m"
	ansiDim     = "\x1b[2m"
)

// Box-drawing characters. Kept as named constants so any future
// "lite" theme (for clients that mangle box drawing) is a single
// swap.
const (
	borderTopLeft     = "╔"
	borderTopRight    = "╗"
	borderBottomLeft  = "╚"
	borderBottomRight = "╝"
	borderVertical    = "║"
	borderHorizontal  = "═"
	dividerHorizontal = "─"
)

// Total rendered width of every card in display cells.
const CardWidth = 80

// Mirrors the ANSI env var; default off for non-terminal clients.
func ansiEnabled() bool {
	return parseBoolEnv("SNXMCP_CARDS_ANSI", false)
}

// Global on/off switch for card rendering.
func cardsEnabled() bool {
	return parseBoolEnv("SNXMCP_CARDS_ENABLED", true)
}

// Reports whether cards should be rendered for this request.
func Enabled() bool { return cardsEnabled() }

func parseBoolEnv(name string, def bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if raw == "" {
		return def
	}
	switch raw {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off":
		return false
	default:
		return def
	}
}

// Wraps text in ANSI color when enabled.
func colorize(s, fg string) string {
	if fg == "" || !ansiEnabled() {
		return s
	}
	return fg + s + ansiReset
}
