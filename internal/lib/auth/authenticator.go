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

// Handles Ethereum signature-based authentication.
type Authenticator struct {
	authCache          *AuthCache
	nonceStore         NonceStore
	subaccountVerifier SubaccountVerifier
	// EIP-712 domain configuration
	domainName    string
	domainVersion string
	chainID       int
}

// Creates a signer verifier with optional authorization caching.
// A nil verifier is valid when an external path pre-primes the cache.
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

// Validates an Ethereum signature authentication request.
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
	// Keep the legacy verifier path for compatibility.
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

// Checks owner or delegated trading access with cached positive/refusal results.
// Service errors are never cached.
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

// Checks owner-only access for sensitive actions such as withdrawals.
// Definitive refusals are cached; delegate responses are not.
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

// Removes cached authorization after a delegation is revoked.
func (a *Authenticator) EvictAuth(
	walletAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
) {
	if a.authCache != nil {
		a.authCache.Evict(walletAddress, subAccountId)
	}
}

// Provides unified authentication for account-based operations.
type AccountAuthenticator struct {
	*Authenticator
}

// Wraps the base verifier for account-scoped checks.
func NewAccountAuthenticator(base *Authenticator) *AccountAuthenticator {
	return &AccountAuthenticator{Authenticator: base}
}

// Configures authentication behavior.
type AuthOptions struct {
	// SupportExpiration enables expiration checking (default: false)
	SupportExpiration bool
	// UseTimestampNonce uses timestamp as nonce instead of separate nonce field (default: false)
	UseTimestampNonce bool
	// Require the account owner rather than a delegate.
	RequireOwner bool
	// Skip replay protection for read-only actions.
	SkipNonceCheck bool
}

// Returns default authentication settings.
func DefaultAuthOptions() *AuthOptions {
	return &AuthOptions{
		SupportExpiration: false,
		UseTimestampNonce: false,
	}
}

// Extracts authentication fields from typed data.
type DataExtractor func(apitypes.TypedData) (
	subAccountId snx_lib_core.SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	err error,
)

// Performs account authentication with pluggable data extraction.
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

	// 1. Validate EIP-712 domain binding before any other checks.
	expectedDomain := GetEIP712Domain(a.domainName, a.domainVersion, a.chainID)

	// Ensure the signed domain matches this server.
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
	// Note: VerifyingContract is always 0x0000... for this signing flow, but we check it for completeness
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

	// 6. Reserve the nonce immediately after signature recovery.
	if opts.SkipNonceCheck {
		// Skip nonce validation for read-only operations (e.g., get* actions)
	} else {
		// State-changing operations require replay protection.
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

// Reports whether authentication is enabled.
func (a *AccountAuthenticator) Enabled() bool {
	return a != nil && a.Authenticator != nil
}

// Returns the EIP-712 domain name.
func (a *AccountAuthenticator) DomainName() string {
	if a == nil || a.Authenticator == nil {
		return ""
	}
	return a.Authenticator.domainName
}

// Returns the EIP-712 domain version.
func (a *AccountAuthenticator) DomainVersion() string {
	if a == nil || a.Authenticator == nil {
		return ""
	}
	return a.Authenticator.domainVersion
}

// Returns the EIP-712 chain ID.
func (a *AccountAuthenticator) ChainID() int {
	if a == nil || a.Authenticator == nil {
		return 0
	}
	return a.Authenticator.chainID
}

// Delegates to the underlying verifier.
func (a *AccountAuthenticator) VerifyAccountOwner(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) (bool, error) {
	if a == nil || a.Authenticator == nil {
		return false, errSubaccountClientNotConfigured
	}
	return a.Authenticator.VerifyAccountOwner(ethereumAddress, accountId)
}
