package tools

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/synthetixio/synthetix-go/types"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// closePositionSnapshot captures everything we need about a
// position the moment BEFORE the close order is sent. It is passed
// into the PnL card builder alongside the post-fill result so the
// card can show held-for, entry/exit, realized PnL, fees, and
// funding paid in one glance.
//
// We build this eagerly (instead of reading positions a second
// time after the close) because the post-close getPositions call
// would race the matching engine: a fully-closed position drops
// out of the response, and a partial close would still be visible
// but with stale CreatedAt / NetFunding values.
type closePositionSnapshot struct {
	Symbol        string
	Side          string
	Quantity      decimal.Decimal
	EntryPrice    decimal.Decimal
	CreatedAt     time.Time
	UnrealizedPnl decimal.Decimal
	UsedMargin    decimal.Decimal
	NetFunding    decimal.Decimal
	// Known indicates whether we were able to find the position
	// before the close. A false Known means we'll render a
	// degraded card (exit price + symbol only) rather than lie
	// with zeroed entry/PnL fields.
	Known bool
}

// captureClosePositionSnapshot reads positions and returns a
// snapshot for the target symbol. It does not error when the
// position is missing — cards are a display concern, and the
// real close flow already validates exposure via
// resolveClosablePosition. A missing snapshot degrades the card
// without blocking the trade.
func captureClosePositionSnapshot(ctx context.Context, reads *TradeReadClient, tc ToolContext, symbol string) closePositionSnapshot {
	if reads == nil {
		return closePositionSnapshot{Symbol: symbol}
	}
	positions, err := reads.GetPositions(ctx, tc)
	if err != nil {
		return closePositionSnapshot{Symbol: symbol}
	}
	normSym := strings.ToUpper(strings.TrimSpace(symbol))
	for i := range positions {
		p := positions[i]
		if !strings.EqualFold(strings.TrimSpace(p.Symbol), normSym) {
			continue
		}
		qty, err := decimal.NewFromString(strings.TrimSpace(p.Quantity))
		if err != nil || qty.IsZero() {
			continue
		}
		return closePositionSnapshot{
			Symbol:        p.Symbol,
			Side:          strings.ToLower(strings.TrimSpace(p.Side)),
			Quantity:      qty.Abs(),
			EntryPrice:    decimalOrZero(p.EntryPrice),
			CreatedAt:     unixMillisToTime(p.CreatedAt),
			UnrealizedPnl: decimalOrZero(p.UnrealizedPnl),
			UsedMargin:    decimalOrZero(p.UsedMargin),
			NetFunding:    decimalOrZero(p.NetFunding),
			Known:         true,
		}
	}
	return closePositionSnapshot{Symbol: symbol}
}

// captureClosePositionSnapshots batch-fetches positions once for
// the multi-symbol close_all path so we don't make N GetPositions
// calls. It returns a symbol→snapshot map keyed by the canonical
// upper-case symbol.
func captureClosePositionSnapshots(ctx context.Context, reads *TradeReadClient, tc ToolContext, symbols []string) map[string]closePositionSnapshot {
	out := make(map[string]closePositionSnapshot, len(symbols))
	if reads == nil {
		return out
	}
	positions, err := reads.GetPositions(ctx, tc)
	if err != nil {
		return out
	}
	byCanonical := make(map[string]*types.Position, len(positions))
	for i := range positions {
		p := &positions[i]
		qty, err := decimal.NewFromString(strings.TrimSpace(p.Quantity))
		if err != nil || qty.IsZero() {
			continue
		}
		byCanonical[strings.ToUpper(strings.TrimSpace(p.Symbol))] = p
	}
	for _, sym := range symbols {
		canonical := strings.ToUpper(strings.TrimSpace(sym))
		p, ok := byCanonical[canonical]
		if !ok {
			out[canonical] = closePositionSnapshot{Symbol: sym}
			continue
		}
		qty, _ := decimal.NewFromString(strings.TrimSpace(p.Quantity))
		out[canonical] = closePositionSnapshot{
			Symbol:        p.Symbol,
			Side:          strings.ToLower(strings.TrimSpace(p.Side)),
			Quantity:      qty.Abs(),
			EntryPrice:    decimalOrZero(p.EntryPrice),
			CreatedAt:     unixMillisToTime(p.CreatedAt),
			UnrealizedPnl: decimalOrZero(p.UnrealizedPnl),
			UsedMargin:    decimalOrZero(p.UsedMargin),
			NetFunding:    decimalOrZero(p.NetFunding),
			Known:         true,
		}
	}
	return out
}

