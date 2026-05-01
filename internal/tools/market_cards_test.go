package tools

import (
	"strings"
	"testing"
)

func TestRenderMarketSummaryCardShape(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := marketSummaryOutput{
		Market: MarketOutput{
			Symbol:           "BTC-USDT",
			IsOpen:           true,
			TickSize:         "0.1",
			MinTradeAmount:   "0.001",
			MinNotionalValue: "10",
			FundingRateCap:   "0.0005",
			DefaultLeverage:  50,
		},
		Prices: marketPriceOutput{
			IndexPrice: "76300",
			MarkPrice:  "76320",
			LastPrice:  "76315",
		},
		Summary: summaryOutput{
			BestAskPrice:    "76322",
			BestBidPrice:    "76318",
			LastTradedPrice: "76315",
			PrevDayPrice:    "74800",
			Volume24h:       "1820",
			QuoteVolume24h:  "138420000",
		},
		OpenInterest: "412000000",
		FundingRate: &FundingRateEntry{
			EstimatedFundingRate: "0.00018",
			Symbol:               "BTC-USDT",
		},
	}
	card := renderMarketSummaryCard(out)
	for _, want := range []string{
		"MARKET", "BTC-USDT", "Mark", "Last trade", "Bid / ask", "24h range",
		"24h volume", "Open interest", "Est. funding", "Funding", "cap",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("market card missing %q:\n%s", want, card)
		}
	}
	if !strings.Contains(card, "🟢") {
		t.Errorf("positive 24h change should flag 🟢; got:\n%s", card)
	}
	if !strings.Contains(card, "138.42M") {
		t.Errorf("quote volume should compact to 138.42M; got:\n%s", card)
	}
}

func TestRenderMarketSummaryCardClosed(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := marketSummaryOutput{
		Market: MarketOutput{Symbol: "SOL-USDT", IsOpen: false, TickSize: "0.01", MinTradeAmount: "0.1", MinNotionalValue: "5", DefaultLeverage: 20},
		Prices: marketPriceOutput{MarkPrice: "180", IndexPrice: "180", LastPrice: "180"},
		Summary: summaryOutput{LastTradedPrice: "180"},
	}
	card := renderMarketSummaryCard(out)
	if !strings.Contains(card, "CLOSED") {
		t.Errorf("closed market should carry CLOSED badge; got:\n%s", card)
	}
	if !strings.Contains(card, "🟡") {
		t.Errorf("closed market should carry 🟡 warning; got:\n%s", card)
	}
}

func TestFundingBarCenterline(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	flat := fundingBar(0, 0.0005, 11)
	if !strings.Contains(flat, "│") {
		t.Errorf("flat funding bar must show centerline; got %q", flat)
	}
	if strings.Contains(flat, "█") {
		t.Errorf("flat funding bar should have no filled cells; got %q", flat)
	}

	longs := fundingBar(0.0004, 0.0005, 11)
	if !strings.Contains(longs, "█") {
		t.Errorf("positive funding should fill right side; got %q", longs)
	}

	shorts := fundingBar(-0.0004, 0.0005, 11)
	if !strings.Contains(shorts, "█") {
		t.Errorf("negative funding should fill left side; got %q", shorts)
	}
}

func TestCompactNumber(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{12, "12"},
		{1500, "1.5K"},
		{12500, "12.5K"},
		{456789, "456.8K"},
		{1_200_000, "1.2M"},
		{138_420_000, "138.42M"},
		{4_500_000_000, "4.5B"},
	}
	for _, c := range cases {
		if got := compactNumber(c.in); got != c.want {
			t.Errorf("compactNumber(%v) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestRenderListMarketsCardListsSymbols(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := listMarketsOutput{
		Markets: []MarketOutput{
			{Symbol: "BTC-USDT", IsOpen: true},
			{Symbol: "ETH-USDT", IsOpen: true},
			{Symbol: "DOGE-USDT", IsOpen: false},
		},
	}
	card := renderListMarketsCard(out)
	for _, want := range []string{"BTC-USDT", "ETH-USDT", "DOGE-USDT", "3 markets", "get_market_summary"} {
		if !strings.Contains(card, want) {
			t.Errorf("list markets card missing %q:\n%s", want, card)
		}
	}
}

func TestRenderListMarketsCardEmpty(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	card := renderListMarketsCard(listMarketsOutput{})
	if !strings.Contains(card, "No markets returned") {
		t.Errorf("empty list card should mention 'No markets returned'; got:\n%s", card)
	}
}
