package resources

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type explicitResourceStore struct {
	state *session.State
	err   error
}

func (s *explicitResourceStore) Delete(context.Context, string) error {
	return nil
}

func (s *explicitResourceStore) DeleteIfExists(context.Context, string) (bool, error) {
	return true, nil
}

func (s *explicitResourceStore) Get(context.Context, string) (*session.State, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.state, nil
}

func (s *explicitResourceStore) Save(context.Context, string, *session.State, time.Duration) error {
	return nil
}

func (s *explicitResourceStore) Touch(context.Context, string, time.Duration) error {
	return nil
}

func TestTextResourceResultPreservesMetadata(t *testing.T) {
	result := textResourceResult("system://server-info", "application/json", `{"ok":true}`)

	if len(result.Contents) != 1 {
		t.Fatalf("expected one resource content entry, got %d", len(result.Contents))
	}
	if result.Contents[0].URI != "system://server-info" {
		t.Fatalf("expected URI system://server-info, got %q", result.Contents[0].URI)
	}
	if result.Contents[0].MIMEType != "application/json" {
		t.Fatalf("expected JSON mime type, got %q", result.Contents[0].MIMEType)
	}
	if result.Contents[0].Text != `{"ok":true}` {
		t.Fatalf("unexpected resource body %q", result.Contents[0].Text)
	}
}

func TestSessionStateForReadReturnsNilWhenStoreMissing(t *testing.T) {
	state, err := sessionStateForRead(context.Background(), nil, nil, nil)
	if err != nil {
		t.Fatalf("expected nil store to be ignored, got %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil store to return nil state, got %#v", state)
	}
}

func TestSessionStateForIDSuppressesMissingSessionError(t *testing.T) {
	store := &explicitResourceStore{err: session.ErrSessionNotFound}

	state, err := sessionStateForID(context.Background(), store, "test-session-id", nil)
	if err != nil {
		t.Fatalf("expected missing session to be suppressed, got %v", err)
	}
	if state != nil {
		t.Fatalf("expected missing session to return nil state, got %#v", state)
	}
}

type revokeSessionTestStore struct {
	state     *session.State
	deleteErr error
}

func (s *revokeSessionTestStore) Get(context.Context, string) (*session.State, error) {
	return s.state, nil
}

func (s *revokeSessionTestStore) Delete(context.Context, string) error {
	return s.deleteErr
}

func (s *revokeSessionTestStore) DeleteIfExists(context.Context, string) (bool, error) {
	if s.deleteErr != nil {
		return false, s.deleteErr
	}
	existed := s.state != nil
	s.state = nil
	return existed, nil
}

func (s *revokeSessionTestStore) Save(context.Context, string, *session.State, time.Duration) error {
	return nil
}

func (s *revokeSessionTestStore) Touch(context.Context, string, time.Duration) error {
	return nil
}

func TestSessionStateForIDReturnsErrorWhenRevokeFails(t *testing.T) {
	verifyErr := errors.New("access revoked")
	store := &revokeSessionTestStore{
		state: &session.State{
			AuthMode:       session.AuthModeAuthenticated,
			SubAccountID:   1,
			WalletAddress:  "0xabc",
		},
		deleteErr: errors.New("redis delete failed"),
	}
	verifier := stubVerifier{err: verifyErr}

	_, err := sessionStateForID(context.Background(), store, "sid", verifier)
	if err == nil {
		t.Fatal("expected error when verify fails and session delete fails")
	}
	if !errors.Is(err, verifyErr) {
		t.Fatalf("expected verify error in chain, got %v", err)
	}
	if !errors.Is(err, store.deleteErr) {
		t.Fatalf("expected delete error in chain, got %v", err)
	}
}

type stubVerifier struct {
	err error
}

func (s stubVerifier) VerifySessionAccess(context.Context, string, int64) error {
	return s.err
}

func TestSessionMetadataHelpersReturnPublicDefaults(t *testing.T) {
	if authMode(nil) != string(session.AuthModePublic) {
		t.Fatalf("expected nil auth mode to default to %q, got %q", session.AuthModePublic, authMode(nil))
	}
	if subaccountID(nil) != "" {
		t.Fatalf("expected nil subaccount ID to be empty, got %q", subaccountID(nil))
	}
	if walletAddress(nil) != "" {
		t.Fatalf("expected nil wallet address to be empty, got %q", walletAddress(nil))
	}
}

func TestAgentGuideAndRunbookContentsIncludeKeyInstructions(t *testing.T) {
	cfg := &config.Config{
		Environment:   "test",
		ServerName:    "synthetix-mcp",
		ServerVersion: "0.1.0",
	}

	guide := agentGuideContents(cfg)
	if !strings.Contains(guide, "get_context") {
		t.Fatalf("expected guide to mention get_context, got %s", guide)
	}
	if !strings.Contains(guide, "synthetix-mcp 0.1.0 in test") {
		t.Fatalf("expected guide to embed current server identity, got %s", guide)
	}

	runbook := runbookContents()
	if !strings.Contains(runbook, "## Session bootstrap") {
		t.Fatalf("expected runbook session bootstrap section, got %s", runbook)
	}
	if !strings.Contains(runbook, "restore_session") {
		t.Fatalf("expected runbook restore_session guidance, got %s", runbook)
	}
}