// renderClosePositionCard turns a pre-close snapshot + post-fill
// result into a rendered PnL card. When the snapshot is unknown
// or the fill didn't populate price/qty fields, we render a
// degraded "trade submitted" card rather than fabricate numbers.
//
// Status mapping:
//   - FILLED with known entry/exit → Positive / Negative / Flat
//     based on sign(exit-entry)*side.
//   - Non-FILLED (ACCEPTED, CANCELLED, REJECTED) → Neutral. There
//     is no realized PnL to show yet, so the card's job shifts
//     from "summarize outcome" to "acknowledge submission".
//   - Zero-length close quantity → no card at all (caller already
//     errored upstream, but defense-in-depth).
func renderClosePositionCard(snapshot closePositionSnapshot, result closePositionOutput, closeQty decimal.Decimal) string {
	if closeQty.IsZero() {
		return ""
	}
	if result.Status != "FILLED" {
		return renderCloseAcknowledgementCard(snapshot, result, closeQty)
	}
	exitPrice := decimalOrZero(result.AvgPrice)
	if exitPrice.IsZero() {
		return renderCloseAcknowledgementCard(snapshot, result, closeQty)
	}

	symbol := resolveCardSymbol(snapshot.Symbol, result.Symbol)
	sideLabel := strings.ToUpper(snapshot.Side)
	if sideLabel == "" {
		sideLabel = "FLAT"
	}

	realizedPnL, pctMove, directionArrow := computeCloseEconomics(snapshot, exitPrice, closeQty)
	realizedPnLFloat, _ := realizedPnL.Float64()
	status := cards.SignedStatus(realizedPnLFloat)

	titleDirection := directionArrow
	if titleDirection == "" {
		titleDirection = "▲"
		if sideLabel == "SHORT" {
			titleDirection = "▼"
		}
	}
	title := "CLOSED " + sideLabel + " " + titleDirection + " " + symbol

	heldFor := ""
	openedAt := ""
	if !snapshot.CreatedAt.IsZero() {
		heldFor = cards.Duration(time.Since(snapshot.CreatedAt))
		openedAt = "Opened: " + cards.TimestampUTC(snapshot.CreatedAt)
	}

	notional := exitPrice.Mul(closeQty)
	pnlDollars := realizedPnLFloat
	pnlPct := pctMove
	entryFloat, _ := snapshot.EntryPrice.Float64()
	exitFloat, _ := exitPrice.Float64()
	notionalFloat, _ := notional.Float64()
	fundingFloat, _ := snapshot.NetFunding.Float64()

	priceDelta := exitFloat - entryFloat
	if sideLabel == "SHORT" {
		priceDelta = entryFloat - exitFloat
	}
	priceDeltaArrow := glyphsForDelta(priceDelta)
	priceDeltaStr := priceDeltaArrow + " " + cards.SignedUSD(priceDelta, priceDecimals(exitFloat)) +
		"   (" + cards.Percent(pnlPct, 2) + ")"

	// The realized-PnL row's hint carries a second glyph (🟢 / 🔴 /
	// ⚪) that reinforces the header status for readers who scan
	// bottom-up. We pick it from the computed Status so a future
	// theme swap in cards/theme.go doesn't have to chase
	// per-card hardcoded strings.
	pnlRowHint := cards.Percent(pnlPct, 2)
	switch status {
	case cards.StatusPositive:
		pnlRowHint = "🟢 " + pnlRowHint
	case cards.StatusNegative:
		pnlRowHint = "🔴 " + pnlRowHint
	case cards.StatusFlat:
		pnlRowHint = "⚪ " + pnlRowHint
	}

	card := cards.Card{
		Status: status,
		Title:  title,
		Sections: []cards.Section{
			{Rows: []cards.Row{
				{Label: "Held for:", Value: firstNonEmptyLocal(heldFor, "—"), Hint: openedAt},
			}},
			{Rows: []cards.Row{
				{Label: "Side:", Value: sideLabel + " " + titleDirection},
				{Label: "Quantity:", Value: formatQuantity(closeQty) + " " + baseAsset(symbol), Hint: "Notional: " + cards.USD(notionalFloat, 2)},
				{Label: "Entry:", Value: cards.USD(entryFloat, priceDecimals(entryFloat))},
				{Label: "Exit:", Value: cards.USD(exitFloat, priceDecimals(exitFloat)), Hint: priceDeltaStr},
			}},
			{Rows: []cards.Row{
				{Label: "Realized PnL:", Value: cards.SignedUSD(pnlDollars, pnlDecimals(pnlDollars)), Hint: pnlRowHint},
			}},
		},
	}

	if !snapshot.NetFunding.IsZero() {
		fundingDirection := "longs paid shorts"
		if fundingFloat > 0 {
			fundingDirection = "shorts paid longs"
		}
		card.Footnote = "Funding " + signedFundingPrefix(fundingFloat) + cards.USD(absFloat(fundingFloat), pnlDecimals(fundingFloat)) +
			"   " + fundingNote(snapshot.CreatedAt, fundingDirection)
	}

	return card.Render()
}

