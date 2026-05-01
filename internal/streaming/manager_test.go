package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/synthetixio/synthetix-go/types"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
)

type fakeConnection struct {
	mu       sync.Mutex
	messages []jsonrpc.Message
}

func (f *fakeConnection) Read(context.Context) (jsonrpc.Message, error) { return nil, nil }

func (f *fakeConnection) Write(_ context.Context, msg jsonrpc.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msg)
	return nil
}

func (f *fakeConnection) Close() error { return nil }

func (f *fakeConnection) SessionID() string { return "session-1" }

func (f *fakeConnection) last() jsonrpc.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.messages) == 0 {
		return nil
	}
	return f.messages[len(f.messages)-1]
}

func (f *fakeConnection) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.messages)
}

func testStreamingConfig() *config.Config {
	return &config.Config{
		MaxSubscriptionsPerSession: 10,
		SessionTTL:                 30 * time.Minute,
	}
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager(snx_lib_logging_doubles.NewStubLogger(), testStreamingConfig(), nil)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return m
}

func decodeNotification(t *testing.T, msg jsonrpc.Message) EventNotificationParams {
	t.Helper()
	req, ok := msg.(*jsonrpc.Request)
	if !ok {
		t.Fatalf("expected JSON-RPC request, got %T", msg)
	}
	var params EventNotificationParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("unmarshal event params: %v", err)
	}
	return params
}

func TestNormalizeSubscription_PublicChannels(t *testing.T) {
	tests := []struct {
		name    string
		req     SubscribeRequest
		wantKey string
	}{
		{
			name:    "trades with symbol",
			req:     SubscribeRequest{Channel: ChannelTrades, Params: map[string]any{"symbol": "BTC-USDT"}},
			wantKey: "trades:BTC-USDT",
		},
		{
			name:    "trades defaults to ALL",
			req:     SubscribeRequest{Channel: ChannelTrades},
			wantKey: "trades:ALL",
		},
		{
			name:    "candles requires timeframe",
			req:     SubscribeRequest{Channel: ChannelCandles, Params: map[string]any{"symbol": "BTC-USDT", "timeframe": "1m"}},
			wantKey: "candles:BTC-USDT:1m",
		},
		{
			name:    "orderbook with explicit depth",
			req:     SubscribeRequest{Channel: ChannelOrderbook, Params: map[string]any{"symbol": "BTC-USDT", "depth": 50}},
			wantKey: "orderbook:BTC-USDT:50",
		},
		{
			name:    "orderbook default depth",
			req:     SubscribeRequest{Channel: ChannelOrderbook, Params: map[string]any{"symbol": "BTC-USDT"}},
			wantKey: "orderbook:BTC-USDT:20",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sub, _, err := normalizeSubscription(tc.req)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if sub.key != tc.wantKey {
				t.Fatalf("key = %q, want %q", sub.key, tc.wantKey)
			}
		})
	}
}

func TestNormalizeSubscription_Rejects(t *testing.T) {
	cases := []struct {
		name string
		req  SubscribeRequest
	}{
		{"unknown channel", SubscribeRequest{Channel: "accountEvents"}},
		{"candles without timeframe", SubscribeRequest{Channel: ChannelCandles, Params: map[string]any{"symbol": "BTC-USDT"}}},
		{"orderbook without symbol", SubscribeRequest{Channel: ChannelOrderbook}},
		{"orderbook depth out of range", SubscribeRequest{Channel: ChannelOrderbook, Params: map[string]any{"symbol": "BTC-USDT", "depth": 999}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := normalizeSubscription(tc.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSubscribe_TracksPerSessionStateAndReturnsWarnings(t *testing.T) {
	m := newTestManager(t)

	active, warnings, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelTrades},
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active subs, got %d", len(active))
	}
	// trades defaulted to ALL, so exactly one warning.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %#v", warnings)
	}
	channels := m.ActiveChannels("s1")
	if len(channels) != 2 {
		t.Fatalf("expected 2 distinct channels, got %#v", channels)
	}
}

func TestSubscribe_AllOrNothingOnValidationError(t *testing.T) {
	m := newTestManager(t)
	_, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
		{Channel: ChannelCandles, Params: map[string]any{"symbol": "BTC-USDT"}}, // missing timeframe
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if chans := m.ActiveChannels("s1"); len(chans) != 0 {
		t.Fatalf("expected no active channels after failed batch, got %#v", chans)
	}
}

func TestSubscribe_DedupesSameKeyWithinSession(t *testing.T) {
	m := newTestManager(t)
	active, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected deduped subscription, got %d", len(active))
	}
}

func TestUnsubscribe_RemovesMatchingChannelAndSymbol(t *testing.T) {
	m := newTestManager(t)
	if _, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "ETH-USDT"}},
		{Channel: ChannelTrades, Params: map[string]any{"symbol": "BTC-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	removed, remaining, err := m.Unsubscribe("s1", []string{ChannelMarketPrices}, "BTC-USDT")
	if err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(removed))
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining, got %d", len(remaining))
	}
}

func TestUnsubscribe_NoSymbolRemovesAcrossSymbols(t *testing.T) {
	m := newTestManager(t)
	if _, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "ETH-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	removed, _, err := m.Unsubscribe("s1", []string{ChannelMarketPrices}, "")
	if err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("expected 2 removed, got %d", len(removed))
	}
}

