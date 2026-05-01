package tools

import (
	"context"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
)

// Bundles parsed orderbook levels with the session's live orders.
// The renderer stays a pure transform over primitive types.
type orderbookCardInput struct {
	Symbol string
	Bids   []orderbookLevel
	Asks   []orderbookLevel
	MyBids []myOrderMark
	MyAsks []myOrderMark
}

type orderbookLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type myOrderMark struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
	Side     string // "BUY" / "SELL"
	OrderID  string // venue OR client id, for readability
}

// Fuses public orderbook levels with live orders for the same session.
// Public or unhydrated sessions simply render without "YOU" markers.
func buildOrderbookCardInput(
	ctx context.Context,
	sessionID string,
	subAccountID int64,
	snapshotManager *risksnapshot.Manager,
	symbol string,
	bids []priceLevelOutput,
	asks []priceLevelOutput,
) orderbookCardInput {
	out := orderbookCardInput{
		Symbol: symbol,
		Bids:   toOrderbookLevels(bids),
		Asks:   toOrderbookLevels(asks),
	}

	if snapshotManager == nil || sessionID == "" || subAccountID <= 0 {
		return out
	}

	snap, err := snapshotManager.EnsureHydrated(ctx, sessionID, subAccountID)
	if err != nil || snap == nil {
		return out
	}
	normSym := strings.ToUpper(strings.TrimSpace(symbol))
	for _, order := range snap.AllOpenOrders() {
		if !strings.EqualFold(strings.TrimSpace(order.Symbol), normSym) {
			continue
		}
		mark := myOrderMark{
			Price:    order.Price,
			Quantity: order.RemainingQuantity,
			Side:     strings.ToUpper(strings.TrimSpace(order.Side)),
			OrderID:  firstNonEmptyLocal(order.ClientOrderID, order.VenueOrderID),
		}
		if mark.Quantity.IsZero() {
			mark.Quantity = order.Quantity
		}
		if mark.Side == "BUY" {
			out.MyBids = append(out.MyBids, mark)
		} else if mark.Side == "SELL" {
			out.MyAsks = append(out.MyAsks, mark)
		}
	}
	return out
}

func toOrderbookLevels(levels []priceLevelOutput) []orderbookLevel {
	out := make([]orderbookLevel, 0, len(levels))
	for _, lvl := range levels {
		price := decimalOrZero(lvl.Price)
		qty := decimalOrZero(lvl.Quantity)
		if price.IsZero() || qty.IsZero() {
			continue
		}
		out = append(out, orderbookLevel{Price: price, Quantity: qty})
	}
	return out
}

// Renders asks, spread, and bids in the conventional ladder layout.
// Quantity bars share one scale across both sides for visual comparison.
// Live orders aggregate into one "YOU" marker per price level.
func renderOrderbookCard(in orderbookCardInput, depth int) string {
	if len(in.Bids) == 0 && len(in.Asks) == 0 {
		return ""
	}
	bestBid := bestPrice(in.Bids, true)
	bestAsk := bestPrice(in.Asks, false)
	mid := decimal.Zero
	spread := decimal.Zero
	spreadBps := 0.0
	if !bestBid.IsZero() && !bestAsk.IsZero() {
		mid = bestBid.Add(bestAsk).Div(decimal.NewFromInt(2))
		spread = bestAsk.Sub(bestBid)
		midFloat, _ := mid.Float64()
		spreadFloat, _ := spread.Float64()
		if midFloat > 0 {
			spreadBps = (spreadFloat / midFloat) * 10_000
		}
	}

	asksSorted := sortedLevels(in.Asks, false) // ascending by price
	bidsSorted := sortedLevels(in.Bids, true)  // descending by price

	if depth <= 0 {
		depth = 6
	}
	asksDisplay := topN(asksSorted, depth)
	bidsDisplay := topN(bidsSorted, depth)

	// Use one max across both sides so bar proportions stay honest.
	maxQty := decimal.Zero
	for _, lvl := range asksDisplay {
		if lvl.Quantity.GreaterThan(maxQty) {
			maxQty = lvl.Quantity
		}
	}
	for _, lvl := range bidsDisplay {
		if lvl.Quantity.GreaterThan(maxQty) {
			maxQty = lvl.Quantity
		}
	}

	myAsksByPrice := groupMarksByPrice(in.MyAsks)
	myBidsByPrice := groupMarksByPrice(in.MyBids)

	// Mid-price is context, not a win/loss outcome.
	titleStatus := cards.StatusNeutral

	midFloat, _ := mid.Float64()
	title := "ORDERBOOK  " + in.Symbol
	if !mid.IsZero() {
		title = title + "  mid " + cards.USD(midFloat, priceDecimals(midFloat))
	}
	rows := make([]cards.Row, 0, len(asksDisplay)+len(bidsDisplay)+3)

	// Show farthest ask first so the ladder reads like a depth chart.
	for i := len(asksDisplay) - 1; i >= 0; i-- {
		lvl := asksDisplay[i]
		rows = append(rows, orderbookLevelRow(lvl, maxQty, myAsksByPrice, "SELL"))
	}

	// Render the divider as a row because card sections have no raw dividers.
	spreadLine := "spread —"
	if !spread.IsZero() {
		spreadFloat, _ := spread.Float64()
		spreadLine = "spread  " + cards.USD(spreadFloat, priceDecimals(spreadFloat)) + "  (" + formatBps(spreadBps) + ")"
	}
	rows = append(rows, cards.Row{Label: "────", Value: spreadLine, Hint: "────"})

	// Bid side: best-bid on top, descending.
	for _, lvl := range bidsDisplay {
		rows = append(rows, orderbookLevelRow(lvl, maxQty, myBidsByPrice, "BUY"))
	}

	totalMyBids := sumQuantities(in.MyBids)
	totalMyAsks := sumQuantities(in.MyAsks)
	footnote := ""
	switch {
	case len(in.MyBids) > 0 && len(in.MyAsks) > 0:
		footnote = "YOU: " + formatQuantity(totalMyBids) + " " + baseAsset(in.Symbol) + " bid / " +
			formatQuantity(totalMyAsks) + " " + baseAsset(in.Symbol) + " ask open on this market"
	case len(in.MyBids) > 0:
		footnote = "YOU: " + formatQuantity(totalMyBids) + " " + baseAsset(in.Symbol) + " bid open on this market"
	case len(in.MyAsks) > 0:
		footnote = "YOU: " + formatQuantity(totalMyAsks) + " " + baseAsset(in.Symbol) + " ask open on this market"
	}

	return cards.Card{
		Status:   titleStatus,
		Title:    title,
		Sections: []cards.Section{{Rows: rows}},
		Footnote: footnote,
	}.Render()
}

