package tools

import (
	"strings"
	"testing"
)

func TestRenderPlaceOrderCardFilled(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	normalized := normalizedOrderOutput{
		Symbol:   "BTC-USDT",
		Side:     "BUY",
		Type:     "MARKET",
		Quantity: "0.001",
	}
	result := placeOrderOutput{
		Status:    "FILLED",
		IsSuccess: true,
		Accepted:  true,
		AvgPrice:  "76300",
		CumQty:    "0.001",
		OrigQty:   "0.001",
		Symbol:    "BTC-USDT",
	}
	card := renderPlaceOrderCard(normalized, result)
	if !strings.Contains(card, "🟢") {
		t.Errorf("filled market buy should carry 🟢 header; got:\n%s", card)
	}
	if !strings.Contains(card, "FILLED BUY ▲") {
		t.Errorf("title should read 'FILLED BUY ▲'; got:\n%s", card)
	}
	if !strings.Contains(card, "Avg fill") {
		t.Errorf("filled card must show Avg fill; got:\n%s", card)
	}
	if !strings.Contains(card, "Notional") {
		t.Errorf("filled card must show Notional; got:\n%s", card)
	}
}

func TestRenderPlaceOrderCardRestingLimit(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	normalized := normalizedOrderOutput{
		Symbol:      "BTC-USDT",
		Side:        "SELL",
		Type:        "LIMIT",
		Quantity:    "0.1",
		Price:       "77000",
		TimeInForce: "GTC",
	}
	result := placeOrderOutput{
		Status:    "ACCEPTED",
		IsSuccess: true,
		Accepted:  true,
		Symbol:    "BTC-USDT",
		OrderID:   orderIDOutput{VenueID: 12345, ClientID: "cli-xyz"},
	}
	card := renderPlaceOrderCard(normalized, result)
	if strings.Contains(card, "🟢") {
		t.Errorf("resting limit should NOT carry 🟢; it's neutral until filled; got:\n%s", card)
	}
	if !strings.Contains(card, "RESTING SELL BTC-USDT") {
		t.Errorf("title should call out RESTING; got:\n%s", card)
	}
	if !strings.Contains(card, "Resting on book") {
		t.Errorf("outcome row should say 'Resting on book'; got:\n%s", card)
	}
	if !strings.Contains(card, "venue:12345") {
		t.Errorf("resting card should show venue order id; got:\n%s", card)
	}
}

func TestRenderPlaceOrderCardRejected(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	normalized := normalizedOrderOutput{
		Symbol:   "BTC-USDT",
		Side:     "BUY",
		Type:     "LIMIT",
		Quantity: "10",
		Price:    "80000",
	}
	result := placeOrderOutput{
		Status:    "REJECTED",
		IsSuccess: false,
		Symbol:    "BTC-USDT",
		Message:   "insufficient margin to open position",
		ErrorCode: "INSUFFICIENT_MARGIN",
		ErrorDetail: &errorDetail{
			Remediation: []string{"reduce order size or deposit more collateral"},
			Retryable:   false,
		},
	}
	card := renderPlaceOrderCard(normalized, result)
	if !strings.Contains(card, "🔴") {
		t.Errorf("rejected order should carry 🔴 header; got:\n%s", card)
	}
	if !strings.Contains(card, "REJECTED BUY BTC-USDT") {
		t.Errorf("title should call out REJECTED; got:\n%s", card)
	}
	if !strings.Contains(card, "INSUFFICIENT_MARGIN") {
		t.Errorf("rejected card must show error code; got:\n%s", card)
	}
	if !strings.Contains(card, "Fix:") {
		t.Errorf("rejected card should surface remediation as Fix: row; got:\n%s", card)
	}
}

func TestRenderPlaceOrderCardPartialFill(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	normalized := normalizedOrderOutput{
		Symbol:      "BTC-USDT",
		Side:        "BUY",
		Type:        "LIMIT",
		Quantity:    "1",
		Price:       "76300",
		TimeInForce: "IOC",
	}
	result := placeOrderOutput{
		Status:   "FILLED",
		AvgPrice: "76300",
		CumQty:   "0.25",
		OrigQty:  "1",
		Symbol:   "BTC-USDT",
	}
	card := renderPlaceOrderCard(normalized, result)
	if !strings.Contains(card, "Partially filled") {
		t.Errorf("partial fill must be called out; got:\n%s", card)
	}
	if !strings.Contains(card, "filled 0.25 of 1") {
		t.Errorf("partial fill hint should show filled-vs-requested; got:\n%s", card)
	}
}
