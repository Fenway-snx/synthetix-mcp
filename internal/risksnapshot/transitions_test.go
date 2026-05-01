package risksnapshot

import (
	"testing"

	"github.com/shopspring/decimal"
)

func decN(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal: %v", err)
	}
	return d
}

func TestDiffPositionTransitionsCloseOpenFlipAdjust(t *testing.T) {
	prior := &Snapshot{
		positionsBySymbol: map[string]decimal.Decimal{
			"BTC-USDT": decN(t, "0.1"),
			"ETH-USDT": decN(t, "-1"),
			"SOL-USDT": decN(t, "5"),
		},
	}
	current := &Snapshot{
		positionsBySymbol: map[string]decimal.Decimal{
			"ETH-USDT":  decN(t, "1"),
			"SOL-USDT":  decN(t, "8"),
			"DOGE-USDT": decN(t, "100"),
		},
		refreshedAtMs: 1714560000000,
	}

	transitions := diffPositionTransitionsLocked(prior, current, "session-1", 42)
	byKind := map[TransitionKind]string{}
	for _, t := range transitions {
		byKind[t.Kind] = t.Symbol
	}
	if byKind[TransitionClosed] != "BTC-USDT" {
		t.Errorf("expected BTC-USDT closed; got %v", byKind)
	}
	if byKind[TransitionFlipped] != "ETH-USDT" {
		t.Errorf("expected ETH-USDT flipped; got %v", byKind)
	}
	if byKind[TransitionAdjusted] != "SOL-USDT" {
		t.Errorf("expected SOL-USDT adjusted; got %v", byKind)
	}
	if byKind[TransitionOpened] != "DOGE-USDT" {
		t.Errorf("expected DOGE-USDT opened; got %v", byKind)
	}
}

func TestDiffPositionTransitionsHonorsSessionFields(t *testing.T) {
	current := &Snapshot{positionsBySymbol: map[string]decimal.Decimal{}, refreshedAtMs: 1}
	prior := &Snapshot{positionsBySymbol: map[string]decimal.Decimal{"BTC-USDT": decN(t, "0.5")}}

	out := diffPositionTransitionsLocked(prior, current, "abc", 7)
	if len(out) != 1 {
		t.Fatalf("expected 1 transition; got %d", len(out))
	}
	if out[0].SessionID != "abc" || out[0].SubAccountID != 7 {
		t.Errorf("session/subaccount not propagated: %+v", out[0])
	}
}

func TestSubscribeTransitionsFiresOnClose(t *testing.T) {
	m := NewManager(nil)
	got := []PositionTransition{}
	m.SubscribeTransitions(func(t PositionTransition) {
		got = append(got, t)
	})

	prior := &Snapshot{positionsBySymbol: map[string]decimal.Decimal{"BTC-USDT": decN(t, "0.1")}}
	current := &Snapshot{positionsBySymbol: map[string]decimal.Decimal{}, refreshedAtMs: 1}

	transitions := diffPositionTransitionsLocked(prior, current, "s", 1)
	if len(transitions) != 1 {
		t.Fatalf("setup: expected 1 transition; got %d", len(transitions))
	}

	for _, transition := range transitions {
		for _, observer := range m.observers {
			observer(transition)
		}
	}

	if len(got) != 1 {
		t.Fatalf("observer fired %d times; want 1", len(got))
	}
	if got[0].Kind != TransitionClosed || got[0].Symbol != "BTC-USDT" {
		t.Errorf("unexpected transition: %+v", got[0])
	}
}
