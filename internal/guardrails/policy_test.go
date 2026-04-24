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
