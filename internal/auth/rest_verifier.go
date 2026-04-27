package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// OwnershipVerifier decides whether an EOA (wallet) may act on a
// given subaccount. Implementations populate the shared AuthCache on
// success so that downstream lib/auth calls hit the cache rather than
// re-checking.
type OwnershipVerifier interface {
	// VerifyOwnership returns nil if wallet owns or has an active
	// delegation on subAccountID. Success MUST populate the auth cache
	// so lib/auth.VerifyAccountOwnership (called transitively during
	// signature validation) short-circuits.
	VerifyOwnership(ctx context.Context, wallet string, subAccountID int64) error
}

// SubAccountIdsLister is the narrow surface of restinfo.Client that
// the REST ownership verifier depends on. Pulled out so tests can
// substitute a fake without spinning up a real HTTP client.
type SubAccountIdsLister interface {
	GetSubAccountIdsWithDelegations(ctx context.Context, walletAddress string) (*subAccountIdsWithDelegations, error)
}

// subAccountIdsWithDelegations mirrors the restinfo response used by
// the verifier. Defined locally (rather than importing the restinfo
// types package) so the unit tests do not pull in restinfo's full
// dependency graph.
type subAccountIdsWithDelegations struct {
	SubAccountIDs          []string
	DelegatedSubAccountIDs []string
}

// restInfoAdapter adapts a real *restinfo.Client to the narrow
// SubAccountIdsLister interface above. We cannot import the restinfo
// package from this file without a cycle (restinfo imports no auth
// symbols, but keeping the adapter here keeps the interface honest
// and the mock trivial to write).
type restInfoAdapter struct {
	fetch func(ctx context.Context, wallet string) (owned []string, delegated []string, err error)
}

func (a restInfoAdapter) GetSubAccountIdsWithDelegations(
	ctx context.Context,
	walletAddress string,
) (*subAccountIdsWithDelegations, error) {
	owned, delegated, err := a.fetch(ctx, walletAddress)
	if err != nil {
		return nil, err
	}
	return &subAccountIdsWithDelegations{
		SubAccountIDs:          owned,
		DelegatedSubAccountIDs: delegated,
	}, nil
}

// restOwnershipVerifier resolves (wallet, subAccountID) ownership via
// /v1/info getSubAccountIds and pre-populates the shared AuthCache so
// subsequent lib/auth.VerifyAccountOwnership calls short-circuit on
// the cache hit.
//
// Concurrency: safe for concurrent use. The AuthCache is the single
// point of coordination; the HTTP lister is expected to be
// goroutine-safe (the real *restinfo.Client is).
//
// Correctness: this verifier is trust-on-first-use from the wallet's
// perspective. The recovered EOA is authoritative (signatures are
// self-authenticating) and the subaccount owner/delegate list is
// authoritative from the upstream REST API. No other state is
// trusted.
type restOwnershipVerifier struct {
	lister SubAccountIdsLister
	cache  *snx_lib_auth.AuthCache
}

func newRESTOwnershipVerifier(
	lister SubAccountIdsLister,
	cache *snx_lib_auth.AuthCache,
) *restOwnershipVerifier {
	return &restOwnershipVerifier{lister: lister, cache: cache}
}

// VerifyOwnership hits the REST lister; on success it primes the
// shared AuthCache with the correct AuthorizationType so downstream
// lib/auth calls short-circuit on the cache hit.
//
// Returns a wrapped error on definitive refusal (wallet is not an
// owner and not a delegate) and a service error when the REST call
// itself fails. Callers treat the former as auth failure and the
// latter as transient (do not negatively cache).
func (v *restOwnershipVerifier) VerifyOwnership(
	ctx context.Context,
	wallet string,
	subAccountID int64,
) error {
	if v == nil {
		return errors.New("rest ownership verifier is nil")
	}
	if wallet == "" {
		return errors.New("wallet address is required")
	}
	if subAccountID <= 0 {
		return errors.New("subaccount id must be positive")
	}

	// Cache fast path: if the shared AuthCache already has a positive
	// entry or an active refusal tombstone, honor it without hitting
	// the REST API. This mirrors lib/auth.VerifyAccountOwnership's
	// ordering so a warm cache keeps standalone mode low-latency.
	if v.cache != nil {
		if _, found := v.cache.Lookup(
			snx_lib_api_types.WalletAddress(wallet),
			snx_lib_core.SubAccountId(subAccountID),
		); found {
			return nil
		}
		if v.cache.LookupRefusal(
			snx_lib_api_types.WalletAddress(wallet),
			snx_lib_core.SubAccountId(subAccountID),
		) {
			return fmt.Errorf("wallet %s is not authorized to act on subaccount %d",
				snx_lib_auth.ShortAddress(snx_lib_api_types.WalletAddress(wallet)),
				subAccountID)
		}
	}

	resp, err := v.lister.GetSubAccountIdsWithDelegations(ctx, wallet)
	if err != nil {
		// Service error: do not negatively cache, surface to caller.
		return fmt.Errorf("resolve subaccount authorization from REST: %w", err)
	}

	subAcctStr := strconv.FormatInt(subAccountID, 10)
	if slicesContains(resp.SubAccountIDs, subAcctStr) {
		if v.cache != nil {
			v.cache.Store(
				snx_lib_api_types.WalletAddress(wallet),
				snx_lib_core.SubAccountId(subAccountID),
				snx_lib_auth.AuthTypeOwner,
			)
		}
		return nil
	}
	if slicesContains(resp.DelegatedSubAccountIDs, subAcctStr) {
		if v.cache != nil {
			v.cache.Store(
				snx_lib_api_types.WalletAddress(wallet),
				snx_lib_core.SubAccountId(subAccountID),
				snx_lib_auth.AuthTypeDelegate,
			)
		}
		return nil
	}

	// Definitive refusal: prime the negative cache so a spam of
	// repeated auth attempts with the same (wallet, subacct) tuple is
	// bounded by the cache's refusal TTL rather than our REST QPS.
	if v.cache != nil {
		v.cache.StoreRefusal(
			snx_lib_api_types.WalletAddress(wallet),
			snx_lib_core.SubAccountId(subAccountID),
		)
	}
	return fmt.Errorf("wallet %s is not authorized to act on subaccount %d",
		snx_lib_auth.ShortAddress(snx_lib_api_types.WalletAddress(wallet)),
		subAccountID)
}

func slicesContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
