package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
	Register(server, testToolDeps(testConfig(), &backend.Clients{}, store, nil), nil)

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
	routing, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: routingGuideURI})
	if err != nil {
		t.Fatalf("read routing guide failed: %v", err)
	}
	for _, want := range []string{`"brokerEnabled": false`, `"mode": "external_wallet"`, `"signed_place_order"`} {
		if len(routing.Contents) != 1 || !strings.Contains(routing.Contents[0].Text, want) {
			t.Fatalf("expected routing guide to contain %q, got %#v", want, routing.Contents)
		}
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
}

func TestRoutingGuideResourcePrefersBrokerToolsWhenBrokerEnabled(t *testing.T) {
	cfg := testConfig()
	cfg.AgentBroker.Enabled = true

	body := routingGuideResource(cfg.AgentBroker.Enabled)
	if body["brokerEnabled"] != true || body["mode"] != "broker" {
		t.Fatalf("unexpected broker routing guide: %#v", body)
	}
	doNotUse, ok := body["doNotUse"].([]string)
	if !ok {
		t.Fatalf("expected doNotUse list, got %#v", body["doNotUse"])
	}
	foundSigned := false
	for _, item := range doNotUse {
		if item == "signed_*" {
			foundSigned = true
		}
	}
	if !foundSigned {
		t.Fatalf("expected broker routing guide to reject signed_* tools, got %#v", doNotUse)
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

func TestAccountRiskLimitsIncludesRateLimitGuidance(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &memorySessionStore{
		sessions: map[string]*session.State{},
	}
	cfg := testConfig()
	Register(server, testToolDeps(cfg, &backend.Clients{}, store, nil), nil)

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
	if !strings.Contains(read.Contents[0].Text, `"rateLimitGuidance"`) {
		t.Fatalf("expected rate limit guidance in payload, got %s", read.Contents[0].Text)
	}
	if !strings.Contains(read.Contents[0].Text, "this MCP does not enforce local request quotas") {
		t.Fatalf("expected non-enforcement guidance in payload, got %s", read.Contents[0].Text)
	}
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

func testToolDeps(cfg *config.Config, clients *backend.Clients, store session.Store, verifier tools.SessionAccessVerifier) *tools.ToolDeps {
	return &tools.ToolDeps{
		Cfg:      cfg,
		Clients:  clients,
		Store:    store,
		Verifier: verifier,
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Environment:                "development",
		MaxSubscriptionsPerSession: 10,
		ServerName:                 "synthetix-mcp",
		ServerVersion:              "0.1.0",
		SessionTTL:                 30 * time.Minute,
	}
}
