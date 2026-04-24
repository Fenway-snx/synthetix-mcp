package auth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

var (
	errAddressDoesNotOwnSubaccount                           = errors.New("address does not own subaccount")
	errNonceAlreadyUsed                                      = errors.New("nonce already used")
	errNonceRequiredForThisOperation                         = errors.New("nonce is required for this operation")
	errOperationRequiresAccountOwnerDelegatesAreNotPermitted = errors.New("operation requires account owner, delegates are not permitted")
	errRequestExpired                                        = errors.New("request expired")
	errSignatureRequired                                     = errors.New("signature is required")
	errSubaccountClientNotConfigured                         = errors.New("subaccount client not configured")
	errTimestampAlreadyUsed                                  = errors.New("timestamp already used")
)

// Authenticator handles Ethereum signature-based authentication
type Authenticator struct {
	authCache          *AuthCache
	nonceStore         NonceStore
	subaccountVerifier SubaccountVerifier
	// EIP-712 domain configuration
	domainName    string
	domainVersion string
	chainID       int
}

// Creates a new Ethereum authenticator. Pass a non-nil authCache to enable
// in-memory caching of authorization results; pass nil to disable caching.
// Pass nil for subaccountVerifier when the cache is pre-primed by an external
// path (e.g. mcp-service primes via REST and never falls through to a
// verifier call).
func NewAuthenticator(nonceStore NonceStore, subaccountVerifier SubaccountVerifier, authCache *AuthCache, domainName, domainVersion string, chainID int) *Authenticator {
	return &Authenticator{
		authCache:          authCache,
		nonceStore:         nonceStore,
		subaccountVerifier: subaccountVerifier,
		domainName:         domainName,
		domainVersion:      domainVersion,
		chainID:            chainID,
	}
}

// ValidateAuthentication validates an Ethereum signature authentication request
func (a *Authenticator) ValidateAuthentication(req *AuthRequest) (*AuthResult, error) {
	// 1. Extract values from typed data message
	subAccountID, timestamp, err := ExtractAuthData(req.TypedData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract auth data: %w", err)
	}

	nonce := Nonce(timestamp)

	// 2. Validate timestamp (prevent old requests)
	if err := ValidateTimestamp(timestamp); err != nil {
		return nil, fmt.Errorf("timestamp validation failed: %w", err)
	}

	// 3. Verify the EIP-712 signature and recover address
	// Note: For backward compatibility, still use the signature verification as-is
	// The typed data domain should match our configuration, but verification handles the actual domain
	recoveredAddress, err := VerifyEIP712Signature(req.TypedData, req.Signature)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// 4. Atomically reserve the timestamp/nonce to prevent replay attacks
	// This must happen immediately after signature recovery to close the TOCTOU window
	reserved, err := a.nonceStore.ReserveNonce(recoveredAddress.Hex(), nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve timestamp: %w", err)
	}
	if !reserved {
		return nil, errTimestampAlreadyUsed
	}

	// 5. Verify account ownership
	walletAddress := snx_lib_api_types.WalletAddressFromStringUnvalidated(recoveredAddress.Hex())
	owns, err := a.VerifyAccountOwnership(walletAddress, subAccountID)
	if err != nil {
		return nil, fmt.Errorf("account ownership verification failed: %w", err)
	}
	if !owns {
		return nil, fmt.Errorf("address does not own the specified account: %d", subAccountID)
	}

	return &AuthResult{
		EthereumAddress: recoveredAddress.Hex(),
		SubAccountId:    subAccountID,
		Valid:           true,
	}, nil
}

// Verifies if an Ethereum address owns a specific account OR has delegated
// trading permission. Calls the SubaccountService to check account
// ownership using a single optimized call. Results are cached: positive
// results use the normal TTL, and definitive refusals are negatively cached
// with a short TTL to prevent DDoS via repeated unauthorized lookups.
// Service errors are never cached. For owner-only checks (e.g.,
// withdrawals), use VerifyAccountOwner instead.
func (a *Authenticator) VerifyAccountOwnership(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) (bool, error) {
	if a.authCache != nil {
		if _, found := a.authCache.Lookup(ethereumAddress, accountId); found {
			return true, nil
		}
		if a.authCache.LookupRefusal(ethereumAddress, accountId) {
			short := ShortAddress(ethereumAddress)
			return false, fmt.Errorf("wallet %s is not authorized to act on this subaccount", short)
		}
	}

	if a.subaccountVerifier == nil {
		return false, errSubaccountClientNotConfigured
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	req := VerifySubaccountAuthorizationRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountID: int64(accountId),
		Address:      snx_lib_api_types.WalletAddressToString(ethereumAddress),
		Permissions:  []string{DelegationPermissionSession.String()},
	}

	resp, err := a.subaccountVerifier.VerifySubaccountAuthorization(ctx, req)
	if err != nil {
		return false, fmt.Errorf("authorization verification failed due to service error: %w", err)
	}

	if resp.IsAuthorized {
		if a.authCache != nil {
			a.authCache.Store(ethereumAddress, accountId, resp.AuthorizationType)
		}
		return true, nil
	}

	if a.authCache != nil {
		a.authCache.StoreRefusal(ethereumAddress, accountId)
	}

	short := ShortAddress(ethereumAddress)
	return false, fmt.Errorf("wallet %s is not authorized to act on this subaccount", short)
}

