package cards

import "strings"

// Card is the builder for a single rendered "agent card". Downstream
// packages compose a card by picking a Status, a Title, and a list
// of Sections, each containing Rows. The renderer handles width,
// padding, emoji alignment, and ANSI wrapping so every card in the
// surface looks identical.
//
// Philosophy: cards are value types. Build, render, throw away. No
// hidden state, no caching, no "mutate after render" semantics.
type Card struct {
	Title    string
	Status   Status
	Sections []Section
	Footnote string
}

// Section is a horizontal band inside a card, separated from its
// neighbors by a `─` divider. Sections carry rows; empty sections
// render as blank bands (useful for visual breathing room inside
// dense tables).
type Section struct {
	Rows []Row
}

// Row is a single line in a card. Label sits in the left column,
// Value in the right, Hint is an optional third column rendered
// dim/neutral at the far right when there's room. A zero-value
// Row is a valid blank line.
type Row struct {
	Label string
	Value string
	Hint  string
}

// Blank is a convenience for inserting an intentional empty line
// inside a Section. Reads better than `Row{}` at call sites.
func Blank() Row { return Row{} }

// Render turns a Card into a ready-to-emit string terminated by a
// trailing newline. The width is always CardWidth cells so cards
// stack cleanly in the chat pane. A fully empty Card renders as
// an empty string so "no card" callers don't have to special-case.
func (c Card) Render() string {
	if c.Title == "" && len(c.Sections) == 0 && c.Footnote == "" {
		return ""
	}

	innerWidth := CardWidth - 2 // Two bytes for the left/right `║`.
	b := strings.Builder{}

	b.WriteString(renderTitleBar(c.Title, c.Status, innerWidth))
	b.WriteByte('\n')

	firstSection := true
	for _, section := range c.Sections {
		if !firstSection {
			b.WriteString(renderDivider(innerWidth))
			b.WriteByte('\n')
		}
		firstSection = false
		for _, row := range section.Rows {
			b.WriteString(renderRow(row, innerWidth))
			b.WriteByte('\n')
		}
	}

	if c.Footnote != "" {
		b.WriteString(renderDivider(innerWidth))
		b.WriteByte('\n')
		b.WriteString(renderFootnote(c.Footnote, innerWidth))
		b.WriteByte('\n')
	}

	b.WriteString(renderBottomBar(innerWidth))
	b.WriteByte('\n')
	return b.String()
}

// renderTitleBar produces `╔═ <glyph> TITLE ═...═╗` padded to
// innerWidth. The glyph sits to the left of the title text so the
// eye lands on the status before reading the words.
func renderTitleBar(title string, status Status, innerWidth int) string {
	glyph := glyphsFor(status).Header
	body := ""
	if title != "" {
		if glyph != "" {
			body = glyph + " " + title
		} else {
			body = title
		}
	} else if glyph != "" {
		body = glyph
	}

	// Leave one space on each side of the body for visual
	// breathing room against the `═` fill.
	bodyCells := displayWidth(body)
	if bodyCells+2 >= innerWidth {
		// Title too long to decorate — render it plain and
		// truncated.
		return borderTopLeft + truncateToWidth(body, innerWidth) + borderTopRight
	}
	filler := innerWidth - bodyCells - 2
	left := filler / 2
	right := filler - left
	return borderTopLeft +
		strings.Repeat(borderHorizontal, left) +
		" " + body + " " +
		strings.Repeat(borderHorizontal, right) +
		borderTopRight
}

// renderBottomBar produces the `╚═...═╝` closing border.
func renderBottomBar(innerWidth int) string {
	return borderBottomLeft + strings.Repeat(borderHorizontal, innerWidth) + borderBottomRight
}

// renderDivider produces a `║ ─...─ ║` line separating sections.
// One space of inset on each side keeps the divider visually
// distinct from the outer border.
func renderDivider(innerWidth int) string {
	fill := strings.Repeat(dividerHorizontal, innerWidth-2)
	return borderVertical + " " + fill + " " + borderVertical
}

// renderRow lays a Row out as:
//
//	║ Label....................  Value            Hint................... ║
//
// The card is split into three logical columns: a flexible label
// zone on the left, a right-aligned Value sitting at the far right
// (no Hint) OR at a fixed interior anchor (Hint present), and a
// right-aligned Hint occupying the remaining right-hand space.
//
// Anchoring Value at a fixed interior column when a Hint is
// present is the UX we want: in a PnL card, "Exit: $76,300" and
// "Quantity: 0.001 BTC" should have their values vertically
// aligned across rows, and the trade-side hint ("+$617 (+0.82%)"
// or "Notional: $76.30") should sit in its own right column. This
// matches the reference card image byte-for-byte in spirit.
func renderRow(row Row, innerWidth int) string {
	if row.Label == "" && row.Value == "" && row.Hint == "" {
		return borderVertical + strings.Repeat(" ", innerWidth) + borderVertical
	}

	// 1-cell margins inside the border.
	contentWidth := innerWidth - 2

	if row.Hint == "" {
		// Two-column row: label flex left, value pinned right.
		valueWidth := displayWidth(row.Value)
		if valueWidth > contentWidth {
			row.Value = truncateToWidth(row.Value, contentWidth)
			valueWidth = displayWidth(row.Value)
		}
		labelZone := contentWidth - valueWidth - 1 // 1 cell of gap
		if labelZone < 0 {
			labelZone = 0
		}
		label := truncateToWidth(row.Label, labelZone)
		return borderVertical + " " +
			padRight(label, labelZone) + " " + row.Value +
			" " + borderVertical
	}

	// Three-column row with Hint. Split contentWidth 50/50: the
	// left half holds Label + Value (value right-aligned inside
	// it), the right half holds the Hint (also right-aligned so
	// it sits against the card edge and long hints don't push
	// into the value column).
	leftZone := contentWidth / 2
	rightZone := contentWidth - leftZone

	valueText := row.Value
	if displayWidth(valueText) > leftZone {
		valueText = truncateToWidth(valueText, leftZone)
	}
	valueCells := displayWidth(valueText)
	labelZone := leftZone - valueCells - 1 // 1 cell of gap between label and value
	if labelZone < 0 {
		labelZone = 0
	}
	label := truncateToWidth(row.Label, labelZone)

	hintText := row.Hint
	if displayWidth(hintText) > rightZone {
		hintText = truncateToWidth(hintText, rightZone)
	}

	return borderVertical + " " +
		padRight(label, labelZone) + " " +
		valueText +
		padLeft(hintText, rightZone) +
		" " + borderVertical
}

// renderFootnote renders a single italic-feeling line dimmed to
// signal "supplementary info". The line is one cell indented on
// each side and truncated to fit.
func renderFootnote(text string, innerWidth int) string {
	body := truncateToWidth(text, innerWidth-2)
	body = colorize(body, ansiDim)
	return borderVertical + " " + padRight(body, innerWidth-2) + " " + borderVertical
}
