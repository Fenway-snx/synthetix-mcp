package tools

import (
	"strings"
	"testing"
)

func TestRenderAccountSummaryCardHealthy(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := accountSummaryOutput{
		SubAccountID:   42,
		Name:           "main",
		PositionCount:  2,
		OpenOrderCount: 1,
		MarginSummary: marginSummaryOutput{
			AccountValue:      "1000",
			InitialMargin:     "100",
			MaintenanceMargin: "50",
			AvailableMargin:   "900",
			UnrealizedPnl:     "25",
		},
		FeeRates: feeRatesOutput{
			TierName:     "T1",
			MakerFeeRate: "0.0002",
			TakerFeeRate: "0.0005",
		},
	}
	card := renderAccountSummaryCard(out)
	if !strings.Contains(card, "🟢") {
		t.Errorf("low utilisation should be healthy (🟢); got:\n%s", card)
	}
	if !strings.Contains(card, "Equity") {
		t.Errorf("card must show Equity; got:\n%s", card)
	}
	if !strings.Contains(card, "Available") {
		t.Errorf("card must show Available margin; got:\n%s", card)
	}
	if !strings.Contains(card, "2") || !strings.Contains(card, "1") {
		t.Errorf("card must show position/order counts; got:\n%s", card)
	}
	if !strings.Contains(card, "2.0 bps") {
		t.Errorf("maker fee 0.0002 should render as 2.0 bps; got:\n%s", card)
	}
}

func TestRenderAccountSummaryCardCritical(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := accountSummaryOutput{
		SubAccountID: 1,
		MarginSummary: marginSummaryOutput{
			AccountValue:      "100",
			InitialMargin:     "95",
			MaintenanceMargin: "85",
			AvailableMargin:   "5",
			UnrealizedPnl:     "-20",
		},
	}
	card := renderAccountSummaryCard(out)
	if !strings.Contains(card, "🔥") {
		t.Errorf("near-maintenance account should carry 🔥 critical status; got:\n%s", card)
	}
}

func TestRenderPositionsCardEmpty(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	card := renderPositionsCard(positionsOutput{SubAccountID: 1})
	if !strings.Contains(card, "No open positions") {
		t.Errorf("empty positions card should say 'No open positions'; got:\n%s", card)
	}
}

func TestRenderPositionsCardAggregatesNetUPnL(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := positionsOutput{
		SubAccountID: 1,
		Positions: []positionOutput{
			{Symbol: "BTC-USDT", Side: "LONG", Quantity: "0.1", EntryPrice: "75000", UnrealizedPnl: "50", LiquidationPrice: "60000"},
			{Symbol: "ETH-USDT", Side: "SHORT", Quantity: "1", EntryPrice: "3000", UnrealizedPnl: "-20", LiquidationPrice: "3500"},
		},
	}
	card := renderPositionsCard(out)
	if !strings.Contains(card, "🟢") {
		t.Errorf("net +30 uPnL should carry 🟢; got:\n%s", card)
	}
	if !strings.Contains(card, "L▲ BTC-USDT") {
		t.Errorf("positions card must mark long with L▲; got:\n%s", card)
	}
	if !strings.Contains(card, "S▼ ETH-USDT") {
		t.Errorf("positions card must mark short with S▼; got:\n%s", card)
	}
	if !strings.Contains(card, "Net unrealized PnL") {
		t.Errorf("positions card must include net uPnL summary; got:\n%s", card)
	}
}

func TestRenderOpenOrdersCardRowsWithBadges(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := openOrdersOutput{
		SubAccountID: 7,
		Orders: []openOrderOutput{
			{Symbol: "BTC-USDT", Side: "BUY", Price: "70000", Quantity: "0.1", RemainingQuantity: "0.1", TimeInForce: "GTC"},
			{Symbol: "ETH-USDT", Side: "SELL", Price: "3200", Quantity: "2", RemainingQuantity: "1.5", TimeInForce: "GTC", ReduceOnly: true},
		},
	}
	card := renderOpenOrdersCard(out)
	if !strings.Contains(card, "BUY ▲ BTC-USDT") {
		t.Errorf("BUY row should carry BUY ▲ badge; got:\n%s", card)
	}
	if !strings.Contains(card, "SELL ▼ ETH-USDT") {
		t.Errorf("SELL row should carry SELL ▼ badge; got:\n%s", card)
	}
	if !strings.Contains(card, "reduce-only") {
		t.Errorf("reduce-only hint should be present; got:\n%s", card)
	}
}

func TestRenderOpenOrdersCardEmpty(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	card := renderOpenOrdersCard(openOrdersOutput{SubAccountID: 1})
	if !strings.Contains(card, "No open orders") {
		t.Errorf("empty card should say 'No open orders'; got:\n%s", card)
	}
}