// Verifies if an Ethereum address is the OWNER of a specific account (not a delegate).
// Used for sensitive operations like withdrawals that should only be allowed for account owners.
// Definitive refusals (wallet has no relationship at all) are negatively cached;
// delegate responses are not negatively cached since VerifyAccountOwnership
// may still accept them.
// For operations that allow delegated access, use VerifyAccountOwnership instead.
func (a *Authenticator) VerifyAccountOwner(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) (bool, error) {
	if a.authCache != nil {
		if authType, found := a.authCache.Lookup(ethereumAddress, accountId); found {
			if authType == AuthTypeOwner {
				return true, nil
			}
			short := ShortAddress(ethereumAddress)
			return false, fmt.Errorf("wallet %s is a delegate but not the owner of this subaccount", short)
		}
		if a.authCache.LookupRefusal(ethereumAddress, accountId) {
			short := ShortAddress(ethereumAddress)
			return false, fmt.Errorf("wallet %s is not the owner of this subaccount", short)
		}
	}

	if a.subaccountVerifier == nil {
		return false, errSubaccountClientNotConfigured
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	req := VerifySubaccountAuthorizationRequest{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountID: int64(accountId),
		Address:      snx_lib_api_types.WalletAddressToString(ethereumAddress),
		Permissions:  []string{DelegationPermissionSession.String()},
	}

	resp, err := a.subaccountVerifier.VerifySubaccountAuthorization(ctx, req)
	if err != nil {
		return false, fmt.Errorf("authorization verification failed due to service error: %w", err)
	}

	if resp.IsAuthorized && resp.AuthorizationType == AuthTypeOwner {
		if a.authCache != nil {
			a.authCache.Store(ethereumAddress, accountId, AuthTypeOwner)
		}
		return true, nil
	}

	short := ShortAddress(ethereumAddress)
	if resp.AuthorizationType == AuthTypeDelegate {
		return false, fmt.Errorf("wallet %s is a delegate but not the owner of this subaccount", short)
	}
	if a.authCache != nil {
		a.authCache.StoreRefusal(ethereumAddress, accountId)
	}
	return false, fmt.Errorf("wallet %s is not the owner of this subaccount", short)
}

// Removes a cached authorization entry for a specific wallet and subaccount.
// Called when a delegation is revoked via NATS event.
func (a *Authenticator) EvictAuth(
	walletAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
) {
	if a.authCache != nil {
		a.authCache.Evict(walletAddress, subAccountId)
	}
}

// AccountAuthenticator provides unified authentication for account-based operations
type AccountAuthenticator struct {
	*Authenticator
}

// NewAccountAuthenticator creates a new account authenticator
func NewAccountAuthenticator(base *Authenticator) *AccountAuthenticator {
	return &AccountAuthenticator{Authenticator: base}
}

// AuthOptions configures authentication behavior
type AuthOptions struct {
	// SupportExpiration enables expiration checking (default: false)
	SupportExpiration bool
	// UseTimestampNonce uses timestamp as nonce instead of separate nonce field (default: false)
	UseTimestampNonce bool
	// RequireOwner requires the signer to be the account owner, not a delegate (default: false)
	// Used for sensitive operations like withdrawals that should only be performed by owners
	RequireOwner bool
	// SkipNonceCheck skips nonce validation and reservation (default: false)
	// Used for read-only operations like get* actions that don't require replay protection
	SkipNonceCheck bool
}

// DefaultAuthOptions returns default authentication options
func DefaultAuthOptions() *AuthOptions {
	return &AuthOptions{
		SupportExpiration: false,
		UseTimestampNonce: false,
	}
}

// DataExtractor is a function that extracts authentication data from typed data
type DataExtractor func(apitypes.TypedData) (
	subAccountId snx_lib_core.SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	err error,
)

// ValidateAccountAuth performs unified account authentication with pluggable data extraction
func (a *AccountAuthenticator) ValidateAccountAuth(
	typedData apitypes.TypedData,
	signature string,
	extractFunc DataExtractor,
	opts *AuthOptions,
) (*AuthResult, error) {
	if opts == nil {
		opts = DefaultAuthOptions()
	}

	// 0. Reject empty signatures immediately
	if signature == "" {
		return nil, errSignatureRequired
	}

	// 1. Validate EIP-712 domain binding FIRST (before any other checks)
	// This prevents signature malleability and ensures the client is signing for the correct domain
	expectedDomain := GetEIP712Domain(a.domainName, a.domainVersion, a.chainID)

	// Check if the TypedData.Domain matches the server's expected domain
	if typedData.Domain.Name != expectedDomain.Name {
		return nil, fmt.Errorf("invalid domain name: expected %s, got '%s'", expectedDomain.Name, typedData.Domain.Name)
	}
	if typedData.Domain.Version != expectedDomain.Version {
		return nil, fmt.Errorf("invalid domain version: expected %s, got '%s'", expectedDomain.Version, typedData.Domain.Version)
	}
	if typedData.Domain.ChainId == nil || (*big.Int)(typedData.Domain.ChainId).String() != (*big.Int)(expectedDomain.ChainId).String() {
		expectedChainStr := "nil"
		if expectedDomain.ChainId != nil {
			expectedChainStr = (*big.Int)(expectedDomain.ChainId).String()
		}
		gotChainStr := "nil"
		if typedData.Domain.ChainId != nil {
			gotChainStr = (*big.Int)(typedData.Domain.ChainId).String()
		}
		return nil, fmt.Errorf("invalid chain ID: expected %s, got '%s'", expectedChainStr, gotChainStr)
	}
	// Note: VerifyingContract is always 0x0000... for off-chain usage, but we check it for completeness
	if typedData.Domain.VerifyingContract != expectedDomain.VerifyingContract {
		return nil, fmt.Errorf("invalid verifying contract: expected %s, got '%s'", expectedDomain.VerifyingContract, typedData.Domain.VerifyingContract)
	}

	// 2. Extract values from typed data (after domain validation)
	subAccountID, nonce, expiresAfter, err := extractFunc(typedData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract account data: %w", err)
	}

	// 4. Check expiration if supported and set
	if opts.SupportExpiration && expiresAfter > 0 && snx_lib_utils_time.Now().Unix() >= expiresAfter {
		return nil, errRequestExpired
	}

	// 5. Verify signature and recover address (domain already validated)
	recoveredAddress, err := VerifyEIP712Signature(typedData, signature)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// 6. Atomically reserve the nonce to prevent replay attacks
	// This must happen immediately after signature recovery to close the TOCTOU window
	if opts.SkipNonceCheck {
		// Skip nonce validation for read-only operations (e.g., get* actions)
	} else {
		// Nonce is required for state-changing operations
		if nonce == 0 {
			return nil, errNonceRequiredForThisOperation
		}
		reserved, err := a.nonceStore.ReserveNonce(recoveredAddress.Hex(), nonce)
		if err != nil {
			return nil, fmt.Errorf("failed to reserve nonce: %w", err)
		}
		if !reserved {
			return nil, errNonceAlreadyUsed
		}
	}

	// 7. Verify account ownership
	walletAddress := snx_lib_api_types.WalletAddressFromStringUnvalidated(recoveredAddress.Hex())
	if opts.RequireOwner {
		isOwner, err := a.VerifyAccountOwner(walletAddress, subAccountID)
		if err != nil {
			return nil, fmt.Errorf("account owner verification failed: %w", err)
		}
		if !isOwner {
			return nil, errOperationRequiresAccountOwnerDelegatesAreNotPermitted
		}
	} else {
		owns, err := a.VerifyAccountOwnership(walletAddress, subAccountID)
		if err != nil {
			return nil, fmt.Errorf("account ownership verification failed: %w", err)
		}
		if !owns {
			return nil, errAddressDoesNotOwnSubaccount
		}
	}

	return &AuthResult{
		EthereumAddress: recoveredAddress.Hex(),
		SubAccountId:    subAccountID,
		Valid:           true,
	}, nil
}

// Enabled returns whether authentication is enabled
func (a *AccountAuthenticator) Enabled() bool {
	return a != nil && a.Authenticator != nil
}

// DomainName returns the EIP-712 domain name
func (a *AccountAuthenticator) DomainName() string {
	if a == nil || a.Authenticator == nil {
		return ""
	}
	return a.Authenticator.domainName
}

// DomainVersion returns the EIP-712 domain version
func (a *AccountAuthenticator) DomainVersion() string {
	if a == nil || a.Authenticator == nil {
		return ""
	}
	return a.Authenticator.domainVersion
}

// ChainID returns the EIP-712 chain ID
func (a *AccountAuthenticator) ChainID() int {
	if a == nil || a.Authenticator == nil {
		return 0
	}
	return a.Authenticator.chainID
}

// Delegates to the underlying Authenticator
func (a *AccountAuthenticator) VerifyAccountOwner(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) (bool, error) {
	if a == nil || a.Authenticator == nil {
		return false, errSubaccountClientNotConfigured
	}
	return a.Authenticator.VerifyAccountOwner(ethereumAddress, accountId)
}
