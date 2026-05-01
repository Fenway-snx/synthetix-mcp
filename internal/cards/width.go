package cards

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Returns terminal cell width for controlled card content.
// Emoji and wide glyphs count as 2, box drawing as 1, ANSI escapes as 0.
func displayWidth(s string) int {
	width := 0
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == 0x1b {
			// ANSI color codes contribute zero cells.
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

// Returns display width for a single rune in card content.
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

// Covers display-width-2 ranges used by card content.
// The list stays narrow because this package controls emitted glyphs.
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
	// Mark only emoji-presented geometric shapes as wide.
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

// Appends spaces until text reaches the requested display width.
// Overwide text is returned unchanged.
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

// Centers text inside a target display-cell width.
// Odd padding puts the extra space on the right.
func centerIn(s string, width int) string {
	deficit := width - displayWidth(s)
	if deficit <= 0 {
		return s
	}
	left := deficit / 2
	right := deficit - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// Shortens text to fit a display-cell width, adding an ellipsis if needed.
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
