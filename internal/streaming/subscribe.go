package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	"github.com/synthetixio/synthetix-go/types"
	"github.com/synthetixio/synthetix-go/wsinfo"
)

// Subscribe registers a batch of subscriptions for sessionID. Returns
// the full active subscription list (after the batch is applied), any
// per-request warnings (e.g. symbol defaulted to ALL, rate-cap hit),
// and an error only if one of the requests was invalid — in which
// case state is unchanged (all-or-nothing within a batch).
func (m *Manager) Subscribe(sessionID string, requests []SubscribeRequest) ([]Subscription, []string, error) {
	if sessionID == "" {
		return nil, nil, fmt.Errorf("session ID is required")
	}

	// Normalize everything first so a bad request in the batch leaves
	// state untouched. Validation is pure so we can do it without a
	// lock; upstream wsinfo.Subscribe calls happen later under mu.
	normalized := make([]Subscription, 0, len(requests))
	warnings := make([]string, 0)
	for _, req := range requests {
		sub, warning, err := normalizeSubscription(req)
		if err != nil {
			return nil, warnings, err
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
		normalized = append(normalized, sub)
	}

	m.mu.Lock()

	state := m.sessions[sessionID]
	if state == nil {
		state = &sessionSubscriptions{subscriptions: make(map[string]Subscription)}
		m.sessions[sessionID] = state
	}

	// Apply the batch honoring the per-session cap. Track which
	// upstream keys we need to acquire so we can call wsinfo.Subscribe
	// *after* releasing the lock (wsinfo.Subscribe may block briefly
	// waiting for a conn).
	toAcquire := make([]Subscription, 0, len(normalized))
	for _, sub := range normalized {
		if _, exists := state.subscriptions[sub.key]; exists {
			continue
		}
		if len(state.subscriptions) >= m.cfg.MaxSubscriptionsPerSession {
			warnings = append(warnings, fmt.Sprintf(
				"max subscriptions per session reached (%d); skipped %s",
				m.cfg.MaxSubscriptionsPerSession, sub.Channel))
			continue
		}
		state.subscriptions[sub.key] = sub
		if existing, ok := m.upstream[sub.key]; ok {
			existing.refs++
		} else {
			toAcquire = append(toAcquire, sub)
		}
	}

	active := sortedSubscriptions(state.subscriptions)
	m.mu.Unlock()

	// Open upstream subscriptions for every newly-unique key. If
	// wsinfo is nil (test / Phase 1.1 fallback), skip silently —
	// local state is still tracked and Unsubscribe still works.
	for _, sub := range toAcquire {
		m.acquireUpstream(sub)
	}

	return active, warnings, nil
}

// Unsubscribe removes matching subscriptions from sessionID. A
// subscription matches if its Channel is in channels and, when symbol
// is non-empty, its symbol param equals symbol. Returns the removed
// subscriptions and the remaining active set.
func (m *Manager) Unsubscribe(sessionID string, channels []string, symbol string) ([]Subscription, []Subscription, error) {
	if sessionID == "" {
		return nil, nil, fmt.Errorf("session ID is required")
	}

	channelSet := make(map[string]struct{}, len(channels))
	for _, c := range channels {
		channelSet[strings.TrimSpace(c)] = struct{}{}
	}

	m.mu.Lock()
	state := m.sessions[sessionID]
	if state == nil {
		m.mu.Unlock()
		return nil, nil, nil
	}

	removed := make([]Subscription, 0)
	releases := make([]func(), 0)
	for key, sub := range state.subscriptions {
		if _, ok := channelSet[sub.Channel]; !ok {
			continue
		}
		if symbol != "" {
			if subSymbol, ok := subscriptionSymbol(sub); !ok || subSymbol != symbol {
				continue
			}
		}
		removed = append(removed, sub)
		delete(state.subscriptions, key)
		if fn := m.releaseUpstreamLocked(key); fn != nil {
			releases = append(releases, fn)
		}
	}
	remaining := sortedSubscriptions(state.subscriptions)
	m.pruneSessionIfIdleLocked(sessionID, state)
	m.mu.Unlock()

	for _, fn := range releases {
		fn()
	}
	return removed, remaining, nil
}

func (m *Manager) pruneSessionIfIdleLocked(sessionID string, state *sessionSubscriptions) {
	if state == nil {
		delete(m.sessions, sessionID)
		return
	}
	if state.notifier != nil {
		return
	}
	if len(state.subscriptions) != 0 {
		return
	}
	delete(m.sessions, sessionID)
}

// releaseUpstreamLocked decrements the upstream refcount for key. If
// the refcount hits zero, the entry is removed from the map and the
// returned func (non-nil) must be invoked *outside* the lock to
// actually close the wsinfo subscription (wsinfo.Subscribe's unsub
// callback takes its own lock; nesting ours inside risks deadlocks
// against wsinfo's notify path).
//
// Caller must hold m.mu.
func (m *Manager) releaseUpstreamLocked(key string) func() {
	up, ok := m.upstream[key]
	if !ok {
		return nil
	}
	up.refs--
	if up.refs > 0 {
		return nil
	}
	delete(m.upstream, key)
	return up.unsubscribe
}

// acquireUpstream opens a wsinfo subscription for sub, wiring its
// handler into Manager.fanOut. Called outside m.mu (wsinfo.Subscribe
// may block briefly waiting for the upstream conn).
//
// On error, a warning is logged and the upstream map entry is
// removed; the local session subscription remains registered so
// Unsubscribe still works. Events will never flow for that key
// unless the caller re-subscribes.
func (m *Manager) acquireUpstream(sub Subscription) {
	if m.ws == nil {
		return
	}
	spec, ok := wsinfoSpec(sub)
	if !ok {
		return
	}
	key := sub.key
	handler := func(msg *types.WSMessage) { m.onWSMessage(key, sub.Channel, msg) }

	unsub, err := m.ws.Subscribe(context.Background(), spec, handler)
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("wsinfo subscribe failed",
				"channel", sub.Channel, "key", key, "error", err)
		}
		m.mu.Lock()
		delete(m.upstream, key)
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	if existing, ok := m.upstream[key]; ok {
		// Rare: another Subscribe raced to open the same key and won.
		// Increment existing refcount and drop our duplicate.
		existing.refs++
		m.mu.Unlock()
		unsub()
		return
	}
	m.upstream[key] = &upstreamSub{unsubscribe: unsub, refs: 1}
	m.mu.Unlock()
}

