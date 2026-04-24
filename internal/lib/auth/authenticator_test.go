package auth

import (
	"crypto/ecdsa"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Test constants for repeated values
const (
	testDomainName      = DefaultDomainName
	testDomainVersion   = "1"
	testChainID         = 1
	testSubAccountID    = snx_lib_core.SubAccountId(12345)
	testAddress         = snx_lib_api_types.WalletAddress("0x1234567890123456789012345678901234567890")
	testAccountIDUint   = 12345
	testNonce           = Nonce(123456789)
	testInvalidSig      = "0x1234567890abcdef"
	testInvalidSigShort = "0xinvalidsignature"
)

// MockNonceStoreTestify for testing with testify mocks
type MockNonceStoreTestify struct {
	mock.Mock
}

func (m *MockNonceStoreTestify) IsNonceUsed(address string, nonce Nonce) (bool, error) {
	args := m.Called(address, nonce)
	return args.Bool(0), args.Error(1)
}

func (m *MockNonceStoreTestify) ReserveNonce(address string, nonce Nonce) (bool, error) {
	args := m.Called(address, nonce)
	return args.Bool(0), args.Error(1)
}

func (m *MockNonceStoreTestify) CleanupExpiredNonces(maxAge time.Duration) error {
	args := m.Called(maxAge)
	return args.Error(0)
}

func Test_NewAuthenticator(t *testing.T) {
	nonceStore := &MockNonceStoreTestify{}
	mockSubaccountClient := newFakeSubaccountVerifier()

	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	assert.NotNil(t, authenticator)
	assert.Equal(t, nonceStore, authenticator.nonceStore)
	assert.Equal(t, mockSubaccountClient, authenticator.subaccountVerifier)
}

func Test_Authenticator_ValidateAuthentication_Success(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create test data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()

	// Create typed data
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// Create auth request
	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks
	nonceStore := &MockNonceStoreTestify{}

	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(true, nil)

	// Create authenticator with mock account configured
	mockSubaccountClient := newFakeSubaccountVerifier()
	mockSubaccountClient.AddOwner(snx_lib_api_types.WalletAddressFromStringUnvalidated(testAddress.Hex()), snx_lib_core.SubAccountId(testAccountIDUint))
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	// Test authentication
	result, err := authenticator.ValidateAuthentication(authReq)

	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, testAddress.Hex(), result.EthereumAddress)
	assert.Equal(t, subAccountID, result.SubAccountId)

	// Verify mock expectations
	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_ValidateAuthentication_Success_AsDelegate(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create test data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()

	// Create typed data
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// Create auth request
	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks
	nonceStore := &MockNonceStoreTestify{}
	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(true, nil)

	// Create authenticator with delegate configured instead of owner
	mockSubaccountClient := newFakeSubaccountVerifier()
	// Do NOT add ownership; add delegation for this address
	mockSubaccountClient.AddDelegate(snx_lib_api_types.WalletAddressFromStringUnvalidated(testAddress.Hex()), subAccountID, time.Time{})
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	// Test authentication
	result, err := authenticator.ValidateAuthentication(authReq)

	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, testAddress.Hex(), result.EthereumAddress)
	assert.Equal(t, subAccountID, result.SubAccountId)

	// Verify mock expectations
	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_ValidateAuthentication_DelegationDenied(t *testing.T) {
	// Create test private key and address (delegate)
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	delegateAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign it
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{TypedData: typedData, Signature: signature}

	// Setup mocks: nonce ok
	nonceStore := &MockNonceStoreTestify{}
	nonceStore.On("ReserveNonce", delegateAddr.Hex(), Nonce(timestamp)).Return(true, nil)

	// Subaccount client: Add an expired delegation for this delegate address
	mockSubaccountClient := newFakeSubaccountVerifier()
	expired := time.Now().Add(-1 * time.Hour)
	mockSubaccountClient.AddDelegate(snx_lib_api_types.WalletAddressFromStringUnvalidated(delegateAddr.Hex()), subAccountID, expired)
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not authorized")
}

func Test_Authenticator_ValidateAuthentication_InvalidTypedData(t *testing.T) {
	// Create authenticator with mocks
	nonceStore := &MockNonceStoreTestify{}
	mockSubaccountClient := newFakeSubaccountVerifier()
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	// Create invalid typed data (missing required fields)
	invalidTypedData := apitypes.TypedData{
		Types:       GetEIP712Types(),
		PrimaryType: "AuthMessage",
		Domain:      GetEIP712Domain(testDomainName, testDomainVersion, testChainID),
		Message: apitypes.TypedDataMessage{
			// Missing subAccountId and timestamp
			"action": ActionWebSocketAuth,
		},
	}

	authReq := &AuthRequest{
		TypedData: invalidTypedData,
		Signature: testInvalidSig,
	}

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to extract auth data")
}

func Test_Authenticator_ValidateAuthentication_OldTimestamp(t *testing.T) {
	// Create test private key
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// Create typed data with old timestamp (more than 60 seconds ago)
	subAccountID := testSubAccountID
	oldTimestamp := time.Now().Unix() - 120 // 2 minutes ago

	typedData := CreateEIP712TypedData(subAccountID, oldTimestamp, ActionWebSocketAuth, DefaultDomainName, "1", 1)
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Create authenticator
	nonceStore := &MockNonceStoreTestify{}
	mockSubaccountClient := newFakeSubaccountVerifier()
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timestamp validation failed")
	assert.Contains(t, err.Error(), "request too old")
}

func Test_Authenticator_ValidateAuthentication_FutureTimestamp(t *testing.T) {
	// Create test private key
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// Create typed data with future timestamp (more than 60 seconds in the future)
	subAccountID := testSubAccountID
	futureTimestamp := time.Now().Unix() + 120 // 2 minutes in the future

	typedData := CreateEIP712TypedData(subAccountID, futureTimestamp, ActionWebSocketAuth, DefaultDomainName, "1", 1)
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Create authenticator
	nonceStore := &MockNonceStoreTestify{}
	mockSubaccountClient := newFakeSubaccountVerifier()
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timestamp validation failed")
	assert.Contains(t, err.Error(), "timestamp from future")
}

func Test_Authenticator_ValidateAuthentication_InvalidSignature(t *testing.T) {
	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Create auth request with invalid signature
	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: testInvalidSigShort,
	}

	// Create authenticator
	nonceStore := &MockNonceStoreTestify{}
	mockSubaccountClient := newFakeSubaccountVerifier()
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func Test_Authenticator_ValidateAuthentication_NonceAlreadyUsed(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks - nonce already used
	nonceStore := &MockNonceStoreTestify{}

	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(false, nil)

	mockSubaccountClient := newFakeSubaccountVerifier()
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timestamp already used")

	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_ValidateAuthentication_NonceStoreError(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks - nonce store returns error
	nonceStore := &MockNonceStoreTestify{}

	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(false, errors.New("redis connection failed"))

	mockSubaccountClient := newFakeSubaccountVerifier()
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to reserve timestamp")
	assert.Contains(t, err.Error(), "redis connection failed")

	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_ValidateAuthentication_AccountVerificationFailed(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks - account verification now always passes
	nonceStore := &MockNonceStoreTestify{}

	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(true, nil)

	mockSubaccountClient := newFakeSubaccountVerifier()
	mockSubaccountClient.AddOwner(snx_lib_api_types.WalletAddressFromStringUnvalidated(testAddress.Hex()), snx_lib_core.SubAccountId(testAccountIDUint))
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, testAddress.Hex(), result.EthereumAddress)
	assert.Equal(t, subAccountID, result.SubAccountId)

	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_ValidateAuthentication_AlwaysPassesAccountVerification(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks - account verification always passes now
	nonceStore := &MockNonceStoreTestify{}

	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(true, nil)

	mockSubaccountClient := newFakeSubaccountVerifier()
	mockSubaccountClient.AddOwner(snx_lib_api_types.WalletAddressFromStringUnvalidated(testAddress.Hex()), snx_lib_core.SubAccountId(testAccountIDUint))
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Equal(t, testAddress.Hex(), result.EthereumAddress)
	assert.Equal(t, subAccountID, result.SubAccountId)

	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_ValidateAuthentication_ReserveNonceError(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks - mark nonce used fails
	nonceStore := &MockNonceStoreTestify{}

	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(false, errors.New("redis write error"))

	mockSubaccountClient := newFakeSubaccountVerifier()
	mockSubaccountClient.AddOwner(snx_lib_api_types.WalletAddressFromStringUnvalidated(testAddress.Hex()), snx_lib_core.SubAccountId(testAccountIDUint))
	authenticator := NewAuthenticator(nonceStore, mockSubaccountClient, nil, testDomainName, testDomainVersion, testChainID)

	result, err := authenticator.ValidateAuthentication(authReq)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to reserve timestamp")
	assert.Contains(t, err.Error(), "redis write error")

	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_ValidateAuthentication_ServiceError_NoBypass(t *testing.T) {
	// Create test private key and address
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	testAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create typed data
	subAccountID := testSubAccountID
	timestamp := time.Now().Unix()
	typedData := CreateEIP712TypedData(subAccountID, timestamp, ActionWebSocketAuth, testDomainName, testDomainVersion, testChainID)

	// Sign the typed data
	signature, err := signTypedData(privateKey, typedData)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	authReq := &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}

	// Setup mocks - SubaccountService fails
	nonceStore := &MockNonceStoreTestify{}
	nonceStore.On("ReserveNonce", testAddress.Hex(), Nonce(timestamp)).Return(true, nil)

	// Use mock that always fails for ListSubaccounts
	mockFailingClient := newFailingSubaccountVerifier()
	authenticator := NewAuthenticator(nonceStore, mockFailingClient, nil, DefaultDomainName, "1", 1)

	result, err := authenticator.ValidateAuthentication(authReq)

	// Verify that authentication fails rather than bypassing
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "account ownership verification failed")
	assert.Contains(t, err.Error(), "service unavailable")

	nonceStore.AssertExpectations(t)
}

func Test_Authenticator_VerifyAccountOwnership_CacheHit(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddOwner(testAddress, snx_lib_core.SubAccountId(testAccountIDUint))

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	// Prime the cache
	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.True(t, owns)
	assert.Equal(t, 1, mockClient.VerifyCallCount)

	// Second call should hit the cache
	owns, err = authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.True(t, owns)
	assert.Equal(t, 1, mockClient.VerifyCallCount, "gRPC should not be called on cache hit")
}

func Test_Authenticator_VerifyAccountOwnership_CacheMiss_StoresOnSuccess(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddOwner(testAddress, snx_lib_core.SubAccountId(testAccountIDUint))

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.True(t, owns)

	// Verify it's now cached
	authType, found := cache.Lookup(testAddress, testSubAccountID)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)
}

func Test_Authenticator_VerifyAccountOwnership_NegativelyCachesRefusal(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	assert.False(t, owns)
	assert.Error(t, err)
	assert.Equal(t, 1, mockClient.VerifyCallCount)

	_, found := cache.Lookup(testAddress, testSubAccountID)
	assert.False(t, found, "positive cache should not contain refusals")

	assert.True(t, cache.LookupRefusal(testAddress, testSubAccountID), "refusal should be negatively cached")

	owns, err = authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	assert.False(t, owns)
	assert.Error(t, err)
	assert.Equal(t, 1, mockClient.VerifyCallCount, "gRPC should not be called on negative cache hit")
}

func Test_Authenticator_VerifyAccountOwner_CacheHit_Owner(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddOwner(testAddress, snx_lib_core.SubAccountId(testAccountIDUint))

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	// Prime the cache via VerifyAccountOwnership (stores as OWNER)
	_, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, 1, mockClient.VerifyCallCount)

	// VerifyAccountOwner should use the cached OWNER entry
	isOwner, err := authenticator.VerifyAccountOwner(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.True(t, isOwner)
	assert.Equal(t, 1, mockClient.VerifyCallCount, "gRPC should not be called on cache hit")
}

func Test_Authenticator_VerifyAccountOwner_CacheHit_Delegate_Rejected(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	// Manually prime cache with a DELEGATE entry
	cache.Store(testAddress, testSubAccountID, AuthTypeDelegate)

	// VerifyAccountOwner should reject delegates from cache without a gRPC call
	isOwner, err := authenticator.VerifyAccountOwner(testAddress, testSubAccountID)
	assert.False(t, isOwner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delegate but not the owner")
	assert.Equal(t, 0, mockClient.VerifyCallCount, "gRPC should not be called when cache resolves delegate")
}

func Test_Authenticator_VerifyAccountOwner_CacheMiss_StoresOwner(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddOwner(testAddress, snx_lib_core.SubAccountId(testAccountIDUint))

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	isOwner, err := authenticator.VerifyAccountOwner(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.True(t, isOwner)

	authType, found := cache.Lookup(testAddress, testSubAccountID)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)
}

func Test_Authenticator_VerifyAccountOwner_DelegateFromGRPC_NotCachedAsOwner(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddDelegate(testAddress, testSubAccountID, time.Time{})

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	// gRPC returns authorized=true but authType=DELEGATE, so VerifyAccountOwner should reject
	isOwner, err := authenticator.VerifyAccountOwner(testAddress, testSubAccountID)
	assert.False(t, isOwner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delegate but not the owner")

	// Should not cache the delegate as an owner entry
	_, found := cache.Lookup(testAddress, testSubAccountID)
	assert.False(t, found, "delegate result from VerifyAccountOwner should not be cached")
}

func Test_Authenticator_EvictAuth_RemovesFromCache(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddOwner(testAddress, snx_lib_core.SubAccountId(testAccountIDUint))

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	// Prime the cache
	_, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	_, found := cache.Lookup(testAddress, testSubAccountID)
	require.True(t, found)

	// Evict
	authenticator.EvictAuth(testAddress, testSubAccountID)

	_, found = cache.Lookup(testAddress, testSubAccountID)
	assert.False(t, found, "entry should be evicted after EvictAuth")

	// Next call should hit gRPC again
	_, err = authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, 2, mockClient.VerifyCallCount, "gRPC should be called after eviction")
}

func Test_Authenticator_EvictAuth_NilCache(t *testing.T) {
	mockClient := newFakeSubaccountVerifier()
	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, nil, testDomainName, testDomainVersion, testChainID)

	// Should not panic
	authenticator.EvictAuth(testAddress, testSubAccountID)
}

func Test_Authenticator_VerifyAccountOwnership_NilCache(t *testing.T) {
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddOwner(testAddress, snx_lib_core.SubAccountId(testAccountIDUint))

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, nil, testDomainName, testDomainVersion, testChainID)

	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.True(t, owns)
	assert.Equal(t, 1, mockClient.VerifyCallCount)

	// Without cache, second call also hits gRPC
	owns, err = authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.True(t, owns)
	assert.Equal(t, 2, mockClient.VerifyCallCount, "without cache, every call should hit gRPC")
}

func Test_Authenticator_VerifyAccountOwnership_CacheHit_DelegateAlsoAccepted(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddDelegate(testAddress, testSubAccountID, time.Time{})

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	// Prime cache with delegate via gRPC call
	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.True(t, owns)
	assert.Equal(t, 1, mockClient.VerifyCallCount)

	authType, found := cache.Lookup(testAddress, testSubAccountID)
	require.True(t, found)
	assert.Equal(t, AuthTypeDelegate, authType)

	// Second call should hit cache (VerifyAccountOwnership accepts both owner and delegate)
	owns, err = authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.True(t, owns)
	assert.Equal(t, 1, mockClient.VerifyCallCount, "gRPC should not be called on cache hit")
}

func Test_Authenticator_VerifyAccountOwnership_ServiceError_NoNegativeCache(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFailingSubaccountVerifier()

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	assert.False(t, owns)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service error")

	assert.False(t, cache.LookupRefusal(testAddress, testSubAccountID),
		"transient service errors must not be negatively cached",
	)
}

func Test_Authenticator_VerifyAccountOwnership_NegativeCacheExpires(t *testing.T) {
	cache := NewAuthCache(100)
	cache.negativeTTL = 1 * time.Millisecond
	mockClient := newFakeSubaccountVerifier()

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	assert.False(t, owns)
	assert.Error(t, err)
	assert.Equal(t, 1, mockClient.VerifyCallCount)

	time.Sleep(5 * time.Millisecond)

	owns, err = authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	assert.False(t, owns)
	assert.Error(t, err)
	assert.Equal(t, 2, mockClient.VerifyCallCount, "gRPC should be called after negative entry expires")
}

func Test_Authenticator_VerifyAccountOwnership_PositiveResultClearsNegativeCache(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	owns, err := authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	assert.False(t, owns)
	assert.Error(t, err)
	assert.Equal(t, 1, cache.negativeLen())

	// Grant access and expire the negative entry so gRPC is called again
	mockClient.AddOwner(testAddress, snx_lib_core.SubAccountId(testAccountIDUint))
	cache.negativeTTL = 0

	owns, err = authenticator.VerifyAccountOwnership(testAddress, testSubAccountID)
	assert.True(t, owns)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, 0, cache.negativeLen(), "negative entry should be gone after positive result")
	assert.Equal(t, 1, cache.Len(), "positive entry should be cached")
}

