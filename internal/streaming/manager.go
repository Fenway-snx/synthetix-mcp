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

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/synthetixio/synthetix-go/wsinfo"
)

// Public channel identifiers surfaced to MCP clients.
// Private/account streams are rejected at the tool layer.
const (
	ChannelCandles      = "candles"
	ChannelMarketPrices = "marketPrices"
	ChannelOrderbook    = "orderbook"
	ChannelTrades       = "trades"
)

// Shape passed from the tool layer into the subscription manager.
type SubscribeRequest struct {
	Channel string
	Params  map[string]any
}

// Single normalized subscription owned by an MCP session.
type Subscription struct {
	Channel      string         `json:"channel"`
	Params       map[string]any `json:"params,omitempty"`
	SubscribedAt int64          `json:"subscribedAt"`
	key          string
}

// JSON-RPC params delivered via the notifications/event method.
type EventNotificationParams struct {
	Channel   string `json:"channel"`
	EventType string `json:"eventType,omitempty"`
	Data      any    `json:"data"`
}

const eventNotificationMethod = "notifications/event"

// Wraps the SDK event store with session-close cleanup.
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

// Owns per-session subscription state and fans out upstream events.
// The mutex guards sessions and upstream subscriptions.
type Manager struct {
	cfg    *config.Config
	logger snx_lib_logging.Logger
	ws     *wsinfo.Client

	mu       sync.RWMutex
	sessions map[string]*sessionSubscriptions
	upstream map[string]*upstreamSub
}

// Per-session notifier plus active subscriptions.
type sessionSubscriptions struct {
	notifier      sessionEventNotifier
	subscriptions map[string]Subscription
}

// Shared upstream subscription backing one or more local subscriptions.
// The refcount controls unsubscribe.
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

// Constructs a subscription manager.
// A nil upstream client tracks subscriptions without receiving events.
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

// Tears down upstream subscriptions opened by this manager.
// The upstream client remains owned by backend clients.
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

// Releases all subscription state for a closed session.
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

	for _, fn := range releases {
		fn()
	}
}

// Wires an MCP connection to a session for later event delivery.
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

// Sends one notifications/event payload to a bound session connection.
// Returns an error when no connection is available or the write fails.
func (m *Manager) NotifySession(ctx context.Context, sessionID string, params EventNotificationParams) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	m.mu.RLock()
	state := m.sessions[sessionID]
	var notifier sessionEventNotifier
	if state != nil {
		notifier = state.notifier
	}
	m.mu.RUnlock()
	if notifier == nil {
		return fmt.Errorf("session %s has no bound connection", sessionID)
	}
	return notifier.Notify(ctx, params)
}

// No-op compatibility hook for old session-reset wiring.
func (m *Manager) ClearPrivateSubscriptions(sessionID string) {}

// Returns distinct subscribed channel names in stable order.
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
