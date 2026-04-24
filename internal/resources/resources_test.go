package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_api_ratelimiting "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/rate_limiting"
	"github.com/Fenway-snx/synthetix-mcp/internal/server/backend"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
)

func TestRegisterResourcesListAndRead(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &memorySessionStore{
		sessions: map[string]*session.State{},
	}
	limiter := &fakeReadRateLimiter{}
	Register(server, testToolDeps(testConfig(), &backend.Clients{}, store, nil, limiter), nil)

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

	list, err := cs.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("list resources failed: %v", err)
	}
	if len(list.Resources) < 6 {
		t.Fatalf("expected public resources to be registered, got %d", len(list.Resources))
	}

	read, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: serverInfoURI})
	if err != nil {
		t.Fatalf("read server info failed: %v", err)
	}
	if len(read.Contents) != 1 || !strings.Contains(read.Contents[0].Text, `"name": "synthetix-mcp"`) {
		t.Fatalf("unexpected server info contents: %#v", read.Contents)
	}
	if limiter.operationName != "read_server_info" {
		t.Fatalf("expected read_server_info limiter operation, got %q", limiter.operationName)
	}
	if limiter.batchSize != 1 {
		t.Fatalf("expected resource reads to consume batch size 1, got %d", limiter.batchSize)
	}

	store.sessions[cs.ID()] = &session.State{
		AuthMode:      session.AuthModeAuthenticated,
		SubAccountID:  77,
		WalletAddress: "0xabc",
	}
	risk, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: accountRiskLimitsURI})
	if err != nil {
		t.Fatalf("read risk limits failed: %v", err)
	}
	if len(risk.Contents) != 1 || !strings.Contains(risk.Contents[0].Text, `"subAccountId": "77"`) {
		t.Fatalf("unexpected risk limits contents: %#v", risk.Contents)
	}
	if limiter.operationName != "read_risk_limits" {
		t.Fatalf("expected read_risk_limits limiter operation, got %q", limiter.operationName)
	}
	if limiter.state == nil || limiter.state.SubAccountID != 77 {
		t.Fatalf("expected authenticated state to be passed to limiter, got %#v", limiter.state)
	}
}

func TestAgentGuideContentsIncludesEmbeddedMarkdown(t *testing.T) {
	t.Parallel()

	body := agentGuideContents(testConfig())
	if !strings.Contains(body, "Synthetix MCP Agent Guide") {
		t.Fatalf("expected agent guide to include title, got %s", body)
	}
	if !strings.Contains(body, "Current server:") {
		t.Fatalf("expected agent guide to include server footer, got %s", body)
	}
}

func TestAccountRiskLimitsUsesEffectiveSpecificOverride(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &memorySessionStore{
		sessions: map[string]*session.State{},
	}
	cfg := testConfig()
	cfg.OrderRateLimiterConfig = snx_lib_api_ratelimiting.PerSubAccountRateLimiterConfig{
		WindowMs:         1000,
		GeneralRateLimit: 20,
		SpecificRateLimits: snx_lib_api_ratelimiting.PerSubAccountRateLimits{
			77: 50,
		},
	}
	Register(server, testToolDeps(cfg, &backend.Clients{}, store, nil, &fakeReadRateLimiter{}), nil)

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

	store.sessions[cs.ID()] = &session.State{
		AuthMode:      session.AuthModeAuthenticated,
		SubAccountID:  77,
		WalletAddress: "0xabc",
	}

	read, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: accountRiskLimitsURI})
	if err != nil {
		t.Fatalf("read risk limits failed: %v", err)
	}
	if len(read.Contents) != 1 {
		t.Fatalf("unexpected risk limit payload: %#v", read.Contents)
	}
	if !strings.Contains(read.Contents[0].Text, `"tokensPerWindow": 50`) {
		t.Fatalf("expected effective override in payload, got %s", read.Contents[0].Text)
	}
}

type fakeReadRateLimiter struct {
	batchSize     int
	operationName string
	state         *session.State
}

func (f *fakeReadRateLimiter) Check(_ context.Context, operationName string, batchSize int, state *session.State) error {
	f.operationName = operationName
	f.batchSize = batchSize
	f.state = state
	return nil
}

type memorySessionStore struct {
	sessions map[string]*session.State
}

func (m *memorySessionStore) Delete(context.Context, string) error { return nil }

func (m *memorySessionStore) DeleteIfExists(_ context.Context, sessionID string) (bool, error) {
	_, ok := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	return ok, nil
}

func (m *memorySessionStore) Get(_ context.Context, sessionID string) (*session.State, error) {
	state, ok := m.sessions[sessionID]
	if !ok {
		return nil, session.ErrSessionNotFound
	}
	copy := *state
	return &copy, nil
}

func (m *memorySessionStore) Save(_ context.Context, sessionID string, state *session.State, _ time.Duration) error {
	copy := *state
	m.sessions[sessionID] = &copy
	return nil
}

func (m *memorySessionStore) Touch(_ context.Context, _ string, _ time.Duration) error { return nil }

func testToolDeps(cfg *config.Config, clients *backend.Clients, store session.Store, verifier tools.SessionAccessVerifier, limiter tools.ToolRateLimiter) *tools.ToolDeps {
	return &tools.ToolDeps{
		Cfg:      cfg,
		Clients:  clients,
		Limiter:  limiter,
		Store:    store,
		Verifier: verifier,
	}
}

func testConfig() *config.Config {
	return &config.Config{
		AuthRPSPerSubAccount:       20,
		Environment:                "development",
		MaxSubscriptionsPerSession: 10,
		PublicRPSPerIP:             10,
		ServerName:                 "synthetix-mcp",
		ServerVersion:              "0.1.0",
		SessionTTL:                 30 * time.Minute,
	}
}