// renderCloseAcknowledgementCard is the degraded variant for
// closes that didn't cleanly fill (resting, cancelled, rejected)
// or where we couldn't capture a pre-close snapshot. It's
// intentionally smaller: a one-section acknowledgement so the
// human sees *something* informative without us fabricating PnL.
func renderCloseAcknowledgementCard(snapshot closePositionSnapshot, result closePositionOutput, closeQty decimal.Decimal) string {
	symbol := resolveCardSymbol(snapshot.Symbol, result.Symbol)
	sideLabel := strings.ToUpper(snapshot.Side)
	if sideLabel == "" {
		sideLabel = "POSITION"
	}

	status := cards.StatusNeutral
	statusText := result.Status
	switch result.Status {
	case "FILLED":
		status = cards.StatusPositive
	case "ACCEPTED":
		status = cards.StatusNeutral
		statusText = "CLOSE ACCEPTED"
	case "CANCELLED":
		status = cards.StatusWarning
		statusText = "CLOSE CANCELLED"
	case "REJECTED":
		status = cards.StatusNegative
		statusText = "CLOSE REJECTED"
	default:
		if statusText == "" {
			statusText = "SUBMITTED"
		}
	}

	title := statusText + " " + sideLabel + " " + symbol

	rows := []cards.Row{
		{Label: "Symbol:", Value: symbol},
		{Label: "Requested:", Value: formatQuantity(closeQty) + " " + baseAsset(symbol)},
	}
	if result.AvgPrice != "" && result.Status == "FILLED" {
		avg := decimalOrZero(result.AvgPrice)
		avgFloat, _ := avg.Float64()
		rows = append(rows, cards.Row{Label: "Fill price:", Value: cards.USD(avgFloat, priceDecimals(avgFloat))})
	}
	if result.Message != "" {
		rows = append(rows, cards.Row{Label: "Detail:", Value: truncateForRow(result.Message, 48)})
	}

	return cards.Card{
		Status:   status,
		Title:    title,
		Sections: []cards.Section{{Rows: rows}},
	}.Render()
}

// computeCloseEconomics returns (realizedPnL, pctMove, arrow) for
// a long or short close. The arrow reflects the trade's
// direction on this side: green-up for a winning close, red-down
// for a losing close. We use the position side to correctly
// flip the sign: closing a SHORT at a lower-than-entry price is
// a win, closing a LONG at a lower-than-entry price is a loss.
func computeCloseEconomics(snapshot closePositionSnapshot, exitPrice, closeQty decimal.Decimal) (pnl decimal.Decimal, pctMove float64, arrow string) {
	if snapshot.EntryPrice.IsZero() {
		return decimal.Zero, 0, "◆"
	}
	delta := exitPrice.Sub(snapshot.EntryPrice)
	if strings.EqualFold(snapshot.Side, "short") {
		delta = snapshot.EntryPrice.Sub(exitPrice)
	}
	pnl = delta.Mul(closeQty)
	pnlFloat, _ := pnl.Float64()
	entryFloat, _ := snapshot.EntryPrice.Float64()
	if entryFloat != 0 {
		deltaFloat, _ := delta.Float64()
		pctMove = (deltaFloat / entryFloat) * 100
	}
	arrow = "◆"
	switch {
	case pnlFloat > 0:
		arrow = "▲"
	case pnlFloat < 0:
		arrow = "▼"
	}
	return pnl, pctMove, arrow
}

// decimalOrZero parses a REST string decimal, returning zero for
// empty / malformed inputs. REST payloads use strings for all
// numeric fields so floats don't lose precision on the wire;
// upstream has been observed to send "" for fields that haven't
// settled yet (e.g. NetFunding on a brand-new position).
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
// time.Time. Zero-valued inputs stay zero so the card can
// recognise "no timestamp known".
func unixMillisToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// resolveCardSymbol prefers the snapshot symbol (the real, canonical
// market symbol from getPositions) over the result symbol. Both
// should match, but snapshot.Symbol preserves the casing as the
// exchange returned it.
func resolveCardSymbol(snapshot, fromResult string) string {
	if snapshot != "" {
		return snapshot
	}
	if fromResult != "" {
		return fromResult
	}
	return "UNKNOWN"
}

// baseAsset extracts the base from a symbol like "BTC-USDT" ->
// "BTC". Used as the unit label on the Quantity row. Falls back
// to a generic "contracts" label when the symbol isn't in the
// expected form.
func baseAsset(symbol string) string {
	for _, sep := range []string{"-", "/", "_"} {
		if idx := strings.Index(symbol, sep); idx > 0 {
			return symbol[:idx]
		}
	}
	return "contracts"
}

