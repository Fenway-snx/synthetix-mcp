package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type fakeLister struct {
	owned     []string
	delegated []string
	err       error
	calls     int
}

func (f *fakeLister) GetSubAccountIdsWithDelegations(
	_ context.Context,
	_ string,
) (*subAccountIdsWithDelegations, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &subAccountIdsWithDelegations{
		SubAccountIDs:          f.owned,
		DelegatedSubAccountIDs: f.delegated,
	}, nil
}

func TestRESTVerifier_OwnerMatchPrimesCacheAsOwner(t *testing.T) {
	cache := snx_lib_auth.NewAuthCache(10)
	lister := &fakeLister{owned: []string{"42"}}
	v := newRESTOwnershipVerifier(lister, cache)

	wallet := "0xAAaa0000000000000000000000000000000000aA"
	if err := v.VerifyOwnership(context.Background(), wallet, 42); err != nil {
		t.Fatalf("verify: %v", err)
	}

	got, found := cache.Lookup(snx_lib_api_types.WalletAddress(wallet), snx_lib_core.SubAccountId(42))
	if !found {
		t.Fatal("expected cache hit after successful verify")
	}
	if got != snx_lib_auth.AuthTypeOwner {
		t.Fatalf("expected OWNER, got %v", got)
	}
}

func TestRESTVerifier_DelegateMatchPrimesCacheAsDelegate(t *testing.T) {
	cache := snx_lib_auth.NewAuthCache(10)
	lister := &fakeLister{delegated: []string{"7"}}
	v := newRESTOwnershipVerifier(lister, cache)

	if err := v.VerifyOwnership(context.Background(), "0xbbbb", 7); err != nil {
		t.Fatalf("verify: %v", err)
	}
	got, found := cache.Lookup(snx_lib_api_types.WalletAddress("0xbbbb"), snx_lib_core.SubAccountId(7))
	if !found {
		t.Fatal("expected cache hit")
	}
	if got != snx_lib_auth.AuthTypeDelegate {
		t.Fatalf("expected DELEGATE, got %v", got)
	}
}

func TestRESTVerifier_NoMatchReturnsErrorAndStoresRefusal(t *testing.T) {
	cache := snx_lib_auth.NewAuthCache(10)
	lister := &fakeLister{owned: []string{"99"}, delegated: []string{"5"}}
	v := newRESTOwnershipVerifier(lister, cache)

	wallet := "0xabcd"
	err := v.VerifyOwnership(context.Background(), wallet, 42)
	if err == nil {
		t.Fatal("expected authorization error")
	}
	if !cache.LookupRefusal(snx_lib_api_types.WalletAddress(wallet), snx_lib_core.SubAccountId(42)) {
		t.Fatal("expected refusal to be negatively cached")
	}
}

func TestRESTVerifier_ListerErrorSkipsNegativeCache(t *testing.T) {
	cache := snx_lib_auth.NewAuthCache(10)
	lister := &fakeLister{err: errors.New("upstream exploded")}
	v := newRESTOwnershipVerifier(lister, cache)

	wallet := "0xabcd"
	err := v.VerifyOwnership(context.Background(), wallet, 42)
	if err == nil {
		t.Fatal("expected upstream error to propagate")
	}
	if cache.LookupRefusal(snx_lib_api_types.WalletAddress(wallet), snx_lib_core.SubAccountId(42)) {
		t.Fatal("transient upstream errors must not populate the refusal cache")
	}
}

func TestRESTVerifier_CacheHitAvoidsListerCall(t *testing.T) {
	cache := snx_lib_auth.NewAuthCache(10)
	wallet := "0xabcd"
	cache.Store(snx_lib_api_types.WalletAddress(wallet), snx_lib_core.SubAccountId(42),
		snx_lib_auth.AuthTypeOwner)

	lister := &fakeLister{owned: []string{"42"}}
	v := newRESTOwnershipVerifier(lister, cache)
	if err := v.VerifyOwnership(context.Background(), wallet, 42); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if lister.calls != 0 {
		t.Fatalf("expected lister to be skipped on cache hit, got %d calls", lister.calls)
	}
}

