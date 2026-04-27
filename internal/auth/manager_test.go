package auth

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type memorySessionStore struct {
	state *session.State
}

func (s *memorySessionStore) Delete(context.Context, string) error {
	s.state = nil
	return nil
}

func (s *memorySessionStore) DeleteIfExists(_ context.Context, _ string) (bool, error) {
	existed := s.state != nil
	s.state = nil
	return existed, nil
}

func (s *memorySessionStore) Get(context.Context, string) (*session.State, error) {
	if s.state == nil {
		return nil, session.ErrSessionNotFound
	}
	return s.state, nil
}

func (s *memorySessionStore) Save(_ context.Context, _ string, state *session.State, _ time.Duration) error {
	s.state = state
	return nil
}

func (s *memorySessionStore) Touch(context.Context, string, time.Duration) error {
	return nil
}

type testSubaccountVerifier struct {
	owners map[snx_lib_api_types.WalletAddress]map[snx_lib_core.SubAccountId]struct{}
}

func newTestSubaccountVerifier() *testSubaccountVerifier {
	return &testSubaccountVerifier{
		owners: make(map[snx_lib_api_types.WalletAddress]map[snx_lib_core.SubAccountId]struct{}),
	}
}

func (v *testSubaccountVerifier) AddOwner(addr snx_lib_api_types.WalletAddress, id snx_lib_core.SubAccountId) {
	if v.owners[addr] == nil {
		v.owners[addr] = make(map[snx_lib_core.SubAccountId]struct{})
	}
	v.owners[addr][id] = struct{}{}
}

func (v *testSubaccountVerifier) VerifySubaccountAuthorization(
	_ context.Context,
	req snx_lib_auth.VerifySubaccountAuthorizationRequest,
) (snx_lib_auth.VerifySubaccountAuthorizationResponse, error) {
	addr := snx_lib_api_types.WalletAddressFromStringUnvalidated(req.Address)
	id := snx_lib_core.SubAccountId(req.SubAccountID)
	_, ok := v.owners[addr][id]
	authType := snx_lib_auth.AuthTypeNone
	if ok {
		authType = snx_lib_auth.AuthTypeOwner
	}
	return snx_lib_auth.VerifySubaccountAuthorizationResponse{
		IsAuthorized:      ok,
		AuthorizationType: authType,
	}, nil
}

func TestAuthenticateStoresSession(t *testing.T) {
	store := &memorySessionStore{}
	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(42)

	verifier := newTestSubaccountVerifier()
	verifier.AddOwner(snx_lib_api_types.WalletAddress(wallet.GetAddress()), subAccountID)

	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
				verifier,
			cache,
			snx_lib_auth.DefaultDomainName,
			"1",
			1,
		),
	)

	manager := newManager(
		snx_lib_logging_doubles.NewStubLogger(),
		store,
		30*time.Minute,
		authenticator,
		cache,
		nil,
	)

	message, signature, err := wallet.GenerateAuthMessage(subAccountID)
	if err != nil {
		t.Fatalf("generate auth message: %v", err)
	}

	result, err := manager.Authenticate(context.Background(), "session-1", message, signature)
	if err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}

	if !result.Authenticated {
		t.Fatal("expected authenticated result")
	}
	if result.SubAccountID != int64(subAccountID) {
		t.Fatalf("expected subaccount %d, got %d", subAccountID, result.SubAccountID)
	}
	if result.WalletAddress != wallet.GetAddress() {
		t.Fatalf("expected wallet %s, got %s", wallet.GetAddress(), result.WalletAddress)
	}

	saved, err := store.Get(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get saved session: %v", err)
	}
	if saved.AuthMode != session.AuthModeAuthenticated {
		t.Fatalf("expected authenticated auth mode, got %s", saved.AuthMode)
	}
	if saved.SubAccountID != int64(subAccountID) {
		t.Fatalf("expected saved subaccount %d, got %d", subAccountID, saved.SubAccountID)
	}
	if saved.WalletAddress != wallet.GetAddress() {
		t.Fatalf("expected saved wallet %s, got %s", wallet.GetAddress(), saved.WalletAddress)
	}
}

func TestAuthenticateRejectsStaleAuthMessage(t *testing.T) {
	store := &memorySessionStore{}
	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(42)

	verifier := newTestSubaccountVerifier()
	verifier.AddOwner(snx_lib_api_types.WalletAddress(wallet.GetAddress()), subAccountID)

	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			verifier,
			cache,
			snx_lib_auth.DefaultDomainName,
			"1",
			1,
		),
	)

	manager := newManager(
		snx_lib_logging_doubles.NewStubLogger(),
		store,
		30*time.Minute,
		authenticator,
		cache,
		nil,
	)

	staleTimestamp := time.Now().UTC().Add(-2 * time.Minute).Unix()
	typedData := snx_lib_auth.CreateEIP712TypedData(subAccountID, staleTimestamp, snx_lib_auth.ActionWebSocketAuth, snx_lib_auth.DefaultDomainName, "1", 1)
	message, err := snx_lib_auth.SerializeTypedData(typedData)
	if err != nil {
		t.Fatalf("serialize typed data: %v", err)
	}
	signatureBytes, err := wallet.SignTypedData(typedData)
	if err != nil {
		t.Fatalf("sign typed data: %v", err)
	}
	if signatureBytes[64] < 27 {
		signatureBytes[64] += 27
	}

	_, err = manager.Authenticate(context.Background(), "session-1", message, "0x"+hex.EncodeToString(signatureBytes))
	if err == nil {
		t.Fatal("expected stale auth message to be rejected")
	}
}

