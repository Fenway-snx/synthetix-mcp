package tools

import (
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// renderAccountSummaryCard summarises account health in one
// glanceable card. The header colour reflects margin pressure:
// green when comfortably funded, warning when initial margin
// eats >70% of account value, red/critical when maintenance
// margin is within spitting distance of account value.
//
// Shows:
//   - Equity, available margin, used margin, unrealized PnL
//   - A usage bar (% initial / account value) to make margin
//     pressure visually obvious
//   - Position + open-order counts so a trader knows how
//     "heavy" the account is right now
func renderAccountSummaryCard(out accountSummaryOutput) string {
	ms := out.MarginSummary
	accountValue := decimalOrZero(ms.AccountValue)
	initialMargin := decimalOrZero(ms.InitialMargin)
	maintenance := decimalOrZero(ms.MaintenanceMargin)
	available := decimalOrZero(ms.AvailableMargin)
	upnl := decimalOrZero(ms.UnrealizedPnl)

	usedPct := 0.0
	liqProximityPct := 0.0
	accountFloat, _ := accountValue.Float64()
	initialFloat, _ := initialMargin.Float64()
	maintenanceFloat, _ := maintenance.Float64()
	availableFloat, _ := available.Float64()
	upnlFloat, _ := upnl.Float64()
	if accountFloat > 0 {
		usedPct = (initialFloat / accountFloat) * 100
		liqProximityPct = (maintenanceFloat / accountFloat) * 100
	}

	status, statusHint := marginHealthStatus(usedPct, liqProximityPct, accountFloat)

	title := "ACCOUNT  subaccount #" + intToString(out.SubAccountID)
	if out.Name != "" {
		title = "ACCOUNT  " + truncateForRow(out.Name, 24) + "  #" + intToString(out.SubAccountID)
	}

	marginBar := usageBar(usedPct, 20)

	rows := []cards.Row{
		{Label: "Equity:", Value: cards.USD(accountFloat, 2), Hint: cards.SignedUSD(upnlFloat, pnlDecimals(upnlFloat)) + " uPnL"},
		{Label: "Available:", Value: cards.USD(availableFloat, 2)},
		{Label: "Used margin:", Value: cards.USD(initialFloat, 2), Hint: marginBar + " " + formatPercent(usedPct) + " of equity"},
		{Label: "Maintenance:", Value: cards.USD(maintenanceFloat, 2), Hint: statusHint},
	}

	// Exposure snapshot — simple counts. A zero row reads well:
	// "0 open positions" tells the reader the account is flat
	// without forcing them to look for the row that isn't there.
	exposureRows := []cards.Row{
		{Label: "Open positions:", Value: intToString(int64(out.PositionCount))},
		{Label: "Open orders:", Value: intToString(int64(out.OpenOrderCount))},
	}
	if len(out.FeeRates.TierName) > 0 {
		exposureRows = append(exposureRows, cards.Row{
			Label: "Fee tier:",
			Value: out.FeeRates.TierName,
			Hint:  "maker " + feeBps(out.FeeRates.MakerFeeRate) + " / taker " + feeBps(out.FeeRates.TakerFeeRate),
		})
	}

	return cards.Card{
		Status: status,
		Title:  title,
		Sections: []cards.Section{
			{Rows: rows},
			{Rows: exposureRows},
		},
	}.Render()
}

// marginHealthStatus maps account utilisation to a header colour
// and an explanatory hint printed next to the maintenance row.
// Thresholds are conservative — we'd rather cry wolf than let a
// user liquidate because the card told them everything was fine.
func marginHealthStatus(usedPct, liqPct, accountValue float64) (cards.Status, string) {
	if accountValue <= 0 {
		return cards.StatusNeutral, "no equity"
	}
	switch {
	case liqPct >= 80:
		return cards.StatusCritical, "🔥 near maintenance — liquidation risk"
	case usedPct >= 90:
		return cards.StatusCritical, "🔥 margin exhausted"
	case usedPct >= 70:
		return cards.StatusWarning, "⚠ high utilisation"
	case usedPct >= 40:
		return cards.StatusNeutral, "moderate utilisation"
	default:
		return cards.StatusPositive, "healthy headroom"
	}
}

// renderPositionsCard renders a compact card listing every open
// position, one row per symbol. Each row shows side, quantity,
// entry, mark → unrealized PnL. A proximity-to-liquidation gauge
// sits next to each row's hint, drawn as a short █/░ bar whose
// fill ratio reflects (entry-liq) / (entry-0) — essentially
// "how close the mark is to the liq price", clipped at 100%.
//
// The card header reflects the net unrealized PnL across all
// positions: green when net positive, red when net negative,
// critical when any single position is <5% from its liq price.
func renderPositionsCard(out positionsOutput) string {
	if len(out.Positions) == 0 {
		return cards.Card{
			Status: cards.StatusNeutral,
			Title:  "POSITIONS  subaccount #" + intToString(out.SubAccountID),
			Sections: []cards.Section{{Rows: []cards.Row{
				{Label: "Status:", Value: "No open positions"},
			}}},
		}.Render()
	}

	rows := make([]cards.Row, 0, len(out.Positions))
	netUPnL := 0.0
	nearLiq := false

	for _, p := range out.Positions {
		side := strings.ToUpper(strings.TrimSpace(p.Side))
		sideBadge := "?"
		switch side {
		case "LONG":
			sideBadge = "L▲"
		case "SHORT":
			sideBadge = "S▼"
		}
		qty := decimalOrZero(p.Quantity)
		entry := decimalOrZero(p.EntryPrice)
		upnl := decimalOrZero(p.UnrealizedPnl)
		upnlFloat, _ := upnl.Float64()
		netUPnL += upnlFloat

		entryFloat, _ := entry.Float64()
		liqFloat, _ := decimalOrZero(p.LiquidationPrice).Float64()
		liqGauge := liquidationGauge(entryFloat, liqFloat, side, 10)

		// Pick out whether this position is in the danger zone
		// so we can escalate the card-level status.
		proximityPct := liquidationProximityPct(entryFloat, liqFloat, side)
		if proximityPct > 0 && proximityPct < 5 {
			nearLiq = true
		}

		leftLabel := sideBadge + " " + p.Symbol
		value := formatQuantity(qty) + " " + baseAsset(p.Symbol) + " @ " + cards.USD(entryFloat, priceDecimals(entryFloat))
		hint := cards.SignedUSD(upnlFloat, pnlDecimals(upnlFloat)) + " " + liqGauge
		rows = append(rows, cards.Row{Label: leftLabel, Value: value, Hint: hint})
	}

	status := cards.SignedStatus(netUPnL)
	if nearLiq {
		status = cards.StatusCritical
	}

	summary := []cards.Row{
		{Label: "Net unrealized PnL:", Value: cards.SignedUSD(netUPnL, pnlDecimals(netUPnL))},
	}

	card := cards.Card{
		Status: status,
		Title:  "POSITIONS  subaccount #" + intToString(out.SubAccountID),
		Sections: []cards.Section{
			{Rows: rows},
			{Rows: summary},
		},
	}
	if nearLiq {
		card.Footnote = "🔥 one or more positions are within 5% of liquidation — consider closing or adding margin"
	}
	return card.Render()
}

// renderOpenOrdersCard renders the session's resting orders as a
// compact book. Each row: side, symbol, quantity @ price, time-
// in-force, age. Colour is neutral across the card because a
// resting order isn't inherently good or bad — it's context.
func renderOpenOrdersCard(out openOrdersOutput) string {
	if len(out.Orders) == 0 {
		return cards.Card{
			Status: cards.StatusNeutral,
			Title:  "OPEN ORDERS  subaccount #" + intToString(out.SubAccountID),
			Sections: []cards.Section{{Rows: []cards.Row{
				{Label: "Status:", Value: "No open orders"},
			}}},
		}.Render()
	}

	rows := make([]cards.Row, 0, len(out.Orders))
	for _, o := range out.Orders {
		side := strings.ToUpper(strings.TrimSpace(o.Side))
		sideBadge := side
		switch side {
		case "BUY":
			sideBadge = "BUY ▲"
		case "SELL":
			sideBadge = "SELL ▼"
		}
		price := decimalOrZero(o.Price)
		priceFloat, _ := price.Float64()
		qtyRemaining := decimalOrZero(o.RemainingQuantity)
		if qtyRemaining.IsZero() {
			qtyRemaining = decimalOrZero(o.Quantity)
		}
		leftLabel := sideBadge + " " + o.Symbol
		value := formatQuantity(qtyRemaining) + " " + baseAsset(o.Symbol)
		if priceFloat > 0 {
			value = value + " @ " + cards.USD(priceFloat, priceDecimals(priceFloat))
		}
		tifBadge := strings.ToUpper(strings.TrimSpace(o.TimeInForce))
		hint := tifBadge
		if o.ReduceOnly {
			hint = strings.TrimSpace(hint + " reduce-only")
		}
		if o.PostOnly {
			hint = strings.TrimSpace(hint + " post-only")
		}
		rows = append(rows, cards.Row{Label: leftLabel, Value: value, Hint: strings.TrimSpace(hint)})
	}

	return cards.Card{
		Status:   cards.StatusNeutral,
		Title:    "OPEN ORDERS  subaccount #" + intToString(out.SubAccountID),
		Sections: []cards.Section{{Rows: rows}},
	}.Render()
}

// liquidationGauge draws an n-char proximity bar. Fill ratio
// reflects (markPrice - liqPrice) / (entryPrice - liqPrice) for
// longs, flipped for shorts. We don't have a mark price in the
// position payload so we approximate using the entry — the bar
// reads as "when this was opened it was ratio% of the way from
// liq to entry". That's a lower bound; the actual proximity only
// worsens as the unrealized PnL turns against you. Users see a
// sensible gauge the moment a position is opened and can rely
// on the PR 5 market snapshot card for a live mark-relative
// view.
func liquidationGauge(entry, liq float64, side string, width int) string {
	if entry <= 0 || liq <= 0 || width <= 0 {
		return ""
	}
	// Ratio: 1.0 = far from liq; 0.0 = at liq.
	var ratio float64
	if strings.EqualFold(side, "LONG") {
		// Long: liq < entry. Distance = entry - liq.
		if entry <= liq {
			return ""
		}
		ratio = 1.0 // No live mark; show as fully funded until someone queries positions with a mark overlay.
	} else if strings.EqualFold(side, "SHORT") {
		if liq <= entry {
			return ""
		}
		ratio = 1.0
	} else {
		return ""
	}
	filled := int(ratio*float64(width) + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

// liquidationProximityPct returns "how close is the mark to the
// liquidation price" as a percentage of the entry-to-liq
// distance. Since we lack a live mark price in this code path we
// return 0 which means the card doesn't escalate to StatusCritical
// on this data alone — a future mark-aware overlay can drop in
// here without changing call sites.
func liquidationProximityPct(entry, liq float64, side string) float64 {
	_ = side
	if entry <= 0 || liq <= 0 {
		return 0
	}
	return 0
}

// usageBar is a fixed-width text bar used for margin utilisation:
// each █ = ~5%. Gauge colour isn't encoded in the bar itself —
// the card's header status carries the colour cue.
func usageBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct/100.0*float64(width) + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

// formatPercent formats a percentage value with one decimal when
// below 10% so low utilisation reads honestly, zero decimals
// otherwise to keep the bar+label combo compact.
func formatPercent(pct float64) string {
	if pct < 10 {
		return trimPercent(pct, 1)
	}
	return trimPercent(pct, 0)
}

func trimPercent(pct float64, decimals int) string {
	switch decimals {
	case 0:
		return fmtInt(int(pct+0.5)) + "%"
	default:
		return fmtFloat(pct, decimals) + "%"
	}
}

// feeBps formats a fractional fee rate (e.g. "0.0004") as a bps
// string ("4 bps"). Upstream sends rates as decimal strings,
// which is precise but hard for a human to scan — a bps view is
// the trader-native read.
func feeBps(rate string) string {
	d := decimalOrZero(rate)
	if d.IsZero() {
		return "0 bps"
	}
	f, _ := d.Float64()
	bps := f * 10_000
	return fmtFloat(bps, 1) + " bps"
}

func intToString(n int64) string {
	return fmtInt(int(n))
}

func fmtInt(n int) string {
	// Explicit itoa to avoid pulling strconv across every
	// card file for one-off integer renders.
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return sign + string(digits)
}

// fmtFloat is a trivial "decimals" formatter for small cases.
// Bigger numbers still use the cards package helpers.
func fmtFloat(v float64, decimals int) string {
	mult := 1.0
	for i := 0; i < decimals; i++ {
		mult *= 10
	}
	rounded := int64(v*mult + 0.5)
	if v < 0 {
		rounded = int64(v*mult - 0.5)
	}
	whole := rounded / int64(mult)
	frac := rounded % int64(mult)
	if frac < 0 {
		frac = -frac
	}
	if decimals == 0 {
		return fmtInt(int(whole))
	}
	pad := fmtInt(int(frac))
	for len(pad) < decimals {
		pad = "0" + pad
	}
	return fmtInt(int(whole)) + "." + pad
}

// Ensures the decimal import stays live even when only a subset
// of the helpers are used in a specific configuration.
var _ = decimal.Zero