func TestRESTVerifier_RejectsInvalidArguments(t *testing.T) {
	cache := snx_lib_auth.NewAuthCache(10)
	v := newRESTOwnershipVerifier(&fakeLister{}, cache)

	if err := v.VerifyOwnership(context.Background(), "", 1); err == nil {
		t.Fatal("expected error for empty wallet")
	}
	if err := v.VerifyOwnership(context.Background(), "0xabc", 0); err == nil {
		t.Fatal("expected error for non-positive subaccount")
	}
}

// End-to-end: a Manager built in REST mode must successfully
// authenticate a signed message by priming the cache via the REST
// verifier and then letting lib/auth see the cache hit.
func TestManager_RESTMode_AuthenticateUsesRESTVerifier(t *testing.T) {
	store := &memorySessionStore{}
	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(42)

	cache := snx_lib_auth.NewAuthCache(10)
	// Critically: pass a nil subaccount client. In REST mode the
	// shared lib/auth.Authenticator must never need it; the REST
	// verifier warms the cache before ValidateAccountAuth runs.
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			nil,
			cache,
			snx_lib_auth.DefaultDomainName,
			"1",
			1,
		),
	)
	lister := &fakeLister{owned: []string{"42"}}
	verifier := newRESTOwnershipVerifier(lister, cache)

	manager := newManager(
		snx_lib_logging_doubles.NewStubLogger(),
		store,
		30*time.Minute,
		authenticator,
		cache,
		verifier,
	)
	t.Cleanup(func() { _ = manager.Close() })

	message, signature, err := wallet.GenerateAuthMessage(subAccountID)
	if err != nil {
		t.Fatalf("generate auth message: %v", err)
	}
	result, err := manager.Authenticate(context.Background(), "session-rest", message, signature)
	if err != nil {
		t.Fatalf("authenticate (REST mode): %v", err)
	}
	if !result.Authenticated {
		t.Fatal("expected authenticated")
	}
	if result.SubAccountID != int64(subAccountID) {
		t.Fatalf("expected subaccount %d, got %d", subAccountID, result.SubAccountID)
	}
	if lister.calls != 1 {
		t.Fatalf("expected exactly one REST lister call, got %d", lister.calls)
	}

	saved, err := store.Get(context.Background(), "session-rest")
	if err != nil {
		t.Fatalf("get saved: %v", err)
	}
	if saved.AuthMode != session.AuthModeAuthenticated {
		t.Fatalf("expected authenticated session, got %v", saved.AuthMode)
	}
}

// Regression: a REST-mode manager must reject authentication when
// the signed subaccount is not owned or delegated to the recovered
// wallet, without ever calling into lib/auth's nil verifier.
func TestManager_RESTMode_AuthenticateRejectsUnauthorizedSubaccount(t *testing.T) {
	store := &memorySessionStore{}
	wallet := snx_lib_auth.NewTestWalletWithSeed(0)
	subAccountID := snx_lib_core.SubAccountId(42)

	cache := snx_lib_auth.NewAuthCache(10)
	authenticator := snx_lib_auth.NewAccountAuthenticator(
		snx_lib_auth.NewAuthenticator(
			snx_lib_auth.NewTestNonceStore(),
			nil,
			cache,
			snx_lib_auth.DefaultDomainName,
			"1",
			1,
		),
	)
	// Lister returns a different subaccount, so the recovered wallet
	// has no claim on 42.
	lister := &fakeLister{owned: []string{"99"}}
	verifier := newRESTOwnershipVerifier(lister, cache)

	manager := newManager(
		snx_lib_logging_doubles.NewStubLogger(),
		store,
		30*time.Minute,
		authenticator,
		cache,
		verifier,
	)
	t.Cleanup(func() { _ = manager.Close() })

	message, signature, err := wallet.GenerateAuthMessage(subAccountID)
	if err != nil {
		t.Fatalf("generate auth message: %v", err)
	}
	if _, err := manager.Authenticate(context.Background(), "session-bad", message, signature); err == nil {
		t.Fatal("expected authentication to fail for unauthorized subaccount")
	}
}
