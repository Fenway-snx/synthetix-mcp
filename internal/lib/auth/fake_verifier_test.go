package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// fakeSubaccountVerifier is a tiny in-package SubaccountVerifier suitable for
// authenticator_test.go. It avoids pulling lib/auth/authtest (and therefore
// google.golang.org/grpc) into lib/auth's own test build.
type fakeSubaccountVerifier struct {
	mu              sync.Mutex
	owners          map[snx_lib_api_types.WalletAddress]map[snx_lib_core.SubAccountId]struct{}
	delegations     map[snx_lib_api_types.WalletAddress]map[snx_lib_core.SubAccountId]time.Time // expiry; zero = no expiry
	failure         error
	VerifyCallCount int
}

func newFakeSubaccountVerifier() *fakeSubaccountVerifier {
	return &fakeSubaccountVerifier{
		owners:      make(map[snx_lib_api_types.WalletAddress]map[snx_lib_core.SubAccountId]struct{}),
		delegations: make(map[snx_lib_api_types.WalletAddress]map[snx_lib_core.SubAccountId]time.Time),
	}
}

// newFailingSubaccountVerifier returns a verifier whose
// VerifySubaccountAuthorization always returns a service error, mirroring the
// previous MockFailingSubaccountServiceClient behaviour.
func newFailingSubaccountVerifier() *fakeSubaccountVerifier {
	v := newFakeSubaccountVerifier()
	v.failure = errors.New("service unavailable")
	return v
}

func (f *fakeSubaccountVerifier) AddOwner(addr snx_lib_api_types.WalletAddress, id snx_lib_core.SubAccountId) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.owners[addr] == nil {
		f.owners[addr] = make(map[snx_lib_core.SubAccountId]struct{})
	}
	f.owners[addr][id] = struct{}{}
}

// expiresAt may be the zero time to indicate no expiration.
func (f *fakeSubaccountVerifier) AddDelegate(addr snx_lib_api_types.WalletAddress, id snx_lib_core.SubAccountId, expiresAt time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.delegations[addr] == nil {
		f.delegations[addr] = make(map[snx_lib_core.SubAccountId]time.Time)
	}
	f.delegations[addr][id] = expiresAt
}

func (f *fakeSubaccountVerifier) VerifySubaccountAuthorization(
	_ context.Context,
	req VerifySubaccountAuthorizationRequest,
) (VerifySubaccountAuthorizationResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.VerifyCallCount++

	if f.failure != nil {
		return VerifySubaccountAuthorizationResponse{}, f.failure
	}

	addr := snx_lib_api_types.WalletAddressFromStringUnvalidated(req.Address)
	id := snx_lib_core.SubAccountId(req.SubAccountID)

	if _, ok := f.owners[addr][id]; ok {
		return VerifySubaccountAuthorizationResponse{
			IsAuthorized:      true,
			AuthorizationType: AuthTypeOwner,
		}, nil
	}

	if exp, ok := f.delegations[addr][id]; ok {
		if !exp.IsZero() && exp.Before(time.Now()) {
			return VerifySubaccountAuthorizationResponse{
				IsAuthorized:      false,
				AuthorizationType: AuthTypeNone,
			}, nil
		}
		return VerifySubaccountAuthorizationResponse{
			IsAuthorized:      true,
			AuthorizationType: AuthTypeDelegate,
		}, nil
	}

	return VerifySubaccountAuthorizationResponse{
		IsAuthorized:      false,
		AuthorizationType: AuthTypeNone,
	}, nil
}
