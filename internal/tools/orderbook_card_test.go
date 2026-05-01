package tools

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// Confirms ladder layout and live-order markers render at the right levels.
func TestRenderOrderbookCardShape(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	in := orderbookCardInput{
		Symbol: "BTC-USDT",
		Asks: []orderbookLevel{
			{Price: decimal.NewFromInt(76320), Quantity: decimal.NewFromFloat(0.125)},
			{Price: decimal.NewFromInt(76318), Quantity: decimal.NewFromFloat(0.090)},
			{Price: decimal.NewFromInt(76316), Quantity: decimal.NewFromFloat(0.050)},
		},
		Bids: []orderbookLevel{
			{Price: decimal.NewFromInt(76312), Quantity: decimal.NewFromFloat(0.100)},
			{Price: decimal.NewFromInt(76310), Quantity: decimal.NewFromFloat(0.200)},
			{Price: decimal.NewFromInt(76308), Quantity: decimal.NewFromFloat(0.080)},
		},
		MyAsks: []myOrderMark{
			{Price: decimal.NewFromInt(76316), Quantity: decimal.NewFromFloat(0.050), Side: "SELL"},
		},
		MyBids: []myOrderMark{
			{Price: decimal.NewFromInt(76312), Quantity: decimal.NewFromFloat(0.100), Side: "BUY"},
		},
	}
	card := renderOrderbookCard(in, 6)

	if !strings.Contains(card, "BTC-USDT") {
		t.Errorf("card must include symbol; got:\n%s", card)
	}
	if !strings.Contains(card, "mid") {
		t.Errorf("card must include mid-price; got:\n%s", card)
	}
	if !strings.Contains(card, "spread") {
		t.Errorf("card must include spread divider; got:\n%s", card)
	}
	// My-order arrows must appear exactly twice (one for bid, one for ask).
	arrowCount := strings.Count(card, "▶")
	if arrowCount != 2 {
		t.Errorf("expected 2 ▶ markers (one per live order), got %d; card:\n%s", arrowCount, card)
	}
	if !strings.Contains(card, "YOU SELL") {
		t.Errorf("ask-side my-order hint missing; got:\n%s", card)
	}
	if !strings.Contains(card, "YOU BUY") {
		t.Errorf("bid-side my-order hint missing; got:\n%s", card)
	}
}

// Confirms public ladders render cleanly without live-order markers.
func TestRenderOrderbookCardRendersWithoutMyOrders(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	in := orderbookCardInput{
		Symbol: "ETH-USDT",
		Asks: []orderbookLevel{
			{Price: decimal.NewFromInt(3005), Quantity: decimal.NewFromFloat(10)},
			{Price: decimal.NewFromInt(3003), Quantity: decimal.NewFromFloat(5)},
		},
		Bids: []orderbookLevel{
			{Price: decimal.NewFromInt(3001), Quantity: decimal.NewFromFloat(8)},
			{Price: decimal.NewFromInt(2999), Quantity: decimal.NewFromFloat(12)},
		},
	}
	card := renderOrderbookCard(in, 6)
	if strings.Contains(card, "▶") {
		t.Errorf("no-my-orders card must not include ▶ markers; got:\n%s", card)
	}
	if strings.Contains(card, "YOU ") {
		t.Errorf("no-my-orders card must not include YOU tags; got:\n%s", card)
	}
}

// Confirms empty books fall through to JSON-only responses.
func TestRenderOrderbookCardEmptyBook(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")
	if got := renderOrderbookCard(orderbookCardInput{Symbol: "BTC-USDT"}, 6); got != "" {
		t.Errorf("empty book must return empty card; got:\n%s", got)
	}
}

// Confirms bar width scales with quantity and zero renders empty.
func TestProportionalBarScalesAcrossSides(t *testing.T) {
	max := decimal.NewFromFloat(1.0)
	if got := proportionalBar(decimal.Zero, max, 10); got != "" {
		t.Errorf("zero quantity must render no bar; got %q", got)
	}
	full := proportionalBar(decimal.NewFromFloat(1.0), max, 10)
	half := proportionalBar(decimal.NewFromFloat(0.5), max, 10)
	if len(full) == 0 {
		t.Fatalf("full-scale quantity must render a non-empty bar")
	}
	if len([]rune(full)) != 10 {
		t.Errorf("full bar expected 10 chars; got %d", len([]rune(full)))
	}
	if len([]rune(half)) != 5 {
		t.Errorf("half bar expected 5 chars; got %d", len([]rune(half)))
	}
}

// Confirms multiple live orders at one price aggregate into one row.
func TestGroupMarksByPriceAggregatesSamePrice(t *testing.T) {
	marks := []myOrderMark{
		{Price: decimal.NewFromInt(100), Quantity: decimal.NewFromFloat(0.5), Side: "BUY"},
		{Price: decimal.NewFromInt(100), Quantity: decimal.NewFromFloat(0.3), Side: "BUY"},
		{Price: decimal.NewFromInt(99), Quantity: decimal.NewFromFloat(1), Side: "BUY"},
	}
	grouped := groupMarksByPrice(marks)
	if len(grouped) != 2 {
		t.Fatalf("expected 2 price buckets; got %d", len(grouped))
	}
	key := priceKey(decimal.NewFromInt(100))
	if got := len(grouped[key]); got != 2 {
		t.Errorf("expected 2 orders at $100; got %d", got)
	}
}