// formatQuantity renders a decimal quantity with up to 6 decimal
// places, trimming trailing zeros. A raw Decimal.String() would
// emit "0.1000000000" which is visually heavy and hides the
// actual precision.
func formatQuantity(q decimal.Decimal) string {
	s := q.StringFixed(6)
	// Trim trailing zeros, then a trailing dot.
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

// priceDecimals picks a reasonable decimal count for a USD price.
// $76,300 reads as 0 decimals; $76.30 reads as 2 decimals; $0.0042
// reads as 4 decimals. Prevents sub-dollar crypto prices from
// looking like zeros in the card.
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

func signedFundingPrefix(v float64) string {
	if v > 0 {
		return "earned +"
	}
	return "paid -"
}

func fundingNote(opened time.Time, direction string) string {
	note := "(" + direction + ")"
	if !opened.IsZero() {
		return "(" + cards.Duration(time.Since(opened)) + " hold — " + direction + ")"
	}
	return note
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

// renderCloseAllPositionsCard builds a single portfolio summary
// card for a close_all_positions batch. Each row condenses one
// closed position to entry→exit + realized PnL so a trader can
// see the outcome of flattening N positions at a glance without
// us emitting N full PnL cards (which would burn thousands of
// tokens for a common operation).
//
// The header reflects the net realized PnL across all legs: green
// when the batch net is positive, red when negative, yellow when
// any leg rejected, flat otherwise.
func renderCloseAllPositionsCard(preSnapshots map[string]closePositionSnapshot, out closeAllPositionsOutput, quantities []string) string {
	if len(out.Positions) == 0 {
		return ""
	}

	var netPnL float64
	anyRejected := false
	rows := make([]cards.Row, 0, len(out.Positions))

	for i, item := range out.Positions {
		canonical := strings.ToUpper(strings.TrimSpace(item.Symbol))
		snapshot := preSnapshots[canonical]
		quantityStr := ""
		if i < len(quantities) {
			quantityStr = quantities[i]
		}
		closeQty := decimalOrZero(quantityStr)
		if closeQty.IsZero() {
			closeQty = decimalOrZero(item.ClosedQuantity)
		}

		sideLabel := strings.ToUpper(firstNonEmptyLocal(item.Side, snapshot.Side))
		sideBadge := "?"
		switch sideLabel {
		case "LONG":
			sideBadge = "L▲"
		case "SHORT":
			sideBadge = "S▼"
		}

		leftLabel := sideBadge + " " + item.Symbol

		if item.Order.Status != "FILLED" || item.Order.AvgPrice == "" {
			anyRejected = anyRejected || item.Order.Status == "REJECTED"
			statusMark := item.Order.Status
			if statusMark == "" {
				statusMark = "PENDING"
			}
			rows = append(rows, cards.Row{
				Label: leftLabel,
				Value: formatQuantity(closeQty) + " " + baseAsset(item.Symbol),
				Hint:  statusMark,
			})
			continue
		}

		exit := decimalOrZero(item.Order.AvgPrice)
		pnl, pctMove, arrow := computeCloseEconomics(snapshot, exit, closeQty)
		pnlFloat, _ := pnl.Float64()
		netPnL += pnlFloat

		exitFloat, _ := exit.Float64()
		entryFloat, _ := snapshot.EntryPrice.Float64()
		value := formatQuantity(closeQty) + " " + baseAsset(item.Symbol)
		if entryFloat > 0 {
			value = value + " " + cards.USD(entryFloat, priceDecimals(entryFloat)) + "→" + cards.USD(exitFloat, priceDecimals(exitFloat))
		} else {
			value = value + " →" + cards.USD(exitFloat, priceDecimals(exitFloat))
		}
		hint := arrow + " " + cards.SignedUSD(pnlFloat, pnlDecimals(pnlFloat)) +
			" (" + cards.Percent(pctMove, 2) + ")"
		rows = append(rows, cards.Row{
			Label: leftLabel,
			Value: value,
			Hint:  hint,
		})
	}

	status := cards.SignedStatus(netPnL)
	if anyRejected {
		status = cards.StatusWarning
	}

	netHint := ""
	switch {
	case anyRejected:
		netHint = "⚠ one or more legs rejected"
	case status == cards.StatusPositive:
		netHint = "🟢 net"
	case status == cards.StatusNegative:
		netHint = "🔴 net"
	default:
		netHint = "⚪ net"
	}

	title := "CLOSED " + pluralize(len(out.Positions), "POSITION", "POSITIONS")

	return cards.Card{
		Status: status,
		Title:  title,
		Sections: []cards.Section{
			{Rows: rows},
			{Rows: []cards.Row{
				{Label: "Net realized PnL:", Value: cards.SignedUSD(netPnL, pnlDecimals(netPnL)), Hint: netHint},
			}},
		},
	}.Render()
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// glyphsForDelta is a tool-side mirror of the cards package's
// internal arrow selector. We duplicate the thin logic here to
// avoid widening the cards public surface just for one exported
// helper.
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
