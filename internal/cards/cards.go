package cards

import "strings"

// Builder for a rendered agent card.
// Rendered values carry no hidden state.
type Card struct {
	Title    string
	Status   Status
	Sections []Section
	Footnote string
}

// Horizontal band inside a card, separated by dividers.
// Empty bands render as visual breathing room.
type Section struct {
	Rows []Row
}

// Single card line with label, value, and optional hint.
// The zero value renders as a blank line.
type Row struct {
	Label string
	Value string
	Hint  string
}

// Convenience value for intentional blank lines.
func Blank() Row { return Row{} }

// Produces a ready-to-emit string with fixed terminal width.
// Empty cards render as an empty string.
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

// Produces the padded title border with status glyph before title text.
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

// Produces the closing border.
func renderBottomBar(innerWidth int) string {
	return borderBottomLeft + strings.Repeat(borderHorizontal, innerWidth) + borderBottomRight
}

// Produces the inset divider between sections.
func renderDivider(innerWidth int) string {
	fill := strings.Repeat(dividerHorizontal, innerWidth-2)
	return borderVertical + " " + fill + " " + borderVertical
}

// Lays out one card row across label, value, and optional hint columns.
// Hinted rows anchor values at a fixed interior column for alignment.
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

	// Three-column row: label/value on the left, hint on the right.
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

// Renders a dimmed supplementary line, truncated to fit.
func renderFootnote(text string, innerWidth int) string {
	body := truncateToWidth(text, innerWidth-2)
	body = colorize(body, ansiDim)
	return borderVertical + " " + padRight(body, innerWidth-2) + " " + borderVertical
}
