package cards

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// displayWidth returns the number of terminal cells a string will
// occupy when rendered in a monospace font. This matters for
// padding/truncation: a 🟢 emoji takes 2 cells, a box-drawing
// character takes 1, an ANSI escape takes 0.
//
// We don't pull in a full East-Asian-Width library because the
// card surface is in our control — we know the glyphs we emit. A
// small explicit range table covers what we ship (CJK, emoji,
// symbol pictographs, regional indicators) and keeps dependencies
// at zero per the design principles.
func displayWidth(s string) int {
	width := 0
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == 0x1b {
			// ESC. Swallow the entire CSI sequence so ANSI
			// color codes contribute zero cells. The CSI
			// terminator is any byte in 0x40..0x7e after
			// the initial `[`.
			end := strings.IndexAny(s[i:], "mK")
			if end < 0 {
				// Malformed escape — treat the ESC as a
				// zero-width codepoint and keep scanning.
				i += size
				continue
			}
			i += end + 1
			continue
		}
		width += runeWidth(r)
		i += size
	}
	return width
}

// runeWidth returns the cell width of a single rune. Control runes
// are treated as 0 (they shouldn't appear in card content), most
// runes are 1, and the East-Asian-Wide / emoji ranges we actually
// emit are 2.
func runeWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 0x20 || r == 0x7f:
		return 0
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), unicode.Is(unicode.Cf, r):
		// Combining marks and format characters attach to the
		// preceding rune and add no width of their own.
		return 0
	case isWideRune(r):
		return 2
	default:
		return 1
	}
}

// isWideRune covers the display-width-2 ranges the card surface
// actually renders. The list is intentionally narrow: we own the
// glyph vocabulary, so we only need to cover ranges present in
// theme.glyphsFor, plus common CJK for trader-supplied symbols.
func isWideRune(r rune) bool {
	switch {
	// Miscellaneous Symbols and Pictographs (🟢 🔴 🟡 🔥 and
	// their neighbors).
	case r >= 0x1F300 && r <= 0x1F5FF:
		return true
	// Emoticons.
	case r >= 0x1F600 && r <= 0x1F64F:
		return true
	// Transport and Map Symbols.
	case r >= 0x1F680 && r <= 0x1F6FF:
		return true
	// Supplemental Symbols and Pictographs.
	case r >= 0x1F900 && r <= 0x1F9FF:
		return true
	// Symbols and Pictographs Extended-A.
	case r >= 0x1FA70 && r <= 0x1FAFF:
		return true
	// Geometric Shapes Extended — home of the colored circles
	// and squares (🟢🔴🟡🟠🟣 etc.) that the theme uses as the
	// primary Status glyphs.
	case r >= 0x1F780 && r <= 0x1F7FF:
		return true
	// Enclosed Alphanumeric Supplement (some emoji live here).
	case r >= 0x1F100 && r <= 0x1F1FF:
		return true
	// Geometric shapes that Unicode/CLDR classifies as emoji
	// presentation by default: ⚪ ⚫ 🟠 etc. The broad geometric
	// shapes block at 0x25A0-0x25FF is single-width for the
	// variants we use (◆ · ○ ●) so we deliberately don't mark it
	// wide.
	case r == 0x26AA, r == 0x26AB:
		return true
	// CJK Unified Ideographs / Hiragana / Katakana / Hangul —
	// we don't use these in the card surface ourselves, but a
	// trader's symbol string might.
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	case r >= 0x3040 && r <= 0x309F:
		return true
	case r >= 0x30A0 && r <= 0x30FF:
		return true
	case r >= 0xAC00 && r <= 0xD7A3:
		return true
	default:
		return false
	}
}

// padRight appends spaces to s until it occupies width display
// cells. If s already exceeds width it is returned unchanged — the
// caller is responsible for truncating beforehand.
func padRight(s string, width int) string {
	deficit := width - displayWidth(s)
	if deficit <= 0 {
		return s
	}
	return s + strings.Repeat(" ", deficit)
}

// padLeft prepends spaces to s until it occupies width display
// cells. If s already exceeds width it is returned unchanged.
func padLeft(s string, width int) string {
	deficit := width - displayWidth(s)
	if deficit <= 0 {
		return s
	}
	return strings.Repeat(" ", deficit) + s
}

// centerIn pads s with spaces on both sides so it sits centered
// inside width display cells. When width-displayWidth(s) is odd
// the extra space goes on the right to keep cards visually
// balanced toward the title bar's left edge.
func centerIn(s string, width int) string {
	deficit := width - displayWidth(s)
	if deficit <= 0 {
		return s
	}
	left := deficit / 2
	right := deficit - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// truncateToWidth shortens s so it fits in width display cells,
// appending an ellipsis when a truncation actually happens. Used
// by row rendering when a label or value is longer than its
// column.
func truncateToWidth(s string, width int) string {
	if displayWidth(s) <= width {
		return s
	}
	if width <= 1 {
		return strings.Repeat(".", width)
	}
	cells := 0
	out := strings.Builder{}
	for _, r := range s {
		rw := runeWidth(r)
		if cells+rw > width-1 {
			break
		}
		out.WriteRune(r)
		cells += rw
	}
	out.WriteRune('…')
	return out.String()
}
