package auth

import (
	"context"
	"encoding/hex"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_authgrpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authgrpc"
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/metrics"
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

func TestAuthenticateStoresSession(t *testing.T) {
	store := &memorySessionStore{}
	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(42)

	mockSubaccountClient := snx_lib_authtest.NewMockSubaccountServiceClient()
	mockSubaccountClient.AddMockAccount(snx_lib_api_types.WalletAddress(wallet.GetAddress()), subAccountID)

	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			snx_lib_authgrpc.NewVerifier(mockSubaccountClient),
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

	mockSubaccountClient := snx_lib_authtest.NewMockSubaccountServiceClient()
	mockSubaccountClient.AddMockAccount(snx_lib_api_types.WalletAddress(wallet.GetAddress()), subAccountID)

	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			snx_lib_authgrpc.NewVerifier(mockSubaccountClient),
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

// Satisfies both session.Store and the internal sessionCounter interface
// so newManager starts the reconciler against it. The atomic counter
// drives the simulated current session count from the test.
type countingMemorySessionStore struct {
	memorySessionStore
	count atomic.Int64
}

func (c *countingMemorySessionStore) Count(context.Context) (int, error) {
	return int(c.count.Load()), nil
}

func TestSessionMetricsReconcilerSetsGaugeFromStoreCount(t *testing.T) {
	// The reconciler runs once before its first tick, so the gauge
	// reflects the store count without needing a sleep or fake clock.
	store := &countingMemorySessionStore{}
	store.count.Store(7)

	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(1)
	mockSubaccountClient := snx_lib_authtest.NewMockSubaccountServiceClient()
	mockSubaccountClient.AddMockAccount(snx_lib_api_types.WalletAddress(wallet.GetAddress()), subAccountID)
	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			snx_lib_authgrpc.NewVerifier(mockSubaccountClient),
			cache,
			snx_lib_auth.DefaultDomainName,
			"1",
			1,
		),
	)

	// Seed the gauge with a wrong value so a reset proves reconciliation
	// rather than coincidentally matching the global default of 0.
	metrics.ActiveSessions().Set(999)

	manager := newManager(
		snx_lib_logging_doubles.NewStubLogger(),
		store,
		30*time.Minute,
		authenticator,
		cache,
		nil,
	)
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Errorf("manager.Close: %v", err)
		}
	})

	// The first reconcile fires immediately on goroutine start; wait
	// briefly for the in-memory atomic load and gauge Set to land.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if got := testutil.ToFloat64(metrics.ActiveSessions()); got == 7 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected mcp_active_sessions to be reconciled to 7, got %v", testutil.ToFloat64(metrics.ActiveSessions()))
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Inc must fire only when Authenticate creates a brand-new Redis key.
// Re-authenticating an existing key (whether its prior authMode was
// authenticated or anything else) must not Inc, because the reconciler
// already counts that key. Otherwise the gauge double-counts until the
// next reconcile.
func TestAuthenticateGaugeIncrementMatchesRedisKeyCreation(t *testing.T) {
	store := &memorySessionStore{}
	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(42)

	mockSubaccountClient := snx_lib_authtest.NewMockSubaccountServiceClient()
	mockSubaccountClient.AddMockAccount(snx_lib_api_types.WalletAddress(wallet.GetAddress()), subAccountID)

	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			snx_lib_authgrpc.NewVerifier(mockSubaccountClient),
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
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Errorf("manager.Close: %v", err)
		}
	})

	metrics.ActiveSessions().Set(0)

	authenticate := func(t *testing.T) {
		t.Helper()
		message, signature, err := wallet.GenerateAuthMessage(subAccountID)
		if err != nil {
			t.Fatalf("generate auth message: %v", err)
		}
		if _, err := manager.Authenticate(context.Background(), "session-1", message, signature); err != nil {
			t.Fatalf("authenticate: %v", err)
		}
	}

	authenticate(t)
	if got := testutil.ToFloat64(metrics.ActiveSessions()); got != 1 {
		t.Fatalf("expected gauge=1 after first authenticate, got %v", got)
	}

	authenticate(t)
	if got := testutil.ToFloat64(metrics.ActiveSessions()); got != 1 {
		t.Fatalf("expected gauge=1 after re-authenticate of existing session, got %v", got)
	}

	// Simulate a persisted state whose authMode is not authenticated:
	// older code would Inc here because the prior auth-mode test
	// failed, double-counting a key the reconciler already saw.
	store.state.AuthMode = session.AuthModePublic
	authenticate(t)
	if got := testutil.ToFloat64(metrics.ActiveSessions()); got != 1 {
		t.Fatalf("expected gauge=1 when prior key existed with non-authenticated authMode, got %v", got)
	}
}

func TestSessionMetricsReconcilerNoOpForStoreWithoutCount(t *testing.T) {
	// memorySessionStore omits Count, so the reconciler must not start.
	// Close would hang on the unpopulated done channel if it had, so a
	// clean Close confirms the no-op path.
	store := &memorySessionStore{}
	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(1)
	mockSubaccountClient := snx_lib_authtest.NewMockSubaccountServiceClient()
	mockSubaccountClient.AddMockAccount(snx_lib_api_types.WalletAddress(wallet.GetAddress()), subAccountID)
	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			snx_lib_authgrpc.NewVerifier(mockSubaccountClient),
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
	if err := manager.Close(); err != nil {
		t.Fatalf("Close on store-without-Count returned error: %v", err)
	}
}