func TestSessionClosed_RemovesAllSubscriptions(t *testing.T) {
	m := newTestManager(t)
	if _, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelTrades, Params: map[string]any{"symbol": "BTC-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	m.SessionClosed("s1")
	if chans := m.ActiveChannels("s1"); len(chans) != 0 {
		t.Fatalf("expected no active channels, got %#v", chans)
	}
}

func TestBindAndFanOut_DeliversNotificationToBoundSession(t *testing.T) {
	m := newTestManager(t)
	conn := &fakeConnection{}
	m.BindSession("s1", conn)

	if _, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Simulate an upstream wsinfo push by decoding a priceUpdate
	// envelope and driving fanOut on the manager. The key must match
	// the normalized Subscription key.
	payload := types.WSPriceUpdateEvent{
		Symbol:    "BTC-USDT",
		MarkPrice: "42000",
		AskPrice:  "42001",
		BidPrice:  "41999",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	msg := &types.WSMessage{Channel: types.WSChannelPriceUpdate, Data: raw}
	m.onWSMessage("marketPrices:BTC-USDT", ChannelMarketPrices, msg)

	if conn.count() != 1 {
		t.Fatalf("expected 1 notification delivered, got %d", conn.count())
	}
	params := decodeNotification(t, conn.last())
	if params.Channel != ChannelMarketPrices {
		t.Fatalf("unexpected channel %q", params.Channel)
	}
}

func TestFanOut_SkipsSessionsWithoutMatchingKey(t *testing.T) {
	m := newTestManager(t)
	connA := &fakeConnection{}
	connB := &fakeConnection{}
	m.BindSession("sA", connA)
	m.BindSession("sB", connB)

	// sA subscribes BTC, sB subscribes ETH; BTC event should only
	// reach sA.
	if _, _, err := m.Subscribe("sA", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe sA: %v", err)
	}
	if _, _, err := m.Subscribe("sB", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "ETH-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe sB: %v", err)
	}

	raw, _ := json.Marshal(types.WSPriceUpdateEvent{Symbol: "BTC-USDT", MarkPrice: "42000"})
	m.onWSMessage("marketPrices:BTC-USDT", ChannelMarketPrices, &types.WSMessage{
		Channel: types.WSChannelPriceUpdate, Data: raw,
	})

	if connA.count() != 1 {
		t.Fatalf("expected sA to receive 1 notification, got %d", connA.count())
	}
	if connB.count() != 0 {
		t.Fatalf("expected sB to receive 0 notifications, got %d", connB.count())
	}
}

func TestFanOut_IgnoresMalformedPayload(t *testing.T) {
	m := newTestManager(t)
	conn := &fakeConnection{}
	m.BindSession("s1", conn)
	if _, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelMarketPrices, Params: map[string]any{"symbol": "BTC-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	m.onWSMessage("marketPrices:BTC-USDT", ChannelMarketPrices, &types.WSMessage{
		Channel: types.WSChannelPriceUpdate,
		Data:    []byte(`not a json object`),
	})
	if conn.count() != 0 {
		t.Fatalf("expected no notification for malformed payload, got %d", conn.count())
	}
}

func TestEventStore_ReplaysEventsAfterIndex(t *testing.T) {
	store := NewEventStore(nil)
	ctx := context.Background()
	if err := store.Open(ctx, "s1", "stream"); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Append(ctx, "s1", "stream", []byte("first")); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := store.Append(ctx, "s1", "stream", []byte("second")); err != nil {
		t.Fatalf("append 2: %v", err)
	}

	var replayed [][]byte
	for data, err := range store.After(ctx, "s1", "stream", 0) {
		if err != nil {
			t.Fatalf("replay: %v", err)
		}
		replayed = append(replayed, data)
	}
	if len(replayed) != 1 || string(replayed[0]) != "second" {
		t.Fatalf("expected [\"second\"], got %v", replayed)
	}
}

func TestEventStore_SessionClosedInvokesCallback(t *testing.T) {
	var got string
	store := NewEventStore(func(id string) { got = id })
	ctx := context.Background()
	if err := store.Open(ctx, "s1", "stream"); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.SessionClosed(ctx, "s1"); err != nil {
		t.Fatalf("session closed: %v", err)
	}
	if got != "s1" {
		t.Fatalf("expected callback for s1, got %q", got)
	}
}

func TestUnsubscribe_UnknownSessionIsNoop(t *testing.T) {
	m := newTestManager(t)
	removed, remaining, err := m.Unsubscribe("missing", []string{ChannelTrades}, "")
	if err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if len(removed) != 0 || len(remaining) != 0 {
		t.Fatalf("expected empty removed/remaining, got %d/%d", len(removed), len(remaining))
	}
}

func TestClearPrivateSubscriptions_IsNoop(t *testing.T) {
	m := newTestManager(t)
	if _, _, err := m.Subscribe("s1", []SubscribeRequest{
		{Channel: ChannelTrades, Params: map[string]any{"symbol": "BTC-USDT"}},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	m.ClearPrivateSubscriptions("s1") // must not disturb public subs
	if chans := m.ActiveChannels("s1"); len(chans) != 1 {
		t.Fatalf("expected 1 active channel, got %#v", chans)
	}
}

// notifierFailing exercises the fanOut error path. Failures in one
// subscriber must not prevent other subscribers from receiving the
// event.
type notifierFailing struct{ err error }

func (n notifierFailing) Notify(context.Context, EventNotificationParams) error { return n.err }

func TestFanOut_LoggingOnNotifierErrorDoesNotCrash(t *testing.T) {
	m := newTestManager(t)
	m.mu.Lock()
	m.sessions["s1"] = &sessionSubscriptions{
		notifier: notifierFailing{err: errors.New("boom")},
		subscriptions: map[string]Subscription{
			"marketPrices:BTC-USDT": {Channel: ChannelMarketPrices, key: "marketPrices:BTC-USDT"},
		},
	}
	m.mu.Unlock()
	// No panic, no propagation.
	m.fanOut("marketPrices:BTC-USDT", ChannelMarketPrices, "", map[string]any{"x": 1})
}
