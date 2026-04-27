package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type fakeSessionStore struct {
	sessions map[string]*session.State
}

func (f *fakeSessionStore) Delete(ctx context.Context, sessionID string) error {
	delete(f.sessions, sessionID)
	return nil
}

func (f *fakeSessionStore) DeleteIfExists(ctx context.Context, sessionID string) (bool, error) {
	_, ok := f.sessions[sessionID]
	delete(f.sessions, sessionID)
	return ok, nil
}

func (f *fakeSessionStore) Get(ctx context.Context, sessionID string) (*session.State, error) {
	state, ok := f.sessions[sessionID]
	if !ok {
		return nil, session.ErrSessionNotFound
	}
	copy := *state
	return &copy, nil
}

func (f *fakeSessionStore) Save(ctx context.Context, sessionID string, state *session.State, ttl time.Duration) error {
	copy := *state
	f.sessions[sessionID] = &copy
	return nil
}

func (f *fakeSessionStore) Touch(ctx context.Context, sessionID string, ttl time.Duration) error {
	state, ok := f.sessions[sessionID]
	if !ok {
		return session.ErrSessionNotFound
	}
	state.LastActivityAt = 1
	return nil
}

type fakeSessionVerifier struct {
	err error
}

func (f *fakeSessionVerifier) VerifySessionAccess(context.Context, string, int64) error {
	return f.err
}

func TestRegisterAccountToolsRegistersExpectedTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	RegisterAccountTools(server, testAccountToolDeps(&fakeSessionStore{sessions: map[string]*session.State{}}, nil), nil)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL,
	}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	toolsResult, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}

	expected := map[string]bool{
		"get_account_summary":     false,
		"get_positions":           false,
		"get_open_orders":         false,
		"get_order_history":       false,
		"get_trade_history":       false,
		"get_funding_payments":    false,
		"get_performance_history": false,
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

func TestGetAccountSummaryRequiresAuthenticatedSession(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterAccountTools(server, testAccountToolDeps(store, nil), nil)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL,
	}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_account_summary"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected unauthenticated call to fail closed")
	}
}

func testAccountConfig() *config.Config {
	return &config.Config{
		MaxSubscriptionsPerSession: 10,
		SessionTTL:                 30 * time.Minute,
	}
}

func testAccountToolDeps(store session.Store, verifier SessionAccessVerifier) *ToolDeps {
	return &ToolDeps{
		Cfg:      testAccountConfig(),
		Clients:  nil,
		Store:    store,
		Verifier: verifier,
	}
}

// With the REST-trade backend disabled (tradeReads=nil), the handler
// must fail fast before touching the authenticated subaccount — covers
// the early-return path that used to be masked by upstream fixtures.
// fake returning an empty response.
func TestGetAccountSummaryFailsWhenRESTTradeUnavailable(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterAccountTools(server, testAccountToolDeps(store, nil), nil)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL,
	}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	store.sessions[cs.ID()] = &session.State{
		AuthMode:      session.AuthModeAuthenticated,
		SubAccountID:  77,
		WalletAddress: "0xabc",
	}

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_account_summary",
		Arguments: map[string]any{
			"subAccountId": "77",
		},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected REST-unavailable branch to fail closed")
	}
}

func TestGetAccountSummaryClearsStaleSessionWhenVerifierRejects(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterAccountTools(
		server,
		testAccountToolDeps(store, &fakeSessionVerifier{err: errors.New("wallet is no longer authorized")}),
		nil,
	)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL,
	}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	store.sessions[cs.ID()] = &session.State{
		AuthMode:      session.AuthModeAuthenticated,
		SubAccountID:  77,
		WalletAddress: "0xabc",
	}

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_account_summary"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected stale authenticated session to fail closed")
	}
	if _, ok := store.sessions[cs.ID()]; ok {
		t.Fatal("expected stale authenticated session to be cleared from the store")
	}
}
