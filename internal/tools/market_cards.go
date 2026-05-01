package tools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// Renders a compact 80-column market dashboard.
// It highlights price, liquidity, funding, and constraints for pre-trade checks.
func renderMarketSummaryCard(out marketSummaryOutput) string {
	if !cards.Enabled() {
		return ""
	}
	m := out.Market
	p := out.Prices
	s := out.Summary

	mark := decimalOrZero(p.MarkPrice)
	index := decimalOrZero(p.IndexPrice)
	last := decimalOrZero(s.LastTradedPrice)
	prev := decimalOrZero(s.PrevDayPrice)
	bestBid := decimalOrZero(s.BestBidPrice)
	bestAsk := decimalOrZero(s.BestAskPrice)
	volBase := decimalOrZero(s.Volume24h)
	volQuote := decimalOrZero(s.QuoteVolume24h)
	openInterest := decimalOrZero(out.OpenInterest)

	markF, _ := mark.Float64()
	indexF, _ := index.Float64()
	lastF, _ := last.Float64()
	prevF, _ := prev.Float64()
	volBaseF, _ := volBase.Float64()
	volQuoteF, _ := volQuote.Float64()
	oiF, _ := openInterest.Float64()

	var changePct float64
	if prevF > 0 && lastF > 0 {
		changePct = (lastF - prevF) / prevF * 100
	}
	status := cards.SignedStatus(changePct)

	priceDec := priceDecimals(markF)
	title := fmt.Sprintf("%s  mark %s", m.Symbol, cards.USD(markF, priceDec))
	if !m.IsOpen {
		title = m.Symbol + "  CLOSED"
		status = cards.StatusWarning
	}

	spreadRow := cards.Row{Label: "Bid / ask:", Value: cards.USD(bidOrZero(bestBid), priceDec) + " / " + cards.USD(askOrZero(bestAsk), priceDec), Hint: spreadHint(bestBid, bestAsk)}
	changeValue := cards.Percent(changePct, 2)
	if prevF <= 0 || lastF <= 0 {
		changeValue = "—"
	}
	changeHint := ""
	if prevF > 0 {
		changeHint = "prev close " + cards.USD(prevF, priceDec)
	}
	priceSection := []cards.Row{
		{Label: "Mark / index:", Value: cards.USD(markF, priceDec) + " / " + cards.USD(indexF, priceDec), Hint: basisHint(markF, indexF)},
		{Label: "Last trade:", Value: cards.USD(lastF, priceDec), Hint: changeValue + " 24h"},
		spreadRow,
		{Label: "24h range:", Value: sparkline24h(prevF, lastF, priceDec), Hint: changeHint},
	}

	volumeHint := ""
	if volBaseF > 0 {
		volumeHint = compactNumber(volBaseF) + " " + baseAsset(m.Symbol)
	}
	liquiditySection := []cards.Row{
		{Label: "24h volume:", Value: "$" + compactNumber(volQuoteF), Hint: volumeHint},
		{Label: "Open interest:", Value: "$" + compactNumber(oiF), Hint: ""},
	}

	fundingSection := fundingRows(out.FundingRate, m)

	constraintsSection := []cards.Row{
		{Label: "Tick / min qty:", Value: m.TickSize + " / " + m.MinTradeAmount, Hint: "min notional $" + compactNumber(decimalFloat(m.MinNotionalValue))},
		{Label: "Max leverage:", Value: fmt.Sprintf("%dx", m.DefaultLeverage), Hint: ""},
	}

	sectionList := []cards.Section{
		{Rows: priceSection},
		{Rows: liquiditySection},
	}
	if len(fundingSection) > 0 {
		sectionList = append(sectionList, cards.Section{Rows: fundingSection})
	}
	sectionList = append(sectionList, cards.Section{Rows: constraintsSection})

	return cards.Card{
		Status:   status,
		Title:    "MARKET  " + title,
		Sections: sectionList,
	}.Render()
}

// Renders a top-N market leaderboard sorted by 24h quote volume.
// Rows show price, change, and a compact liquidity bar.
func renderListMarketsCard(out listMarketsOutput) string {
	if !cards.Enabled() {
		return ""
	}
	if len(out.Markets) == 0 {
		return cards.Card{
			Status:   cards.StatusWarning,
			Title:    "MARKETS",
			Sections: []cards.Section{{Rows: []cards.Row{{Label: "No markets returned.", Value: "", Hint: ""}}}},
		}.Render()
	}

	type marketRow struct {
		symbol    string
		markPrice float64
		change24h float64
		volQuote  float64
		isOpen    bool
	}
	rows := make([]marketRow, 0, len(out.Markets))
	for _, m := range out.Markets {
		rows = append(rows, marketRow{
			symbol: m.Symbol,
			isOpen: m.IsOpen,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].volQuote != rows[j].volQuote {
			return rows[i].volQuote > rows[j].volQuote
		}
		return rows[i].symbol < rows[j].symbol
	})

	maxRows := len(rows)
	if maxRows > 15 {
		maxRows = 15
	}

	cardRows := make([]cards.Row, 0, maxRows)
	for _, r := range rows[:maxRows] {
		badge := "◆"
		if !r.isOpen {
			badge = "✕"
		}
		label := badge + " " + r.symbol
		value := r.symbol
		if !r.isOpen {
			value = "closed"
		}
		cardRows = append(cardRows, cards.Row{Label: label, Value: value, Hint: ""})
	}

	footer := []cards.Row{
		{Label: fmt.Sprintf("%d markets", len(out.Markets)), Value: "", Hint: "get_market_summary <symbol> for detail"},
	}
	return cards.Card{
		Status: cards.StatusNeutral,
		Title:  "MARKETS",
		Sections: []cards.Section{
			{Rows: cardRows},
			{Rows: footer},
		},
	}.Render()
}