// Produces a single ladder row with optional live-order marker.
func orderbookLevelRow(lvl orderbookLevel, maxQty decimal.Decimal, marks map[string][]myOrderMark, side string) cards.Row {
	priceFloat, _ := lvl.Price.Float64()
	priceStr := cards.USD(priceFloat, priceDecimals(priceFloat))

	key := priceKey(lvl.Price)
	myHere, isMine := marks[key]

	label := "  " + priceStr
	if isMine {
		label = "▶ " + priceStr
	}

	bar := proportionalBar(lvl.Quantity, maxQty, 14)
	value := bar + " " + formatQuantity(lvl.Quantity)

	hint := ""
	if isMine {
		totalMyQty := decimal.Zero
		for _, mark := range myHere {
			totalMyQty = totalMyQty.Add(mark.Quantity)
		}
		hint = "YOU " + side + " " + formatQuantity(totalMyQty)
	}

	return cards.Row{Label: label, Value: value, Hint: hint}
}

// Renders quantity as a fixed-width bar.
// Any non-zero quantity gets at least one cell.
func proportionalBar(qty, max decimal.Decimal, maxChars int) string {
	if qty.IsZero() || max.IsZero() || maxChars <= 0 {
		return ""
	}
	ratio, _ := qty.Div(max).Float64()
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	width := int(ratio*float64(maxChars) + 0.5)
	if width < 1 {
		width = 1
	}
	if width > maxChars {
		width = maxChars
	}
	return strings.Repeat("█", width)
}

func bestPrice(levels []orderbookLevel, highestWins bool) decimal.Decimal {
	best := decimal.Zero
	for _, lvl := range levels {
		if best.IsZero() {
			best = lvl.Price
			continue
		}
		if highestWins && lvl.Price.GreaterThan(best) {
			best = lvl.Price
		}
		if !highestWins && lvl.Price.LessThan(best) {
			best = lvl.Price
		}
	}
	return best
}

func sortedLevels(levels []orderbookLevel, descending bool) []orderbookLevel {
	out := make([]orderbookLevel, len(levels))
	copy(out, levels)
	sort.Slice(out, func(i, j int) bool {
		if descending {
			return out[i].Price.GreaterThan(out[j].Price)
		}
		return out[i].Price.LessThan(out[j].Price)
	})
	return out
}

func topN(levels []orderbookLevel, n int) []orderbookLevel {
	if len(levels) <= n {
		return levels
	}
	return levels[:n]
}

// Groups live orders by canonical price string for one row overlay.
// String keys avoid decimal representation equality traps.
func groupMarksByPrice(marks []myOrderMark) map[string][]myOrderMark {
	out := make(map[string][]myOrderMark, len(marks))
	for _, mark := range marks {
		key := priceKey(mark.Price)
		out[key] = append(out[key], mark)
	}
	return out
}

func priceKey(price decimal.Decimal) string {
	// Use a deliberately verbose key to absorb tiny precision drift.
	return price.StringFixed(10)
}

func sumQuantities(marks []myOrderMark) decimal.Decimal {
	total := decimal.Zero
	for _, mark := range marks {
		total = total.Add(mark.Quantity)
	}
	return total
}

// Formats raw basis points for compact card display.
// Tight markets keep one decimal place.
func formatBps(bps float64) string {
	if bps < 10 {
		return cards.SignedNumber(bps, 1)[1:] + " bps" // strip sign, keep one decimal
	}
	return cards.SignedNumber(bps, 0)[1:] + " bps"
}