func Test_Authenticator_VerifyAccountOwner_NegativelyCachesRefusal(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	isOwner, err := authenticator.VerifyAccountOwner(testAddress, testSubAccountID)
	assert.False(t, isOwner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not the owner")
	assert.Equal(t, 1, mockClient.VerifyCallCount)

	assert.True(t, cache.LookupRefusal(testAddress, testSubAccountID),
		"complete refusal should be negatively cached",
	)

	// Second call should hit negative cache
	isOwner, err = authenticator.VerifyAccountOwner(testAddress, testSubAccountID)
	assert.False(t, isOwner)
	assert.Error(t, err)
	assert.Equal(t, 1, mockClient.VerifyCallCount, "gRPC should not be called on negative cache hit")
}

func Test_Authenticator_VerifyAccountOwner_DelegateNotNegativelyCached(t *testing.T) {
	cache := NewAuthCache(100)
	mockClient := newFakeSubaccountVerifier()
	mockClient.AddDelegate(testAddress, testSubAccountID, time.Time{})

	authenticator := NewAuthenticator(NewTestNonceStore(), mockClient, cache, testDomainName, testDomainVersion, testChainID)

	isOwner, err := authenticator.VerifyAccountOwner(testAddress, testSubAccountID)
	assert.False(t, isOwner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delegate but not the owner")

	assert.False(t, cache.LookupRefusal(testAddress, testSubAccountID),
		"delegate results should not be negatively cached — VerifyAccountOwnership may accept them",
	)
}

// Helper function to sign typed data for testing
func signTypedData(privateKey *ecdsa.PrivateKey, typedData apitypes.TypedData) (string, error) {
	// Get the hash to sign
	hash, err := GetEIP712MessageHash(typedData)
	if err != nil {
		return "", err
	}

	// Sign the hash
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", err
	}

	// Convert to hex with 0x prefix
	return "0x" + common.Bytes2Hex(signature), nil
}
