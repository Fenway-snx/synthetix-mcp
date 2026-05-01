package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/synthetixio/synthetix-go/types"
)

func TestRenderGuardrailRejectionCardOrderQtyOverCap(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	ctx := context.Background()
	manager := risksnapshot.NewManager(fakeGuardrailHydrationClient{})
	state := &session.State{
		SubAccountID: 1,
		AgentGuardrails: &guardrails.Config{
			Preset:            guardrails.PresetStandard,
			AllowedSymbols:    []string{"*"},
			AllowedOrderTypes: []string{"*"},
			MaxOrderQuantity:  "1",
		},
	}
	prices := fakeGuardrailPriceReader{"BTC-USDT": types.MarketPriceResponse{MarkPrice: "76000"}}

	normalized := normalizedOrderOutput{
		Symbol:   "BTC-USDT",
		Side:     "BUY",
		Type:     "MARKET",
		Quantity: "5",
	}
	err := enforcePlaceOrderGuardrails(ctx, "session", state, manager, prices, normalized)
	if err == nil {
		t.Fatal("expected guardrail violation")
	}
	v := isGuardrailViolation(err)
	if v == nil {
		t.Fatalf("expected *guardrailViolation; got %T (%v)", err, err)
	}
	if v.Field != guardrailFieldOrderQuantity {
		t.Errorf("Field = %q; want %q", v.Field, guardrailFieldOrderQuantity)
	}
	card := renderGuardrailRejectionCard(v, normalized)
	for _, want := range []string{
		"BLOCKED BUY", "BTC-USDT", "ORDER QTY OVER CAP",
		"Reason:", "Submitted qty:", "Cap:", "split the order",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("rejection card missing %q:\n%s", want, card)
		}
	}
	if !strings.Contains(card, "🔴") {
		t.Errorf("rejection card should carry 🔴 negative status:\n%s", card)
	}
	// Visual smoke: ensure the card width is exactly 80 cells per the
	// project's terminal-friendly card width contract.
	for _, line := range strings.Split(card, "\n") {
		if line == "" {
			continue
		}
		if got := len([]rune(line)); got > 84 {
			t.Errorf("card line exceeds expected width: %d runes:\n%q", got, line)
		}
	}
}

func TestRenderGuardrailRejectionCardSymbolNotAllowed(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	ctx := context.Background()
	manager := risksnapshot.NewManager(fakeGuardrailHydrationClient{})
	state := &session.State{
		SubAccountID: 1,
		AgentGuardrails: &guardrails.Config{
			Preset:            guardrails.PresetStandard,
			AllowedSymbols:    []string{"ETH-USDT"},
			AllowedOrderTypes: []string{"*"},
		},
	}
	prices := fakeGuardrailPriceReader{}

	normalized := normalizedOrderOutput{Symbol: "BTC-USDT", Side: "BUY", Type: "MARKET", Quantity: "1"}
	err := enforcePlaceOrderGuardrails(ctx, "session", state, manager, prices, normalized)
	v := isGuardrailViolation(err)
	if v == nil {
		t.Fatalf("expected *guardrailViolation; got %v", err)
	}
	if v.Field != guardrailFieldSymbolNotAllowed {
		t.Errorf("Field = %q; want %q", v.Field, guardrailFieldSymbolNotAllowed)
	}
	card := renderGuardrailRejectionCard(v, normalized)
	if !strings.Contains(card, "SYMBOL NOT ALLOWED") {
		t.Errorf("card should mention SYMBOL NOT ALLOWED:\n%s", card)
	}
	if !strings.Contains(card, "allow-list") {
		t.Errorf("card should suggest allow-list remediation:\n%s", card)
	}
}

func TestRenderGuardrailRejectionCardReadOnly(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	ctx := context.Background()
	manager := risksnapshot.NewManager(fakeGuardrailHydrationClient{})
	state := &session.State{
		SubAccountID: 1,
		AgentGuardrails: &guardrails.Config{
			Preset: guardrails.PresetReadOnly,
		},
	}
	prices := fakeGuardrailPriceReader{}

	normalized := normalizedOrderOutput{Symbol: "BTC-USDT", Side: "BUY", Type: "MARKET", Quantity: "1"}
	err := enforcePlaceOrderGuardrails(ctx, "session", state, manager, prices, normalized)
	v := isGuardrailViolation(err)
	if v == nil {
		t.Fatalf("expected *guardrailViolation; got %v", err)
	}
	if v.Field != guardrailFieldReadOnly {
		t.Errorf("Field = %q; want %q", v.Field, guardrailFieldReadOnly)
	}
	card := renderGuardrailRejectionCard(v, normalized)
	if !strings.Contains(card, "READ-ONLY SESSION") {
		t.Errorf("card should mention READ-ONLY SESSION:\n%s", card)
	}
}

func TestGuardrailRejectionResponseAttachesCard(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	v := &guardrailViolation{
		Reason: "quantity 5 exceeds maxOrderQuantity 1",
		Field:  guardrailFieldOrderQuantity,
		Symbol: "BTC-USDT",
		Side:   "BUY",
	}
	normalized := normalizedOrderOutput{Symbol: "BTC-USDT", Side: "BUY", Type: "MARKET", Quantity: "5"}
	result, _, _ := guardrailRejectionResponse[placeOrderOutput](v, normalized)
	if result == nil {
		t.Fatal("guardrailRejectionResponse returned nil result")
	}
	if !result.IsError {
		t.Error("result should carry IsError=true")
	}
	if len(result.Content) < 2 {
		t.Fatalf("expected at least 2 content blocks (card + error JSON); got %d", len(result.Content))
	}
}
