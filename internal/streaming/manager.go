package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/synthetixio/synthetix-go/wsinfo"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/metrics"
)

// Public channel identifiers surfaced to MCP clients. These are the
// values carried in subscription requests and in the `channel` field
// of notifications/event payloads delivered to bound MCP connections.
//
// Private/account streams were removed in Phase 1.4. The server no
// longer has an in-process source for per-subaccount events once the
// NATS consumer is gone, so accountEvents subscriptions are hard-
// rejected at the tool layer. A client that needs private streams
// must use the mcp-signer-bridge against /v1/ws/trade.
const (
	ChannelCandles      = "candles"
	ChannelMarketPrices = "marketPrices"
	ChannelOrderbook    = "orderbook"
	ChannelTrades       = "trades"
)

// SubscribeRequest is the shape the tool layer passes to Manager.Subscribe.
// Channel is one of the Channel* constants above; Params carries
// channel-specific fields (symbol, timeframe, depth).
type SubscribeRequest struct {
	Channel string
	Params  map[string]any
}

// Subscription is a single, normalized subscription owned by an MCP
// session. The zero-value is not meaningful; use normalizeSubscription
// to construct one.
type Subscription struct {
	Channel      string         `json:"channel"`
	Params       map[string]any `json:"params,omitempty"`
	SubscribedAt int64          `json:"subscribedAt"`
	key          string
}

// EventNotificationParams is the JSON-RPC params shape delivered to
// bound MCP connections via the "notifications/event" method.
type EventNotificationParams struct {
	Channel   string `json:"channel"`
	EventType string `json:"eventType,omitempty"`
	Data      any    `json:"data"`
}

const eventNotificationMethod = "notifications/event"

// EventStore wraps the SDK's in-memory event store with a callback so
// the server can clean up subscription state when the MCP session is
// closed.
type EventStore struct {
	base            mcp.EventStore
	onSessionClosed func(string)
}

func NewEventStore(onSessionClosed func(string)) *EventStore {
	return &EventStore{
		base:            mcp.NewMemoryEventStore(nil),
		onSessionClosed: onSessionClosed,
	}
}

func (s *EventStore) Open(ctx context.Context, sessionID, streamID string) error {
	return s.base.Open(ctx, sessionID, streamID)
}

func (s *EventStore) Append(ctx context.Context, sessionID, streamID string, data []byte) error {
	return s.base.Append(ctx, sessionID, streamID, data)
}

func (s *EventStore) After(ctx context.Context, sessionID, streamID string, index int) iter.Seq2[[]byte, error] {
	return s.base.After(ctx, sessionID, streamID, index)
}

func (s *EventStore) SessionClosed(ctx context.Context, sessionID string) error {
	if s.onSessionClosed != nil {
		s.onSessionClosed(sessionID)
	}
	return s.base.SessionClosed(ctx, sessionID)
}

// Manager owns per-session streaming subscription state and fans
// wsinfo notifications out to bound MCP connections.
//
// Concurrency: mu guards sessions and upstream. wsinfo.Client is
// safe for concurrent use. Handler callbacks from wsinfo run on the
// wsinfo delivery goroutine and must not block.
type Manager struct {
	cfg    *config.Config
	logger snx_lib_logging.Logger
	ws     *wsinfo.Client

	mu       sync.RWMutex
	sessions map[string]*sessionSubscriptions
	upstream map[string]*upstreamSub
}

// sessionSubscriptions is the per-session view: the bound MCP
// connection notifier (nil until BindSession fires) and the set of
// active subscription keys -> Subscription records.
type sessionSubscriptions struct {
	notifier      sessionEventNotifier
	subscriptions map[string]Subscription
}

// upstreamSub is the shared wsinfo subscription backing one or more
// session-level subscriptions that share the same key. Refcounted:
// when refs drops to zero, unsubscribe is called and the entry is
// dropped from Manager.upstream.
type upstreamSub struct {
	unsubscribe func()
	refs        int
}

