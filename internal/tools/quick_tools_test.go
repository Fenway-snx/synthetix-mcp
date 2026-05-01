package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Fenway-snx/synthetix-mcp/internal/agentbroker"
	internal_auth "github.com/Fenway-snx/synthetix-mcp/internal/auth"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type fakeQuickDomainProvider struct{}

func (fakeQuickDomainProvider) DomainName() string    { return "Synthetix" }
func (fakeQuickDomainProvider) DomainVersion() string { return "1" }
func (fakeQuickDomainProvider) ChainID() int          { return 1 }

type fakeQuickSubaccountResolver struct{}

func (fakeQuickSubaccountResolver) GetSubAccountIdsWithDelegations(context.Context, string) (*agentbroker.ResolvedSubAccounts, error) {
	return &agentbroker.ResolvedSubAccounts{}, nil
}

type storingQuickAuthenticator struct {
	store *fakeSessionStore
}

func (a *storingQuickAuthenticator) Authenticate(ctx context.Context, sessionID, message, signatureHex string) (*internal_auth.AuthenticateResult, error) {
	state := &session.State{
		AuthMode:      session.AuthModeAuthenticated,
		SubAccountID:  42,
		WalletAddress: "0xabc",
	}
	if err := a.store.Save(ctx, sessionID, state, time.Minute); err != nil {
		return nil, err
	}
	return &internal_auth.AuthenticateResult{
		Authenticated:    true,
		SessionExpiresAt: time.Now().Add(time.Minute).Unix(),
		SubAccountID:     state.SubAccountID,
		WalletAddress:    state.WalletAddress,
	}, nil
}

func TestEnsureBrokerSessionStoresMaterializedDefaultGuardrails(t *testing.T) {
	broker, err := agentbroker.New(agentbroker.Options{
		DomainProvider:     fakeQuickDomainProvider{},
		SubaccountResolver: fakeQuickSubaccountResolver{},
		PrivateKeyHex:      strings.Repeat("1", 64),
		SubAccountID:       42,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	store := &fakeSessionStore{sessions: map[string]*session.State{}}
	deps := &ToolDeps{
		Cfg:   &config.Config{SessionTTL: time.Minute},
		Store: store,
	}

	got, err := ensureBrokerSession(context.Background(), deps, broker, &storingQuickAuthenticator{store: store}, "session-1", nil)
	if err != nil {
		t.Fatalf("ensure broker session: %v", err)
	}

	if got.AgentGuardrails == nil {
		t.Fatal("expected guardrails to be materialized on returned state")
	}
	if got.AgentGuardrails.Preset != "standard" {
		t.Fatalf("expected standard preset, got %q", got.AgentGuardrails.Preset)
	}
	if len(got.AgentGuardrails.AllowedSymbols) != 1 || got.AgentGuardrails.AllowedSymbols[0] != "*" {
		t.Fatalf("expected wildcard symbol default, got %#v", got.AgentGuardrails.AllowedSymbols)
	}
	if len(got.AgentGuardrails.AllowedOrderTypes) != 2 ||
		got.AgentGuardrails.AllowedOrderTypes[0] != "LIMIT" ||
		got.AgentGuardrails.AllowedOrderTypes[1] != "MARKET" {
		t.Fatalf("expected LIMIT/MARKET order type defaults, got %#v", got.AgentGuardrails.AllowedOrderTypes)
	}

	stored := store.sessions["session-1"]
	if stored == nil || stored.AgentGuardrails == nil {
		t.Fatalf("expected materialized guardrails to be saved, got %#v", stored)
	}
	if len(stored.AgentGuardrails.AllowedSymbols) != 1 || stored.AgentGuardrails.AllowedSymbols[0] != "*" {
		t.Fatalf("expected saved wildcard symbol default, got %#v", stored.AgentGuardrails.AllowedSymbols)
	}
}
