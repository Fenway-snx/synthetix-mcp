package tools

import (
	"context"
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
)

// orderbookCardInput bundles the parsed orderbook and the set of
// live orders belonging to this session for one symbol. It is
// produced by buildOrderbookCardInput so the render function is
// a pure transform over primitive types — easy to unit test
// without spinning up the REST client or snapshot manager.
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

// buildOrderbookCardInput fuses the orderbook response with the
// session's live orders. When the session is unauthenticated or
// the snapshot manager isn't available, the my-orders slices
// come back empty and the card still renders a clean ladder
// without the "YOU" markers.
//
// We only fetch orders when the orderbook and session both exist
// because a hydration call is measurably expensive; a public user
// running get_orderbook shouldn't pay for a snapshot refresh they
// can't benefit from.
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

// renderOrderbookCard renders the ladder. Ask side comes first
// in descending-price order (furthest from mid on top), then
// the spread divider, then the bid side in descending-price
// order (best bid on top, furthest below at the bottom). This
// mirrors every professional trading UI, so the visual read
// matches the trader's mental model.
//
// Each row has a quantity bar whose width is proportional to
// the row's quantity relative to the max quantity across both
// sides — a single common scale keeps bids and asks visually
// comparable, which is how a trader judges where real liquidity
// sits.
//
// My-orders overlays collapse into a single [YOU: SIDE qty @ px]
// tag per row; if a trader has multiple orders at the same
// price they get aggregated into one tag so the card stays at
// one row per level.
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

	// Max quantity across all displayed levels — used as the
	// denominator for the bar width so depth proportions stay
	// honest across both sides.
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

	// Neutral status — mid-price is context, not outcome. A
	// winner/loser framing would imply a judgement the data
	// doesn't support.
	titleStatus := cards.StatusNeutral

	midFloat, _ := mid.Float64()
	title := "ORDERBOOK  " + in.Symbol
	if !mid.IsZero() {
		title = title + "  mid " + cards.USD(midFloat, priceDecimals(midFloat))
	}
	rows := make([]cards.Row, 0, len(asksDisplay)+len(bidsDisplay)+3)

	// Ask side: show farthest-from-mid on top (reverse the
	// ascending sort) so the ladder reads top-down like a
	// conventional depth chart.
	for i := len(asksDisplay) - 1; i >= 0; i-- {
		lvl := asksDisplay[i]
		rows = append(rows, orderbookLevelRow(lvl, maxQty, myAsksByPrice, "SELL"))
	}

	// Spread divider row. The cards engine doesn't support raw
	// dividers inside a section, so we emit a "label" row whose
	// value is centered through the label.
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

// orderbookLevelRow produces a single ladder row:
//
//	Label = "→ $76,316" when the trader has an order here, else
//	        "  $76,316" to keep the column aligned.
//	Value = bar (proportional width) + raw quantity
//	Hint  = "[YOU: SELL 0.050 @ $76,316]" when applicable
//
// The leading "→" arrow is our most important visual cue — it's
// the "flag" the user asked for on the orderbook card. We render
// it explicitly so a trader can scan for their own prices in a
// hundred-deep ladder at a glance.
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

// proportionalBar renders a quantity as a fixed-width bar of
// █ characters. Width is qty/max * maxChars, rounded; any
// non-zero quantity yields at least one █ so microscopic levels
// still appear on the ladder.
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

// groupMarksByPrice keys the trader's live orders by a canonical
// price string so multiple orders at the same price collapse
// into one row overlay. A stringified Decimal is used instead of
// a Decimal itself so map equality works — two decimals that
// compare equal may have different internal representations.
func groupMarksByPrice(marks []myOrderMark) map[string][]myOrderMark {
	out := make(map[string][]myOrderMark, len(marks))
	for _, mark := range marks {
		key := priceKey(mark.Price)
		out[key] = append(out[key], mark)
	}
	return out
}

func priceKey(price decimal.Decimal) string {
	// 10 decimals is overkill for any real market but protects
	// against subtle precision drift when the exchange reports
	// a level at 0.00001234 and our internal order was placed
	// at 0.000012340000000001. The trailing zeros are stripped
	// only for display; the key is explicitly verbose.
	return price.StringFixed(10)
}

func sumQuantities(marks []myOrderMark) decimal.Decimal {
	total := decimal.Zero
	for _, mark := range marks {
		total = total.Add(mark.Quantity)
	}
	return total
}

// formatBps returns "3.2 bps" for a raw basis-points number.
// Tight markets go to one decimal so a 0.5 bps spread doesn't
// round to "0 bps" and confuse the reader.
func formatBps(bps float64) string {
	if bps < 10 {
		return cards.SignedNumber(bps, 1)[1:] + " bps" // strip sign, keep one decimal
	}
	return cards.SignedNumber(bps, 0)[1:] + " bps"
}