type sessionEventNotifier interface {
	Notify(context.Context, EventNotificationParams) error
}

type connectionNotifier struct {
	conn mcp.Connection
}

func (n connectionNotifier) Notify(ctx context.Context, params EventNotificationParams) error {
	if n.conn == nil {
		return fmt.Errorf("connection is nil")
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal event notification: %w", err)
	}
	return n.conn.Write(ctx, &jsonrpc.Request{
		Method: eventNotificationMethod,
		Params: payload,
	})
}

// NewManager constructs a Manager. ws may be nil: when nil (e.g. when
// APIBaseURL is unset and the service is still wired to internal
// backends), subscriptions are accepted and tracked but no upstream
// events ever arrive. Tests that only exercise the registry can pass
// a nil client.
func NewManager(
	logger snx_lib_logging.Logger,
	cfg *config.Config,
	ws *wsinfo.Client,
) (*Manager, error) {
	return &Manager{
		cfg:      cfg,
		logger:   logger,
		ws:       ws,
		sessions: make(map[string]*sessionSubscriptions),
		upstream: make(map[string]*upstreamSub),
	}, nil
}

// Close tears down any upstream subscriptions this manager opened.
// Does not close the underlying wsinfo.Client; that is owned by
// backend.Clients and torn down alongside other backends.
func (m *Manager) Close() error {
	m.mu.Lock()
	up := m.upstream
	m.upstream = make(map[string]*upstreamSub)
	m.mu.Unlock()
	for _, s := range up {
		if s != nil && s.unsubscribe != nil {
			s.unsubscribe()
		}
	}
	return nil
}

// SessionClosed releases all subscription state for sessionID and
// decrements upstream refcounts (unsubscribing from wsinfo when the
// last local consumer goes away).
func (m *Manager) SessionClosed(sessionID string) {
	m.mu.Lock()
	state := m.sessions[sessionID]
	if state == nil {
		m.mu.Unlock()
		return
	}
	subCount := len(state.subscriptions)
	releases := make([]func(), 0, subCount)
	for _, sub := range state.subscriptions {
		if fn := m.releaseUpstreamLocked(sub.key); fn != nil {
			releases = append(releases, fn)
		}
	}
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	if subCount > 0 {
		metrics.ActiveSubscriptions().Sub(float64(subCount))
	}
	for _, fn := range releases {
		fn()
	}
}

// BindSession wires an MCP connection to a sessionID. Subsequent
// notifications for that session go out through the provided
// connection. Safe to call before or after Subscribe.
func (m *Manager) BindSession(sessionID string, conn mcp.Connection) {
	if sessionID == "" || conn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.sessions[sessionID]
	if state == nil {
		state = &sessionSubscriptions{subscriptions: make(map[string]Subscription)}
		m.sessions[sessionID] = state
	}
	state.notifier = connectionNotifier{conn: conn}
}

// ClearPrivateSubscriptions is a no-op kept for API compatibility
// with the pre-Phase-1.4 session-reset flow. Private streams no
// longer exist; authenticate() invokes this but there is nothing to
// clear. Retained so the tool-layer interface does not need to fork.
func (m *Manager) ClearPrivateSubscriptions(sessionID string) {}

// ActiveChannels returns the distinct channel names this session is
// subscribed to, in stable order (sorted by channel name).
func (m *Manager) ActiveChannels(sessionID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := m.sessions[sessionID]
	if state == nil {
		return []string{}
	}
	seen := make(map[string]struct{}, len(state.subscriptions))
	out := make([]string, 0, len(state.subscriptions))
	for _, sub := range state.subscriptions {
		if _, ok := seen[sub.Channel]; ok {
			continue
		}
		seen[sub.Channel] = struct{}{}
		out = append(out, sub.Channel)
	}
	slices.Sort(out)
	return out
}
