package tools

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/synthetixio/synthetix-go/types"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// Captures pre-close position context for PnL cards.
// Eager capture avoids post-close races where filled positions disappear.
type closePositionSnapshot struct {
	Symbol        string
	Side          string
	Quantity      decimal.Decimal
	EntryPrice    decimal.Decimal
	CreatedAt     time.Time
	UnrealizedPnl decimal.Decimal
	UsedMargin    decimal.Decimal
	NetFunding    decimal.Decimal
	// Marks whether pre-close data was found.
	// False triggers a degraded card instead of zeroed PnL.
	Known bool
}

// Reads pre-close position context for one symbol.
// Missing data degrades the card without blocking the trade.
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

// Batch-fetches pre-close position context for multiple symbols.
// The result is keyed by canonical upper-case symbol.
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

// Renders a PnL card from pre-close context and post-fill output.
// Unknown snapshots or missing fill data produce a degraded acknowledgement.
// Non-filled states stay neutral because realized PnL is not final.
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

	// Mirror the header status in the PnL row for bottom-up scanning.
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

// Renders the degraded close card when realized PnL is unavailable.
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

// Computes realized PnL, percent move, and direction glyph for a close.
// Shorts invert the price delta so lower exits are wins.
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

// Prefers the pre-close symbol casing over the fill result symbol.
func resolveCardSymbol(snapshot, fromResult string) string {
	if snapshot != "" {
		return snapshot
	}
	if fromResult != "" {
		return fromResult
	}
	return "UNKNOWN"
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

// Renders one portfolio summary card for a batched position close.
// Rows condense each leg to entry, exit, and realized PnL.
// The header reflects net PnL or warning when any leg rejects.
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
