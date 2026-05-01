// Package tradeclosed wires the risksnapshot.Manager's
// position-transition observer to the streaming.Manager's per-session
// notifier so agents receive an MCP notifications/event when a
// position closes (nonzero → zero).
//
// This is intentionally a thin glue layer: detection lives in
// risksnapshot, notification transport lives in streaming, and card
// rendering lives in the `cards` package. The three are stitched
// together here so neither layer learns about the other.
package tradeclosed

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/streaming"
)

// EventChannel is the MCP `channel` value carried on the event
// notification when a position closes. Clients filter on this to
// route the notification into a UI or chat side-panel.
const EventChannel = "trade.closed"

// Notifier is the subset of streaming.Manager that this package needs.
// Defined as an interface so the package can be unit tested without a
// full streaming Manager + websocket stack.
type Notifier interface {
	NotifySession(ctx context.Context, sessionID string, params streaming.EventNotificationParams) error
}

// TransitionSubscriber is the subset of risksnapshot.Manager this
// package consumes — only the subscription hook.
type TransitionSubscriber interface {
	SubscribeTransitions(observer risksnapshot.TransitionObserver)
}

// Service stitches the two managers together. Construct via Wire
// and call Stop on shutdown so the dispatch goroutine drains.
type Service struct {
	notifier Notifier
	logger   snx_lib_logging.Logger

	mu     sync.Mutex
	closed bool
	queue  chan risksnapshot.PositionTransition
	done   chan struct{}
}

// Wire registers a transition observer on src that pushes
// trade-closed notifications to notifier. Returns the Service that
// owns the dispatch goroutine; callers should call Stop on shutdown.
//
// The observer fires synchronously while risksnapshot holds no
// internal locks, but we still hand off to a goroutine so a slow
// notifier (e.g. a stalled MCP write) cannot back-pressure a
// hydration call. The queue is bounded to keep memory predictable;
// drops are logged at warn level so operators can see when an agent
// connection has gone unresponsive.
func Wire(src TransitionSubscriber, notifier Notifier, logger snx_lib_logging.Logger) *Service {
	if src == nil || notifier == nil {
		return nil
	}
	s := &Service{
		notifier: notifier,
		logger:   logger,
		queue:    make(chan risksnapshot.PositionTransition, 64),
		done:     make(chan struct{}),
	}
	go s.run()

	src.SubscribeTransitions(func(t risksnapshot.PositionTransition) {
		if t.Kind != risksnapshot.TransitionClosed {
			return
		}
		s.enqueue(t)
	})
	return s
}

// Stop closes the dispatch queue and waits for the goroutine to
// drain. Idempotent.
func (s *Service) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.queue)
	s.mu.Unlock()
	<-s.done
}

func (s *Service) enqueue(t risksnapshot.PositionTransition) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	select {
	case s.queue <- t:
	default:
		if s.logger != nil {
			s.logger.Warn(
				"trade-closed notification queue full; dropping event",
				"sessionId", t.SessionID,
				"subAccountId", t.SubAccountID,
				"symbol", t.Symbol,
			)
		}
	}
	s.mu.Unlock()
}

func (s *Service) run() {
	defer close(s.done)
	for t := range s.queue {
		s.dispatch(t)
	}
}

func (s *Service) dispatch(t risksnapshot.PositionTransition) {
	if t.SessionID == "" {
		return
	}
	payload := buildPayload(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.notifier.NotifySession(ctx, t.SessionID, payload); err != nil {
		if s.logger != nil {
			s.logger.Warn(
				"trade-closed notification failed",
				"sessionId", t.SessionID,
				"subAccountId", t.SubAccountID,
				"symbol", t.Symbol,
				"err", err,
			)
		}
	}
}

// PositionClosedEvent is the JSON shape carried inside the
// streaming `data` field. The card lives alongside as a `card`
// string so agents that ignore visual content can still parse the
// structured fields.
type PositionClosedEvent struct {
	SubAccountID int64     `json:"subAccountId"`
	Symbol       string    `json:"symbol"`
	PriorSide    string    `json:"priorSide"`
	PriorSize    string    `json:"priorSize"`
	ObservedAt   time.Time `json:"observedAt"`
	Card         string    `json:"card,omitempty"`
	FollowUp     []string  `json:"followUp"`
}

func buildPayload(t risksnapshot.PositionTransition) streaming.EventNotificationParams {
	side := sideFromQuantity(t.Prior)
	priorSize := t.Prior.Abs().String()
	event := PositionClosedEvent{
		SubAccountID: t.SubAccountID,
		Symbol:       t.Symbol,
		PriorSide:    side,
		PriorSize:    priorSize,
		ObservedAt:   t.ObservedAt,
		FollowUp: []string{
			"Read account://trade-journal for realized PnL on the closed position.",
			fmt.Sprintf("Call get_trade_history with symbol=%s for the underlying fills.", t.Symbol),
		},
	}
	event.Card = renderClosedCard(t)
	return streaming.EventNotificationParams{
		Channel:   EventChannel,
		EventType: "position.closed",
		Data:      event,
	}
}

// renderClosedCard is intentionally compact: we don't have realized
// PnL on the snapshot alone, so the card is a "go look up details"
// hint rather than a duplicate of the close_position PnL card. The
// agent's followUp instructions point at the journal resource.
func renderClosedCard(t risksnapshot.PositionTransition) string {
	if !cards.Enabled() {
		return ""
	}
	side := sideFromQuantity(t.Prior)
	arrow := "▲"
	if side == "SHORT" {
		arrow = "▼"
	}
	priorSize, _ := t.Prior.Abs().Float64()
	return cards.Card{
		Status: cards.StatusNeutral,
		Title:  "POSITION CLOSED  " + arrow + " " + side + " " + t.Symbol,
		Sections: []cards.Section{
			{Rows: []cards.Row{
				{Label: "Symbol:", Value: t.Symbol, Hint: ""},
				{Label: "Prior side:", Value: side, Hint: ""},
				{Label: "Prior size:", Value: trimDecimal(priorSize), Hint: baseAsset(t.Symbol)},
				{Label: "Observed:", Value: cards.TimestampUTC(t.ObservedAt), Hint: ""},
			}},
			{Rows: []cards.Row{
				{Label: "Next:", Value: "read account://trade-journal", Hint: "for realized PnL"},
			}},
		},
	}.Render()
}

func sideFromQuantity(qty decimal.Decimal) string {
	if qty.Sign() < 0 {
		return "SHORT"
	}
	return "LONG"
}

func baseAsset(symbol string) string {
	for _, sep := range []string{"-", "/", "_"} {
		if idx := strings.Index(symbol, sep); idx > 0 {
			return symbol[:idx]
		}
	}
	return ""
}

func trimDecimal(v float64) string {
	abs := v
	if abs < 0 {
		abs = -abs
	}
	s := decimal.NewFromFloat(abs).StringFixed(6)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}