func fundingRows(rate *FundingRateEntry, m MarketOutput) []cards.Row {
	if rate == nil {
		return nil
	}
	est := decimalFloat(rate.EstimatedFundingRate)
	cap := decimalFloat(m.FundingRateCap)
	if cap <= 0 {
		cap = 0.0005
	}
	barWidth := 26
	bar := fundingBar(est, cap, barWidth)

	hint := "longs pay shorts"
	if est < 0 {
		hint = "shorts pay longs"
	}
	if est == 0 {
		hint = "flat"
	}
	return []cards.Row{
		{Label: "Est. funding:", Value: cards.PercentFraction(est, 4), Hint: hint},
		{Label: "Funding:", Value: bar, Hint: "cap ±" + cards.PercentFraction(cap, 3)},
	}
}

// Renders a centered funding gauge that fills toward the rate direction.
func fundingBar(rate, cap float64, width int) string {
	if width < 7 {
		width = 7
	}
	if width%2 == 0 {
		width++
	}
	half := width / 2
	cells := make([]rune, width)
	for i := range cells {
		cells[i] = '·'
	}
	cells[half] = '│'
	if cap <= 0 {
		return string(cells)
	}
	ratio := rate / cap
	if ratio > 1 {
		ratio = 1
	}
	if ratio < -1 {
		ratio = -1
	}
	filled := int(float64(half) * absFloat(ratio))
	if rate > 0 {
		for i := 1; i <= filled; i++ {
			cells[half+i] = '█'
		}
	} else if rate < 0 {
		for i := 1; i <= filled; i++ {
			cells[half-i] = '█'
		}
	}
	return string(cells)
}

// Renders a minimal previous-to-latest segment for 24-hour movement.
func sparkline24h(prev, last float64, decimals int) string {
	if prev <= 0 || last <= 0 {
		return "—"
	}
	arrow := "▲"
	if last < prev {
		arrow = "▼"
	} else if last == prev {
		arrow = "◆"
	}
	return cards.USD(prev, decimals) + " → " + arrow + " " + cards.USD(last, decimals)
}

func basisHint(mark, index float64) string {
	if mark <= 0 || index <= 0 {
		return ""
	}
	basis := (mark - index) / index * 100
	if absFloat(basis) < 0.005 {
		return "basis flat"
	}
	return "basis " + cards.Percent(basis, 3)
}

func spreadHint(bid, ask decimal.Decimal) string {
	if bid.IsZero() || ask.IsZero() {
		return ""
	}
	spread := ask.Sub(bid)
	mid := bid.Add(ask).Div(decimal.NewFromInt(2))
	if mid.IsZero() {
		return ""
	}
	bps, _ := spread.Div(mid).Mul(decimal.NewFromInt(10000)).Float64()
	spreadFloat, _ := spread.Float64()
	return fmt.Sprintf("spread %s (%.1f bps)", cards.USD(spreadFloat, priceDecimals(spreadFloat)), bps)
}

func bidOrZero(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

func askOrZero(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

func decimalFloat(s string) float64 {
	f, _ := decimalOrZero(s).Float64()
	return f
}

// compactNumber turns 12,345,678 into "12.3M", 456,789 into "456.8K",
// keeping cards narrow while preserving order of magnitude. Used for
// notionals and volumes where the last few dollars are noise.
func compactNumber(v float64) string {
	abs := absFloat(v)
	switch {
	case abs >= 1_000_000_000:
		return trimTrailingFractionZeros(fmt.Sprintf("%.2f", v/1_000_000_000)) + "B"
	case abs >= 1_000_000:
		return trimTrailingFractionZeros(fmt.Sprintf("%.2f", v/1_000_000)) + "M"
	case abs >= 10_000:
		return trimTrailingFractionZeros(fmt.Sprintf("%.1f", v/1_000)) + "K"
	case abs >= 1_000:
		return trimTrailingFractionZeros(fmt.Sprintf("%.2f", v/1_000)) + "K"
	case abs >= 1:
		return fmt.Sprintf("%.0f", v)
	case abs == 0:
		return "0"
	default:
		return trimTrailingFractionZeros(fmt.Sprintf("%.4f", v))
	}
}

// trimTrailingFractionZeros strips trailing zeros (and a lone dot)
// from a decimal string, preserving the value. Used by compact
// formatters that don't know whether the fraction is meaningful.
func trimTrailingFractionZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}
