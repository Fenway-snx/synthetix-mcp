package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"

	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/streaming"
)

func TestRegisterStreamingToolsRegistersExpectedTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	manager, err := streaming.NewManager(snx_lib_logging_doubles.NewStubLogger(), testAccountConfig(), nil)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}
	RegisterStreamingTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: &fakeSessionStore{sessions: map[string]*session.State{}},
	}, manager)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	toolsResult, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}

	expected := map[string]bool{
		"subscribe":   false,
		"unsubscribe": false,
	}
	for _, tool := range toolsResult.Tools {
		if _, ok := expected[tool.Name]; ok {
			expected[tool.Name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("expected tool %s to be registered", name)
		}
	}
}

func TestSubscribePublicChannelWithoutAuth(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	manager, err := streaming.NewManager(snx_lib_logging_doubles.NewStubLogger(), testAccountConfig(), nil)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterStreamingTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: store,
	}, manager)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "subscribe",
		Arguments: map[string]any{
			"subscriptions": []map[string]any{
				{"channel": "marketPrices"},
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful subscribe, got error: %#v", result.Content)
	}
	removed, remaining, err := manager.Unsubscribe(cs.ID(), []string{"marketPrices"}, "")
	if err != nil {
		t.Fatalf("manager unsubscribe failed: %v", err)
	}
	if len(removed) != 1 || len(remaining) != 0 {
		t.Fatalf("expected one active marketPrices subscription, got removed=%d remaining=%d", len(removed), len(remaining))
	}
}

func TestSubscribeAccountEventsRequiresAuthenticatedSession(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	manager, err := streaming.NewManager(snx_lib_logging_doubles.NewStubLogger(), testAccountConfig(), nil)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}
	RegisterStreamingTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: &fakeSessionStore{sessions: map[string]*session.State{}},
	}, manager)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "subscribe",
		Arguments: map[string]any{
			"subscriptions": []map[string]any{
				{"channel": "accountEvents"},
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected accountEvents subscription to fail closed without auth")
	}
}

func TestUnsubscribeRemovesMatchingSubscriptions(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	manager, err := streaming.NewManager(snx_lib_logging_doubles.NewStubLogger(), testAccountConfig(), nil)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}
	RegisterStreamingTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: &fakeSessionStore{sessions: map[string]*session.State{}},
	}, manager)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	_, err = cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "subscribe",
		Arguments: map[string]any{
			"subscriptions": []map[string]any{
				{"channel": "trades", "params": map[string]any{"symbol": "BTC-USDT"}},
				{"channel": "marketPrices", "params": map[string]any{"symbol": "BTC-USDT"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "unsubscribe",
		Arguments: map[string]any{
			"channels": []string{"trades"},
			"symbol":   "BTC-USDT",
		},
	})
	if err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful unsubscribe, got error: %#v", result.Content)
	}
	removed, remaining, err := manager.Unsubscribe(cs.ID(), []string{"marketPrices"}, "")
	if err != nil {
		t.Fatalf("manager unsubscribe failed: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected marketPrices to remain subscribed, got removed=%d", len(removed))
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no subscriptions after cleanup, got %d", len(remaining))
	}
}
