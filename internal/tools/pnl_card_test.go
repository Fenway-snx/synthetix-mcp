package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// TestRenderClosePositionCardWinLossShape confirms a winning long
// close renders with a green header and up arrow, a losing short
// close with a red header and down arrow. This is the single most
// important semantic property of the card — if this regresses,
// a losing trade could visually present as a win.
func TestRenderClosePositionCardWinLossShape(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	winningLong := closePositionSnapshot{
		Symbol:     "BTC-USDT",
		Side:       "long",
		Quantity:   decimal.NewFromFloat(0.001),
		EntryPrice: decimal.NewFromFloat(75683),
		CreatedAt:  time.Date(2026, 4, 30, 6, 24, 0, 0, time.UTC),
		Known:      true,
	}
	winResult := closePositionOutput{
		placeOrderOutput: placeOrderOutput{
			Status:   "FILLED",
			AvgPrice: "76300",
			Symbol:   "BTC-USDT",
		},
		ClosedQuantity: "0.001",
	}
	winCard := renderClosePositionCard(winningLong, winResult, decimal.NewFromFloat(0.001))
	if !strings.Contains(winCard, "🟢") {
		t.Errorf("winning long close should carry 🟢 header; got:\n%s", winCard)
	}
	if !strings.Contains(winCard, "▲") {
		t.Errorf("winning long close should carry ▲ somewhere; got:\n%s", winCard)
	}
	if strings.Contains(winCard, "🔴") {
		t.Errorf("winning long close should not carry 🔴; got:\n%s", winCard)
	}

	losingShort := closePositionSnapshot{
		Symbol:     "ETH-USDT",
		Side:       "short",
		Quantity:   decimal.NewFromFloat(1),
		EntryPrice: decimal.NewFromFloat(3000),
		CreatedAt:  time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		Known:      true,
	}
	// Short closed at a higher price than entry → losing short.
	losResult := closePositionOutput{
		placeOrderOutput: placeOrderOutput{
			Status:   "FILLED",
			AvgPrice: "3100",
			Symbol:   "ETH-USDT",
		},
		ClosedQuantity: "1",
	}
	losCard := renderClosePositionCard(losingShort, losResult, decimal.NewFromInt(1))
	if !strings.Contains(losCard, "🔴") {
		t.Errorf("losing short close should carry 🔴 header; got:\n%s", losCard)
	}
	if !strings.Contains(losCard, "▼") {
		t.Errorf("losing short close should carry ▼ somewhere; got:\n%s", losCard)
	}
	if strings.Contains(losCard, "🟢") {
		t.Errorf("losing short close should not carry 🟢; got:\n%s", losCard)
	}
}

// TestRenderClosePositionCardDegradedWhenFillUnknown confirms a
// close where we couldn't capture a snapshot (or where the fill
// didn't report an avg price) renders a neutral acknowledgement
// card instead of fabricating PnL numbers.
func TestRenderClosePositionCardDegradedWhenFillUnknown(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	emptySnapshot := closePositionSnapshot{Symbol: "BTC-USDT"}
	accepted := closePositionOutput{
		placeOrderOutput: placeOrderOutput{Status: "ACCEPTED", Symbol: "BTC-USDT"},
		ClosedQuantity:   "0.5",
	}
	card := renderClosePositionCard(emptySnapshot, accepted, decimal.NewFromFloat(0.5))
	if card == "" {
		t.Fatal("expected degraded acknowledgement card, got empty string")
	}
	if strings.Contains(card, "Realized PnL") {
		t.Errorf("degraded card should NOT invent a realized PnL row; got:\n%s", card)
	}
	if !strings.Contains(card, "CLOSE ACCEPTED") {
		t.Errorf("degraded card should call out the ACCEPTED status; got:\n%s", card)
	}
}

// TestRenderCloseAllPositionsCardAggregatesNet confirms the
// portfolio-close card computes a net PnL across legs and picks
// the header status from the net.
func TestRenderCloseAllPositionsCardAggregatesNet(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	snapshots := map[string]closePositionSnapshot{
		"BTC-USDT": {
			Symbol:     "BTC-USDT",
			Side:       "long",
			Quantity:   decimal.NewFromFloat(0.001),
			EntryPrice: decimal.NewFromFloat(75000),
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			Known:      true,
		},
		"ETH-USDT": {
			Symbol:     "ETH-USDT",
			Side:       "long",
			Quantity:   decimal.NewFromFloat(0.1),
			EntryPrice: decimal.NewFromFloat(3000),
			CreatedAt:  time.Now().Add(-30 * time.Minute),
			Known:      true,
		},
	}
	out := closeAllPositionsOutput{
		Positions: []closeAllPositionItemOutput{
			{
				Symbol:         "BTC-USDT",
				Side:           "long",
				ClosedQuantity: "0.001",
				Order:          placeOrderOutput{Status: "FILLED", AvgPrice: "76000", Symbol: "BTC-USDT"},
			},
			{
				Symbol:         "ETH-USDT",
				Side:           "long",
				ClosedQuantity: "0.1",
				Order:          placeOrderOutput{Status: "FILLED", AvgPrice: "2900", Symbol: "ETH-USDT"},
			},
		},
	}
	quantities := []string{"0.001", "0.1"}
	card := renderCloseAllPositionsCard(snapshots, out, quantities)

	// BTC leg wins ~$1 (0.001 * 1000), ETH leg loses $10 (0.1 *
	// -100). Net is ~-$9 → red header.
	if !strings.Contains(card, "🔴") {
		t.Errorf("net-negative close_all card should carry 🔴 header; got:\n%s", card)
	}
	if !strings.Contains(card, "Net realized PnL") {
		t.Errorf("close_all card must include Net realized PnL row; got:\n%s", card)
	}
	if !strings.Contains(card, "BTC-USDT") || !strings.Contains(card, "ETH-USDT") {
		t.Errorf("close_all card must list every symbol; got:\n%s", card)
	}
}

