package resources

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal parse %q: %v", s, err)
	}
	return d
}

func TestBuildTradeJournalAggregates(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	day := func(offset int) time.Time {
		return now.Add(time.Duration(-offset) * 24 * time.Hour)
	}
	entries := []tradeJournalEntry{
		{TradedAt: day(2), Symbol: "BTC-USDT", Side: "SELL", Direction: "CLOSE_LONG", Price: mustDec(t, "76000"), Quantity: mustDec(t, "0.1"), ClosedPnl: mustDec(t, "120"), Fee: mustDec(t, "3.8")},
		{TradedAt: day(2), Symbol: "BTC-USDT", Side: "SELL", Direction: "CLOSE_LONG", Price: mustDec(t, "75800"), Quantity: mustDec(t, "0.05"), ClosedPnl: mustDec(t, "-40"), Fee: mustDec(t, "1.9")},
		{TradedAt: day(1), Symbol: "ETH-USDT", Side: "BUY", Direction: "CLOSE_SHORT", Price: mustDec(t, "3120"), Quantity: mustDec(t, "1"), ClosedPnl: mustDec(t, "60"), Fee: mustDec(t, "1.6")},
		{TradedAt: day(0), Symbol: "BTC-USDT", Side: "BUY", Direction: "OPEN_LONG", Price: mustDec(t, "76300"), Quantity: mustDec(t, "0.05"), ClosedPnl: mustDec(t, "0"), Fee: mustDec(t, "1.9")},
	}

	j := buildTradeJournal(entries, now, 14*24*time.Hour)
	if j.TotalTrades != 4 {
		t.Errorf("TotalTrades = %d; want 4", j.TotalTrades)
	}
	if j.Wins != 2 {
		t.Errorf("Wins = %d; want 2", j.Wins)
	}
	if j.Losses != 1 {
		t.Errorf("Losses = %d; want 1", j.Losses)
	}
	if j.Flat != 1 {
		t.Errorf("Flat = %d; want 1", j.Flat)
	}
	if got := j.NetPnl.String(); got != "140" {
		t.Errorf("NetPnl = %s; want 140", got)
	}
	if got := j.NetFees.String(); got != "9.2" {
		t.Errorf("NetFees = %s; want 9.2", got)
	}
	if len(j.Days) != 3 {
		t.Errorf("Days = %d; want 3 (one per distinct calendar day)", len(j.Days))
	}
	if len(j.BySymbol) != 2 {
		t.Errorf("BySymbol = %d; want 2", len(j.BySymbol))
	}
	if j.BySymbol[0].Symbol != "BTC-USDT" {
		t.Errorf("BySymbol top = %q; want BTC-USDT (highest net)", j.BySymbol[0].Symbol)
	}
	if len(j.Recent) == 0 {
		t.Errorf("Recent should include closed trades")
	}
}

func TestBuildTradeJournalEmpty(t *testing.T) {
	now := time.Now().UTC()
	j := buildTradeJournal(nil, now, 14*24*time.Hour)
	if j.TotalTrades != 0 {
		t.Errorf("TotalTrades = %d; want 0", j.TotalTrades)
	}
}

func TestRenderTradeJournalDocumentEmpty(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	j := buildTradeJournal(nil, now, 14*24*time.Hour)
	body := renderTradeJournalDocument(j, 42)
	if !strings.Contains(body, "TRADE JOURNAL") {
		t.Errorf("card header should still render for empty journal:\n%s", body)
	}
	if !strings.Contains(body, "No fills in window") {
		t.Errorf("empty journal must include 'No fills in window':\n%s", body)
	}
}

func TestRenderTradeJournalDocumentShape(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	day := func(offset int) time.Time { return now.Add(time.Duration(-offset) * 24 * time.Hour) }

	entries := []tradeJournalEntry{
		{TradedAt: day(2), Symbol: "BTC-USDT", Side: "SELL", Price: mustDec(t, "76000"), Quantity: mustDec(t, "0.1"), ClosedPnl: mustDec(t, "120"), Fee: mustDec(t, "3.8")},
		{TradedAt: day(1), Symbol: "ETH-USDT", Side: "BUY", Price: mustDec(t, "3120"), Quantity: mustDec(t, "1"), ClosedPnl: mustDec(t, "-40"), Fee: mustDec(t, "1.6")},
		{TradedAt: day(0), Symbol: "BTC-USDT", Side: "BUY", Price: mustDec(t, "76300"), Quantity: mustDec(t, "0.05"), ClosedPnl: mustDec(t, "0"), Fee: mustDec(t, "1.9")},
	}
	j := buildTradeJournal(entries, now, 14*24*time.Hour)
	body := renderTradeJournalDocument(j, 42)

	for _, want := range []string{
		"TRADE JOURNAL", "Net PnL", "Daily PnL", "Per-symbol", "Recent closed trades",
		"BTC-USDT", "ETH-USDT",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("journal body missing %q:\n%s", want, body)
		}
	}
}

func TestDailyPnlBarsScalesAndDirection(t *testing.T) {
	days := []tradeJournalDay{
		{NetPnl: mustDec(t, "0")},
		{NetPnl: mustDec(t, "50")},
		{NetPnl: mustDec(t, "200")},
		{NetPnl: mustDec(t, "-150")},
		{NetPnl: mustDec(t, "-30")},
	}
	bar := dailyPnlBars(days)
	if bar == "" {
		t.Fatal("expected non-empty bar")
	}
	runes := []rune(bar)
	if len(runes) != len(days) {
		t.Errorf("bar should have one cell per day; got %d cells for %d days", len(runes), len(days))
	}
	if runes[0] != '·' {
		t.Errorf("flat day should render as '·'; got %q", string(runes[0]))
	}
}

func TestParseRESTTimeAcceptsRFC3339(t *testing.T) {
	got := parseRESTTime("2026-04-30T05:30:00Z")
	if got.IsZero() {
		t.Fatal("parseRESTTime returned zero for valid RFC3339")
	}
	if got.Year() != 2026 || got.Month() != 4 || got.Day() != 30 {
		t.Errorf("parseRESTTime returned wrong date: %s", got)
	}
	if !parseRESTTime("garbage").IsZero() {
		t.Error("invalid input should produce zero time")
	}
}

func TestMapTradesToJournalEntriesRoundTrip(t *testing.T) {
	upstream := []map[string]any{
		{
			"tradedAt":               "2026-04-30T10:15:00Z",
			"symbol":                 "BTC-USDT",
			"side":                   "SELL",
			"direction":              "CLOSE_LONG",
			"filledPrice":            "76000",
			"filledQuantity":         "0.1",
			"closedPnl":              "120",
			"fee":                    "3.8",
			"fillType":               "TAKER",
			"orderType":              "MARKET",
			"triggeredByLiquidation": false,
		},
	}
	got := mapTradesToJournalEntries(upstream)
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].Symbol != "BTC-USDT" {
		t.Errorf("symbol = %q; want BTC-USDT", got[0].Symbol)
	}
	if got[0].ClosedPnl.String() != "120" {
		t.Errorf("ClosedPnl = %s; want 120", got[0].ClosedPnl.String())
	}
	if got[0].TradedAt.IsZero() {
		t.Error("TradedAt should parse RFC3339")
	}
}