// normalizeSubscription validates and canonicalizes req, returning a
// Subscription with a stable dedup key. On validation error, returns
// a zero Subscription and the error.
func normalizeSubscription(req SubscribeRequest) (Subscription, string, error) {
	channel := strings.TrimSpace(req.Channel)
	params := req.Params
	if params == nil {
		params = map[string]any{}
	}
	sub := Subscription{
		Channel:      channel,
		Params:       map[string]any{},
		SubscribedAt: snx_lib_utils_time.Now().UnixMilli(),
	}

	switch channel {
	case ChannelCandles:
		symbol, err := requiredCanonicalSymbol(params, "symbol")
		if err != nil {
			return Subscription{}, "", err
		}
		timeframe, ok := params["timeframe"].(string)
		if !ok || strings.TrimSpace(timeframe) == "" {
			return Subscription{}, "", fmt.Errorf("candles subscription requires timeframe")
		}
		sub.Params["symbol"] = symbol
		sub.Params["timeframe"] = timeframe
		sub.key = fmt.Sprintf("%s:%s:%s", channel, symbol, timeframe)
	case ChannelMarketPrices, ChannelTrades:
		symbol, warning, err := optionalSymbolOrAll(params)
		if err != nil {
			return Subscription{}, "", err
		}
		sub.Params["symbol"] = symbol
		sub.key = fmt.Sprintf("%s:%s", channel, symbol)
		return sub, warning, nil
	case ChannelOrderbook:
		symbol, err := requiredCanonicalSymbol(params, "symbol")
		if err != nil {
			return Subscription{}, "", err
		}
		depth, err := optionalInt(params, "depth", 20, 1, 100)
		if err != nil {
			return Subscription{}, "", err
		}
		sub.Params["symbol"] = symbol
		sub.Params["depth"] = depth
		sub.key = fmt.Sprintf("%s:%s:%d", channel, symbol, depth)
	default:
		return Subscription{}, "", fmt.Errorf("unsupported channel %q", channel)
	}

	return sub, "", nil
}

// wsinfoSpec maps an MCP-level Subscription onto a wsinfo.SubscribeSpec.
// Returns (_, false) for channels where symbol="ALL" — wsinfo does
// not support a wildcard symbol, so an ALL-symbol MCP subscription
// would need one wsinfo sub per live symbol. Phase 1.4 scope: we
// deliver events for ALL-symbol subscriptions only when the caller
// has ALSO subscribed to a specific symbol on the same channel (rare
// in practice), and rely on per-symbol subscriptions to drive the
// stream otherwise. This intentionally drops ALL-symbol fan-out in
// the new backend; a future phase can add a symbol enumerator.
func wsinfoSpec(sub Subscription) (wsinfo.SubscribeSpec, bool) {
	switch sub.Channel {
	case ChannelTrades:
		symbol, _ := sub.Params["symbol"].(string)
		if symbol == "" || symbol == "ALL" {
			return wsinfo.SubscribeSpec{}, false
		}
		return wsinfo.SubscribeSpec{Type: types.WSSubscribeTrade, Symbol: symbol}, true
	case ChannelMarketPrices:
		symbol, _ := sub.Params["symbol"].(string)
		if symbol == "" || symbol == "ALL" {
			return wsinfo.SubscribeSpec{}, false
		}
		return wsinfo.SubscribeSpec{Type: types.WSSubscribePrice, Symbol: symbol}, true
	case ChannelCandles:
		symbol, _ := sub.Params["symbol"].(string)
		timeframe, _ := sub.Params["timeframe"].(string)
		if symbol == "" || timeframe == "" {
			return wsinfo.SubscribeSpec{}, false
		}
		return wsinfo.SubscribeSpec{
			Type:      types.WSSubscribeCandle,
			Symbol:    symbol,
			Timeframe: timeframe,
		}, true
	case ChannelOrderbook:
		symbol, _ := sub.Params["symbol"].(string)
		depth, _ := sub.Params["depth"].(int)
		if symbol == "" {
			return wsinfo.SubscribeSpec{}, false
		}
		return wsinfo.SubscribeSpec{
			Type:   types.WSSubscribeOrderbook,
			Symbol: symbol,
			Depth:  depth,
			Format: "diff",
		}, true
	}
	return wsinfo.SubscribeSpec{}, false
}

