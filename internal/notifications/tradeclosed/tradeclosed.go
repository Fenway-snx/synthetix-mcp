// Package tradeclosed connects position-close detection to MCP notifications.
// It stays thin so snapshot, streaming, and card layers remain independent.
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

// MCP event channel value used for closed-position notifications.
const EventChannel = "trade.closed"

// Minimal streaming surface needed for notification dispatch.
type Notifier interface {
	NotifySession(ctx context.Context, sessionID string, params streaming.EventNotificationParams) error
}

// Minimal snapshot surface needed to subscribe to transitions.
type TransitionSubscriber interface {
	SubscribeTransitions(observer risksnapshot.TransitionObserver)
}

// Owns the dispatch queue between transition events and notifications.
type Service struct {
	notifier Notifier
	logger   snx_lib_logging.Logger

	mu     sync.Mutex
	closed bool
	queue  chan risksnapshot.PositionTransition
	done   chan struct{}
}

// Registers transition observation and starts asynchronous dispatch.
// The bounded queue prevents slow MCP writes from blocking hydration.
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

// Closes the dispatch queue and waits for drain. Idempotent.
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

// JSON shape carried inside the streaming data field.
// The card sits beside structured fields for visual clients.
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

// Renders a compact close notice without fabricating realized PnL.
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
