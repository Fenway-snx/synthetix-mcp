package tradeclosed

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/streaming"
)

type stubNotifier struct {
	mu     sync.Mutex
	events []streaming.EventNotificationParams
	err    error
}

func (s *stubNotifier) NotifySession(_ context.Context, sessionID string, p streaming.EventNotificationParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, p)
	return s.err
}

func (s *stubNotifier) snapshot() []streaming.EventNotificationParams {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]streaming.EventNotificationParams, len(s.events))
	copy(out, s.events)
	return out
}

type stubSubscriber struct {
	observers []risksnapshot.TransitionObserver
}

func (s *stubSubscriber) SubscribeTransitions(observer risksnapshot.TransitionObserver) {
	s.observers = append(s.observers, observer)
}

func (s *stubSubscriber) emit(t risksnapshot.PositionTransition) {
	for _, observer := range s.observers {
		observer(t)
	}
}

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal: %v", err)
	}
	return d
}

func TestWireRoutesClosedOnly(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	subscriber := &stubSubscriber{}
	notifier := &stubNotifier{}
	svc := Wire(subscriber, notifier, nil)
	if svc == nil {
		t.Fatal("Wire returned nil")
	}
	defer svc.Stop()

	subscriber.emit(risksnapshot.PositionTransition{
		Kind:         risksnapshot.TransitionClosed,
		SessionID:    "session-1",
		SubAccountID: 99,
		Symbol:       "BTC-USDT",
		Prior:        mustDec(t, "0.1"),
		Current:      decimal.Zero,
		ObservedAt:   time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	})
	subscriber.emit(risksnapshot.PositionTransition{
		Kind:         risksnapshot.TransitionOpened,
		SessionID:    "session-1",
		SubAccountID: 99,
		Symbol:       "ETH-USDT",
		Prior:        decimal.Zero,
		Current:      mustDec(t, "1"),
	})
	subscriber.emit(risksnapshot.PositionTransition{
		Kind:         risksnapshot.TransitionAdjusted,
		SessionID:    "session-1",
		SubAccountID: 99,
		Symbol:       "SOL-USDT",
		Prior:        mustDec(t, "5"),
		Current:      mustDec(t, "8"),
	})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(notifier.snapshot()) >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	events := notifier.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 routed event (the CLOSE); got %d: %+v", len(events), events)
	}
	if events[0].Channel != EventChannel {
		t.Errorf("channel = %q; want %q", events[0].Channel, EventChannel)
	}
	if events[0].EventType != "position.closed" {
		t.Errorf("eventType = %q; want position.closed", events[0].EventType)
	}
	payload, ok := events[0].Data.(PositionClosedEvent)
	if !ok {
		t.Fatalf("payload is %T; want PositionClosedEvent", events[0].Data)
	}
	if payload.Symbol != "BTC-USDT" {
		t.Errorf("symbol = %q; want BTC-USDT", payload.Symbol)
	}
	if payload.PriorSide != "LONG" {
		t.Errorf("PriorSide = %q; want LONG", payload.PriorSide)
	}
	if !strings.Contains(payload.Card, "POSITION CLOSED") {
		t.Errorf("card missing POSITION CLOSED title:\n%s", payload.Card)
	}
	if !strings.Contains(payload.Card, "BTC-USDT") {
		t.Errorf("card missing symbol:\n%s", payload.Card)
	}
	if len(payload.FollowUp) == 0 {
		t.Errorf("FollowUp should include remediation hints")
	}
}

func TestWireSkipsCardWhenDisabled(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "false")

	subscriber := &stubSubscriber{}
	notifier := &stubNotifier{}
	svc := Wire(subscriber, notifier, nil)
	defer svc.Stop()

	subscriber.emit(risksnapshot.PositionTransition{
		Kind:      risksnapshot.TransitionClosed,
		SessionID: "s",
		Symbol:    "BTC-USDT",
		Prior:     mustDec(t, "0.1"),
	})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(notifier.snapshot()) >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	events := notifier.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 event; got %d", len(events))
	}
	payload := events[0].Data.(PositionClosedEvent)
	if payload.Card != "" {
		t.Errorf("Card should be empty when SNXMCP_CARDS_ENABLED=false; got:\n%s", payload.Card)
	}
}

func TestWireDropsOnEmptySessionID(t *testing.T) {
	subscriber := &stubSubscriber{}
	notifier := &stubNotifier{}
	svc := Wire(subscriber, notifier, nil)
	defer svc.Stop()

	subscriber.emit(risksnapshot.PositionTransition{
		Kind:   risksnapshot.TransitionClosed,
		Symbol: "BTC-USDT",
		Prior:  mustDec(t, "0.1"),
	})

	time.Sleep(50 * time.Millisecond)
	if got := len(notifier.snapshot()); got != 0 {
		t.Errorf("expected no notifications when SessionID is empty; got %d", got)
	}
}