func requiredCanonicalSymbol(params map[string]any, field string) (string, error) {
	raw, ok := params[field].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if err := snx_lib_api_validation.ValidateCanonicalSymbol(snx_lib_api_types.Symbol(raw), field); err != nil {
		return "", err
	}
	return raw, nil
}

func optionalSymbolOrAll(params map[string]any) (string, string, error) {
	raw, _ := params["symbol"].(string)
	if strings.TrimSpace(raw) == "" {
		return "ALL", "symbol omitted; subscribing to ALL symbols for this channel", nil
	}
	if strings.EqualFold(raw, "ALL") {
		return "ALL", "", nil
	}
	if err := snx_lib_api_validation.ValidateCanonicalSymbol(snx_lib_api_types.Symbol(raw), "symbol"); err != nil {
		return "", "", err
	}
	return raw, "", nil
}

func optionalInt(params map[string]any, field string, def, min, max int) (int, error) {
	raw, ok := params[field]
	if !ok {
		return def, nil
	}
	switch v := raw.(type) {
	case float64:
		if v != float64(int(v)) {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
		i := int(v)
		if i < min || i > max {
			return 0, fmt.Errorf("%s must be between %d and %d", field, min, max)
		}
		return i, nil
	case int:
		if v < min || v > max {
			return 0, fmt.Errorf("%s must be between %d and %d", field, min, max)
		}
		return v, nil
	default:
		return 0, fmt.Errorf("%s must be an integer", field)
	}
}

func subscriptionSymbol(sub Subscription) (string, bool) {
	value, ok := sub.Params["symbol"].(string)
	return value, ok
}

func sortedSubscriptions(subs map[string]Subscription) []Subscription {
	out := make([]Subscription, 0, len(subs))
	for _, sub := range subs {
		out = append(out, sub)
	}
	// Stable key-sorted order so callers comparing snapshots see
	// deterministic output.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].key > out[j].key; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// onWSMessage is the wsinfo delivery callback. It decodes the channel
// payload, selects the target symbol (or timeframe) for fan-out, and
// dispatches to every local session subscribed to the matching key.
//
// Runs on the wsinfo deliver goroutine. Must not block on external
// I/O; notifications go out via session notifiers which are
// best-effort (errors are logged, not propagated).
func (m *Manager) onWSMessage(upstreamKey, channel string, msg *types.WSMessage) {
	if msg == nil {
		return
	}
	payload, eventType, ok := decodeWSPayload(channel, msg)
	if !ok {
		return
	}
	m.fanOut(upstreamKey, channel, eventType, payload)
}

func (m *Manager) fanOut(upstreamKey, channel, eventType string, data any) {
	m.mu.RLock()
	notifiers := make([]sessionEventNotifier, 0)
	ids := make([]string, 0)
	for sessionID, state := range m.sessions {
		if _, subscribed := state.subscriptions[upstreamKey]; !subscribed {
			continue
		}
		if state.notifier == nil {
			continue
		}
		notifiers = append(notifiers, state.notifier)
		ids = append(ids, sessionID)
	}
	m.mu.RUnlock()

	if len(notifiers) == 0 {
		return
	}
	params := EventNotificationParams{Channel: channel, EventType: eventType, Data: data}
	for i, n := range notifiers {
		if err := n.Notify(context.Background(), params); err != nil && m.logger != nil {
			m.logger.Warn("failed to notify subscriber",
				"error", err, "channel", channel, "session_id", ids[i])
		}
	}
}

// decodeWSPayload picks the per-channel payload type, decodes into
// it, and returns the result (as `any`) alongside the MCP-level
// eventType label. Returns ok=false when the message is malformed or
// when the channel is unknown.
func decodeWSPayload(channel string, msg *types.WSMessage) (any, string, bool) {
	switch channel {
	case ChannelTrades:
		var ev types.WSTradeEvent
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			return nil, "", false
		}
		return ev, "", true
	case ChannelMarketPrices:
		var ev types.WSPriceUpdateEvent
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			return nil, "", false
		}
		return ev, "", true
	case ChannelCandles:
		var ev types.WSCandleEvent
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			return nil, "", false
		}
		return ev, "", true
	case ChannelOrderbook:
		var ev types.WSOrderbookEvent
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			return nil, "", false
		}
		return ev, "", true
	}
	return nil, "", false
}
