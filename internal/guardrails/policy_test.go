package guardrails

import "testing"

func TestResolveUnknownPresetFallsBackToReadOnly(t *testing.T) {
	resolved, err := Resolve(&Config{Preset: "surprise"})
	if err != nil {
		t.Fatalf("expected unknown preset fallback, got error: %v", err)
	}
	if resolved.EffectivePreset != PresetReadOnly {
		t.Fatalf("expected %s, got %s", PresetReadOnly, resolved.EffectivePreset)
	}
	if resolved.WriteEnabled() {
		t.Fatal("expected write disabled")
	}
}

func TestResolveNilDefaultsToUnrestrictedStandard(t *testing.T) {
	resolved, err := Resolve(nil)
	if err != nil {
		t.Fatalf("expected nil config to resolve, got error: %v", err)
	}
	if resolved.EffectivePreset != PresetStandard {
		t.Fatalf("expected %s, got %s", PresetStandard, resolved.EffectivePreset)
	}
	if !resolved.WriteEnabled() {
		t.Fatal("expected writes enabled")
	}
	if !resolved.IsSymbolAllowed("BTC-USDT") {
		t.Fatalf("expected default symbol wildcard, got %#v", resolved.AllowedSymbols)
	}
	if !resolved.IsOrderTypeAllowed("LIMIT") || !resolved.IsOrderTypeAllowed("MARKET") {
		t.Fatalf("expected default order types, got %#v", resolved.AllowedOrderTypes)
	}
	if resolved.HasMaxOrderQuantity() || resolved.HasMaxOrderNotional() || resolved.HasMaxPositionQuantity() || resolved.HasMaxPositionNotional() {
		t.Fatal("expected no default quantity/notional caps")
	}
}

func TestResolveStandardNormalizesSymbolsAndTypes(t *testing.T) {
	resolved, err := Resolve(&Config{
		Preset:              PresetStandard,
		AllowedSymbols:      []string{"btc-usdt", "BTC-USDT"},
		AllowedOrderTypes:   []string{"limit", "stop_limit", "take_profit_limit"},
		MaxOrderQuantity:    "2",
		MaxPositionQuantity: "4",
	})
	if err != nil {
		t.Fatalf("resolve standard failed: %v", err)
	}
	if len(resolved.AllowedSymbols) != 1 || resolved.AllowedSymbols[0] != "BTC-USDT" {
		t.Fatalf("expected normalized symbol list, got %#v", resolved.AllowedSymbols)
	}
	if !resolved.IsOrderTypeAllowed("limit") || !resolved.IsOrderTypeAllowed("STOP_LIMIT") || !resolved.IsOrderTypeAllowed("take_profit_limit") {
		t.Fatalf("expected normalized order types, got %#v", resolved.AllowedOrderTypes)
	}
	if len(resolved.AllowedOrderTypes) != 3 || resolved.AllowedOrderTypes[1] != "STOP" || resolved.AllowedOrderTypes[2] != "TAKE_PROFIT" {
		t.Fatalf("expected stop-limit aliases to normalize to MCP-supported types, got %#v", resolved.AllowedOrderTypes)
	}
}

func TestResolveStandardSupportsWildcardSymbolsAndOrderTypes(t *testing.T) {
	resolved, err := Resolve(&Config{
		Preset:              PresetStandard,
		AllowedSymbols:      []string{"all"},
		AllowedOrderTypes:   []string{"*"},
		MaxOrderNotional:    "100000",
		MaxPositionNotional: "1000000",
	})
	if err != nil {
		t.Fatalf("resolve standard failed: %v", err)
	}
	if !resolved.IsSymbolAllowed("BTC-USDT") || !resolved.IsSymbolAllowed("any-market") {
		t.Fatalf("expected wildcard symbol allowlist, got %#v", resolved.AllowedSymbols)
	}
	for _, orderType := range supportedOrderTypes {
		if !resolved.IsOrderTypeAllowed(orderType) {
			t.Fatalf("expected wildcard order types to include %s, got %#v", orderType, resolved.AllowedOrderTypes)
		}
	}
	if !resolved.HasMaxOrderNotional() || resolved.MaxOrderNotional.String() != "100000" {
		t.Fatalf("expected max order notional, got %s", resolved.MaxOrderNotional.String())
	}
	if !resolved.HasMaxPositionNotional() || resolved.MaxPositionNotional.String() != "1000000" {
		t.Fatalf("expected max position notional, got %s", resolved.MaxPositionNotional.String())
	}
	if resolved.HasMaxOrderQuantity() || resolved.HasMaxPositionQuantity() {
		t.Fatal("did not expect quantity caps when only notional caps were configured")
	}
}
