package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
)

// Pulls the order side out of the validated placeOrders payload so
// tests can assert the preview matches what close_position will submit.
func extractClosePreviewSide(t *testing.T, payload any) string {
	t.Helper()
	validated, ok := payload.(*validation.ValidatedPlaceOrdersAction)
	if !ok {
		t.Fatalf("expected *ValidatedPlaceOrdersAction, got %T", payload)
	}
	if validated.Payload == nil || len(validated.Payload.Orders) != 1 {
		t.Fatalf("expected 1 order in close-position preview, got payload=%+v", validated.Payload)
	}
	return strings.ToUpper(strings.TrimSpace(validated.Payload.Orders[0].Side))
}

// Quantity counterpart to extractClosePreviewSide.
func extractClosePreviewQuantity(t *testing.T, payload any) string {
	t.Helper()
	validated, ok := payload.(*validation.ValidatedPlaceOrdersAction)
	if !ok {
		t.Fatalf("expected *ValidatedPlaceOrdersAction, got %T", payload)
	}
	if validated.Payload == nil || len(validated.Payload.Orders) != 1 {
		t.Fatalf("expected 1 order in close-position preview, got payload=%+v", validated.Payload)
	}
	return string(validated.Payload.Orders[0].Quantity)
}

// Explicit side + quantity is the branch that doesn't depend on
// the broker-signed REST read. These unit tests cover the purely
// local normalization path; the REST fetch path is covered by the
// integration suite (partA).

func TestBuildTradePreviewPayloadClosePositionExplicitLongUsesSell(t *testing.T) {
	payload, action, err := buildTradePreviewPayload(context.Background(), nil, ToolContext{}, 42, previewTradeSignatureInput{
		Action: "closePosition",
		ClosePosition: &previewTradeClosePosInput{
			Symbol:   "BTC-USDT",
			Side:     "long",
			Quantity: "1.5",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(action) != "placeOrders" {
		t.Fatalf("expected action=placeOrders, got %q", action)
	}
	if got := extractClosePreviewSide(t, payload); got != "SELL" {
		t.Fatalf("expected long position to close with SELL, got %q", got)
	}
	if got := extractClosePreviewQuantity(t, payload); got != "1.5" {
		t.Fatalf("expected quantity to equal explicit full long position, got %q", got)
	}
}

func TestBuildTradePreviewPayloadClosePositionExplicitShortUsesBuy(t *testing.T) {
	payload, _, err := buildTradePreviewPayload(context.Background(), nil, ToolContext{}, 42, previewTradeSignatureInput{
		Action: "closePosition",
		ClosePosition: &previewTradeClosePosInput{
			Symbol:   "BTC-USDT",
			Side:     "short",
			Quantity: "0.25",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := extractClosePreviewSide(t, payload); got != "BUY" {
		t.Fatalf("expected short position to close with BUY, got %q", got)
	}
	if got := extractClosePreviewQuantity(t, payload); got != "0.25" {
		t.Fatalf("expected quantity to equal explicit full short position, got %q", got)
	}
}

func TestBuildTradePreviewPayloadClosePositionExplicitRequiresQuantity(t *testing.T) {
	_, _, err := buildTradePreviewPayload(context.Background(), nil, ToolContext{}, 42, previewTradeSignatureInput{
		Action: "closePosition",
		ClosePosition: &previewTradeClosePosInput{
			Symbol: "BTC-USDT",
			Side:   "long",
		},
	})
	if err == nil {
		t.Fatal("expected missing quantity to fail")
	}
	if !strings.Contains(err.Error(), "quantity is required") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestBuildTradePreviewPayloadClosePositionExplicitRejectsInvalidSide(t *testing.T) {
	_, _, err := buildTradePreviewPayload(context.Background(), nil, ToolContext{}, 42, previewTradeSignatureInput{
		Action: "closePosition",
		ClosePosition: &previewTradeClosePosInput{
			Symbol:   "BTC-USDT",
			Side:     "flat",
			Quantity: "1",
		},
	})
	if err == nil {
		t.Fatal("expected invalid side to fail")
	}
	if !strings.Contains(err.Error(), "side must be") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestBuildTradePreviewPayloadClosePositionNilReadsWithoutExplicit(t *testing.T) {
	_, _, err := buildTradePreviewPayload(context.Background(), nil, ToolContext{}, 42, previewTradeSignatureInput{
		Action: "closePosition",
		ClosePosition: &previewTradeClosePosInput{
			Symbol: "BTC-USDT",
		},
	})
	if err == nil {
		t.Fatal("expected missing broker-signed read fallback to fail")
	}
	if !errors.Is(err, ErrReadUnavailable) {
		t.Fatalf("expected ErrReadUnavailable, got %v", err)
	}
}

func TestBuildTradePreviewPayloadClosePositionExplicitLimitMethod(t *testing.T) {
	payload, _, err := buildTradePreviewPayload(context.Background(), nil, ToolContext{}, 42, previewTradeSignatureInput{
		Action: "closePosition",
		ClosePosition: &previewTradeClosePosInput{
			Symbol:     "BTC-USDT",
			Side:       "short",
			Quantity:   "0.4",
			Method:     "limit",
			LimitPrice: "30000",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := extractClosePreviewSide(t, payload); got != "BUY" {
		t.Fatalf("expected short position to close with BUY, got %q", got)
	}
	validated := payload.(*validation.ValidatedPlaceOrdersAction)
	order := validated.Payload.Orders[0]
	if !strings.HasPrefix(order.OrderType, "limit") {
		t.Fatalf("expected limit-style order type, got %q", order.OrderType)
	}
	if order.Price != "30000" {
		t.Fatalf("expected limit price preserved, got %q", order.Price)
	}
}
