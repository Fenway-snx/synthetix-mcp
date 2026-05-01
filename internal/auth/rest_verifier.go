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

// Decides whether a wallet may act on a subaccount.
// Successful checks populate the shared authorization cache.
type OwnershipVerifier interface {
	// Return nil only after populating the cache for owner or delegate access.
	VerifyOwnership(ctx context.Context, wallet string, subAccountID int64) error
}

// Narrow REST client surface used by ownership verification.
type SubAccountIdsLister interface {
	GetSubAccountIdsWithDelegations(ctx context.Context, walletAddress string) (*subAccountIdsWithDelegations, error)
}

// Local mirror of the REST response used by the verifier.
// Avoids pulling the full REST client dependency graph into tests.
type subAccountIdsWithDelegations struct {
	SubAccountIDs          []string
	DelegatedSubAccountIDs []string
}

// Adapts the REST client to the narrow ownership lookup interface.
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

// Resolves wallet authorization through REST and primes the shared cache.
// The recovered signer and upstream owner/delegate list are the trusted inputs.
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

// Checks REST ownership/delegation and primes cache on success.
// Definitive refusals and transient service errors stay distinct.
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

	// Honor cache hits and refusal tombstones before hitting REST.
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