// TestRenderCloseAllPositionsCardRejectedSwitchesToWarning keeps
// the card visually distinct when any leg rejected: the trader
// must be prompted to read the JSON for recovery, not led to
// believe the batch succeeded.
func TestRenderCloseAllPositionsCardRejectedSwitchesToWarning(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := closeAllPositionsOutput{
		Positions: []closeAllPositionItemOutput{
			{
				Symbol:         "BTC-USDT",
				Side:           "long",
				ClosedQuantity: "0.001",
				Order:          placeOrderOutput{Status: "REJECTED", Symbol: "BTC-USDT", Message: "insufficient margin"},
			},
		},
	}
	snapshots := map[string]closePositionSnapshot{
		"BTC-USDT": {Symbol: "BTC-USDT", Side: "long", EntryPrice: decimal.NewFromFloat(70000), Known: true},
	}
	card := renderCloseAllPositionsCard(snapshots, out, []string{"0.001"})
	if !strings.Contains(card, "🟡") {
		t.Errorf("any-leg-rejected close_all card must carry 🟡 header; got:\n%s", card)
	}
	if !strings.Contains(card, "REJECTED") {
		t.Errorf("card must surface the rejection status; got:\n%s", card)
	}
}

// TestComputeCloseEconomicsFlipsSignForShort pins the direction
// logic: a SHORT closed BELOW entry is a WIN, a LONG closed
// BELOW entry is a LOSS. This is the easiest place to introduce
// a sign-flip bug and it'd be catastrophic for UX.
func TestComputeCloseEconomicsFlipsSignForShort(t *testing.T) {
	longSnapshot := closePositionSnapshot{
		Side: "long", EntryPrice: decimal.NewFromFloat(100), Quantity: decimal.NewFromInt(1),
	}
	shortSnapshot := closePositionSnapshot{
		Side: "short", EntryPrice: decimal.NewFromFloat(100), Quantity: decimal.NewFromInt(1),
	}
	exitBelow := decimal.NewFromFloat(90)

	longPnL, _, longArrow := computeCloseEconomics(longSnapshot, exitBelow, decimal.NewFromInt(1))
	if longPnL.IsPositive() {
		t.Errorf("long closed below entry must lose money; got PnL=%s", longPnL)
	}
	if longArrow != "▼" {
		t.Errorf("long loss must carry ▼ arrow; got %q", longArrow)
	}

	shortPnL, _, shortArrow := computeCloseEconomics(shortSnapshot, exitBelow, decimal.NewFromInt(1))
	if !shortPnL.IsPositive() {
		t.Errorf("short closed below entry must WIN money; got PnL=%s", shortPnL)
	}
	if shortArrow != "▲" {
		t.Errorf("short win must carry ▲ arrow; got %q", shortArrow)
	}
}

// TestFormatQuantityTrimsTrailingZeros keeps the Quantity row
// readable across assets — "0.001 BTC" not "0.0010000 BTC".
func TestFormatQuantityTrimsTrailingZeros(t *testing.T) {
	cases := []struct {
		in   decimal.Decimal
		want string
	}{
		{decimal.NewFromFloat(0.001), "0.001"},
		{decimal.NewFromFloat(1), "1"},
		{decimal.NewFromFloat(123.456), "123.456"},
		{decimal.NewFromFloat(0.00001), "0.00001"},
		{decimal.Zero, "0"},
	}
	for _, tc := range cases {
		if got := formatQuantity(tc.in); got != tc.want {
			t.Errorf("formatQuantity(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestBaseAssetHandlesCommonSeparators guards the unit label on
// the Quantity row — anything before the first - / or _ wins,
// unrecognized formats fall back to "contracts".
func TestBaseAssetHandlesCommonSeparators(t *testing.T) {
	cases := map[string]string{
		"BTC-USDT":   "BTC",
		"ETH/USD":    "ETH",
		"SOL_USDT":   "SOL",
		"BARE":       "contracts",
		"":           "contracts",
	}
	for in, want := range cases {
		if got := baseAsset(in); got != want {
			t.Errorf("baseAsset(%q) = %q, want %q", in, got, want)
		}
	}
}
