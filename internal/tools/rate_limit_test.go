package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/server/backend"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/streaming"
)

type fakeToolRateLimiter struct {
	batchSize     int
	err           error
	operationName string
	state         *session.State
}

func (f *fakeToolRateLimiter) Check(_ context.Context, operationName string, batchSize int, state *session.State) error {
	f.operationName = operationName
	f.batchSize = batchSize
	f.state = state
	return f.err
}

func TestPingReturnsStructuredRateLimitError(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	RegisterPublicTools(server, &ToolDeps{
		Cfg:     testAccountConfig(),
		Clients: &backend.Clients{},
		Store:   &fakeSessionStore{sessions: map[string]*session.State{}},
		Limiter: &fakeToolRateLimiter{err: &rateLimitExceededError{
			appliedLimit: 10,
			scope:        "ip",
			toolName:     "ping",
		}},
	})

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

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "ping"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected rate-limited ping to fail")
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	if !strings.Contains(text.Text, `"error":"RATE_LIMIT_EXCEEDED"`) {
		t.Fatalf("expected RATE_LIMIT_EXCEEDED payload, got %s", text.Text)
	}
	if !strings.Contains(text.Text, `"httpStatusCode":429`) {
		t.Fatalf("expected 429 details in payload, got %s", text.Text)
	}
}

func TestSubscribeScalesRateLimitByRequestedSubscriptions(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	limiter := &fakeToolRateLimiter{}
	manager, err := streaming.NewManager(snx_lib_logging_doubles.NewStubLogger(), testAccountConfig(), nil)
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	RegisterStreamingTools(server, &ToolDeps{
		Cfg:     testAccountConfig(),
		Store:   &fakeSessionStore{sessions: map[string]*session.State{}},
		Limiter: limiter,
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
				{"channel": "trades", "params": map[string]any{"symbol": "BTC-USDT"}},
				{"channel": "orderbook", "params": map[string]any{"symbol": "BTC-USDT"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful subscribe, got error: %#v", result.Content)
	}
	if limiter.operationName != "subscribe" {
		t.Fatalf("expected subscribe operation name, got %q", limiter.operationName)
	}
	if limiter.batchSize != 3 {
		t.Fatalf("expected batch size 3, got %d", limiter.batchSize)
	}
}
