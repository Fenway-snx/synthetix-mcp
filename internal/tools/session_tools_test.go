package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	internal_auth "github.com/Fenway-snx/synthetix-mcp/internal/auth"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeSubscriptionReader struct {
	bySession map[string][]string
}

type fakeSessionAuthenticator struct{}

func (f *fakeSessionAuthenticator) Authenticate(context.Context, string, string, string) (*internal_auth.AuthenticateResult, error) {
	return &internal_auth.AuthenticateResult{
		Authenticated:    true,
		SessionExpiresAt: 123,
		SubAccountID:     77,
		WalletAddress:    "0xabc",
	}, nil
}

type fakePrivateSubscriptionResetter struct {
	sessionIDs []string
}

func (f *fakePrivateSubscriptionResetter) ClearPrivateSubscriptions(sessionID string) {
	f.sessionIDs = append(f.sessionIDs, sessionID)
}

func (f *fakeSubscriptionReader) ActiveChannels(sessionID string) []string {
	if f == nil {
		return []string{}
	}
	return append([]string{}, f.bySession[sessionID]...)
}

func TestGetSessionIncludesActiveSubscriptions(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	subscriptions := &fakeSubscriptionReader{bySession: map[string][]string{}}
	RegisterSessionStateTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: store,
	}, subscriptions)

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
		AuthMode:       session.AuthModeAuthenticated,
		SubAccountID:   77,
		WalletAddress:  "0xabc",
		CreatedAt:      10,
		LastActivityAt: 11,
		ExpiresAt:      12,
	}
	subscriptions.bySession[cs.ID()] = []string{"marketPrices", "accountEvents"}

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_session"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %#v", result.Content)
	}

	structured := result.StructuredContent.(map[string]any)
	active := structured["activeSubscriptions"].([]any)
	if len(active) != 2 {
		t.Fatalf("expected two active subscriptions, got %d", len(active))
	}
	if structured["authMode"].(string) != string(session.AuthModeAuthenticated) {
		t.Fatalf("expected authenticated mode, got %v", structured["authMode"])
	}
}

func TestAuthenticateClearsPrivateSubscriptions(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	resetter := &fakePrivateSubscriptionResetter{}
	RegisterSessionTools(server, &ToolDeps{
		Cfg:   &config.Config{SessionTTL: 30 * time.Minute},
		Store: &fakeSessionStore{sessions: map[string]*session.State{}},
	}, &fakeSessionAuthenticator{}, resetter)

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
		Name: "authenticate",
		Arguments: map[string]any{
			"message":      "{}",
			"signatureHex": "0x01",
		},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected authenticate success, got error: %#v", result.Content)
	}
	if len(resetter.sessionIDs) != 1 || resetter.sessionIDs[0] != cs.ID() {
		t.Fatalf("expected private subscriptions to be cleared for %q, got %#v", cs.ID(), resetter.sessionIDs)
	}
}

func TestRefreshSessionReturnsCurrentSessionState(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterSessionStateTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: store,
	}, nil)

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
		SubAccountID:  55,
		WalletAddress: "0xdef",
		ExpiresAt:     999,
	}

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "restore_session",
		Arguments: map[string]any{"sessionId": cs.ID()},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected restore_session to succeed, got error: %#v", result.Content)
	}

	structured := result.StructuredContent.(map[string]any)
	if !structured["restored"].(bool) {
		t.Fatal("expected restored=true")
	}
	if got, err := strconv.ParseInt(structured["subAccountId"].(string), 10, 64); err != nil || got != 55 {
		t.Fatalf("expected subAccountId 55, got %#v (err=%v)", structured["subAccountId"], err)
	}
}

func TestRestoreSessionReturnsErrorWithoutStoredSession(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterSessionStateTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: store,
	}, nil)

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
		Name:      "restore_session",
		Arguments: map[string]any{"sessionId": cs.ID()},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected restore_session to return error when no stored session exists")
	}
}

func TestGetSessionClearsStaleAuthenticatedStateWhenVerifierRejects(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterSessionStateTools(server, &ToolDeps{
		Cfg:      testAccountConfig(),
		Store:    store,
		Verifier: &fakeSessionVerifier{err: errors.New("wallet is no longer authorized")},
	}, nil)

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

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_session"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected get_session to degrade to public mode, got error: %#v", result.Content)
	}

	structured := result.StructuredContent.(map[string]any)
	if structured["authMode"].(string) != string(session.AuthModePublic) {
		t.Fatalf("expected public auth mode, got %v", structured["authMode"])
	}
	if _, ok := store.sessions[cs.ID()]; ok {
		t.Fatal("expected stale authenticated state to be deleted")
	}
}

func TestSetGuardrailsStoresResolvedSessionPolicy(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterSessionStateTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: store,
	}, nil)

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

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "set_guardrails",
		Arguments: map[string]any{
			"preset":              "standard",
			"allowedSymbols":      []string{"btc-usdt"},
			"allowedOrderTypes":   []string{"limit"},
			"maxOrderQuantity":    "2",
			"maxPositionQuantity": "3",
		},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected set_guardrails success, got error: %#v", result.Content)
	}

	structured := result.StructuredContent.(map[string]any)
	guardrailState := structured["agentGuardrails"].(map[string]any)
	if guardrailState["effectivePreset"].(string) != "standard" {
		t.Fatalf("expected standard effective preset, got %v", guardrailState["effectivePreset"])
	}
	if !guardrailState["writeEnabled"].(bool) {
		t.Fatal("expected writeEnabled=true")
	}

	stored := store.sessions[cs.ID()]
	if stored.AgentGuardrails == nil {
		t.Fatal("expected guardrails to be stored on session state")
	}
	if stored.AgentGuardrails.AllowedSymbols[0] != "btc-usdt" {
		t.Fatalf("expected original config to be stored, got %#v", stored.AgentGuardrails.AllowedSymbols)
	}
}

func TestSetGuardrailsUnknownPresetFallsBackToReadOnly(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	RegisterSessionStateTools(server, &ToolDeps{
		Cfg:   testAccountConfig(),
		Store: store,
	}, nil)

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

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "set_guardrails",
		Arguments: map[string]any{
			"preset": "custom-risky-mode",
		},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected unknown preset to fall back successfully, got error: %#v", result.Content)
	}

	structured := result.StructuredContent.(map[string]any)
	guardrailState := structured["agentGuardrails"].(map[string]any)
	if guardrailState["effectivePreset"].(string) != "read_only" {
		t.Fatalf("expected read_only effective preset, got %v", guardrailState["effectivePreset"])
	}
	if guardrailState["writeEnabled"].(bool) {
		t.Fatal("expected writeEnabled=false for unknown preset fallback")
	}
}
