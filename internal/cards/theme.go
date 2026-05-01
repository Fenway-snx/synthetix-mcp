// Package cards renders fixed-width ASCII/unicode "agent cards" that
// sit alongside the structured JSON in every MCP tool response. The
// cards are designed for terminal-first MCP clients (Claude Code)
// where the chat pane is a monospace markdown surface.
//
// Visual language is locked in one place (this file). Downstream
// cards never invent their own glyphs or colors — they select a
// Status and the theme picks the right glyph and ANSI pair. That
// keeps every card in the surface consistent and makes a future
// theme swap one-file change.
package cards

import (
	"os"
	"strings"
)

// Status classifies the semantic outcome of a card so the renderer
// picks the right header glyph, inline arrow, and (optional) ANSI
// color. Adding a Status is a one-line change; every downstream
// card picks from this set instead of hard-coding glyphs.
type Status int

const (
	// StatusNeutral is for informational cards that have no
	// positive/negative bias — orderbooks, market snapshots,
	// account summaries without a dominant signal.
	StatusNeutral Status = iota
	// StatusPositive is a winning trade, a filled order, a healthy
	// margin state — anything the trader wants to see more of.
	StatusPositive
	// StatusNegative is a losing trade, a rejected order, a
	// deteriorating account — anything the trader wants to know
	// about but probably doesn't want to see more of.
	StatusNegative
	// StatusFlat is a break-even trade or a neutral outcome where
	// "no change" is the headline. Rendered deliberately
	// differently from Neutral so zero-PnL closes don't look like
	// market data.
	StatusFlat
	// StatusWarning is a guardrail trip, a soft rejection, a
	// cautionary state. Trader should read the card before
	// continuing.
	StatusWarning
	// StatusCritical is a liquidation, a hard failure, a
	// dead-man-switch trigger — anything the trader needs to see
	// right now.
	StatusCritical
)

// Glyphs is the full visual vocabulary for one Status. Header is the
// colorful emoji that lives in the card title bar; Inline is the
// plain-text arrow glyph that goes next to signed numbers inside
// dense rows (tables, sparklines) where emoji double-width would
// break column alignment. ANSIFg is an optional foreground color
// that only renders when cards are ANSI-enabled.
type Glyphs struct {
	Header string
	Inline string
	ANSIFg string
}

// glyphsFor returns the glyph triple for a Status. The emoji layer
// carries all necessary color on its own; the ANSI layer is purely
// additive and gated on SNXMCP_CARDS_ANSI.
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

// CardWidth is the total rendered width of every card in display
// cells. 80 is the historical terminal width and matches the chat
// pane on Claude Code without wrapping; every card in the surface
// renders to exactly this width so they stack cleanly.
const CardWidth = 80

// ansiEnabled mirrors the SNXMCP_CARDS_ANSI env var. Checked once
// per render; cheap enough that we don't bother caching.
//
// Default off: Claude Desktop, Cursor, and the structured-content
// consumers we care about would render ANSI as literal garbage
// like `[32m+$0.62[0m`. Operators running Claude Code from a true
// terminal can opt in with SNXMCP_CARDS_ANSI=true.
func ansiEnabled() bool {
	return parseBoolEnv("SNXMCP_CARDS_ANSI", false)
}

// cardsEnabled is the global on/off switch for the whole card
// surface. Default on so the trading UX ships live; operators who
// need pure JSON (CI, smoke tests, tight token budgets) can set
// SNXMCP_CARDS_ENABLED=false.
func cardsEnabled() bool {
	return parseBoolEnv("SNXMCP_CARDS_ENABLED", true)
}

// Enabled reports whether cards should be rendered at all. Cheap
// to call per-request — the MCP server hits this once per tool
// response and callers can short-circuit JSON-only paths when it
// returns false.
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

// colorize wraps s in an ANSI sequence when ANSI mode is on and
// fg is non-empty. Otherwise returns s unchanged. The reset is
// always paired so color never bleeds into surrounding cells.
func colorize(s, fg string) string {
	if fg == "" || !ansiEnabled() {
		return s
	}
	return fg + s + ansiReset
}
