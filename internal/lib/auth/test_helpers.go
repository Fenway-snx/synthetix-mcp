package auth

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Grpc-typed test doubles (MockSubaccountServiceClient, CreateTestAuthenticator,
// etc.) live in the lib/auth/authtest sub-package so production builds of
// lib/auth stay free of google.golang.org/grpc.

// Test-only AccountVerifier that always succeeds; lets tests skip the
// owner-verification round-trip without standing up Redis or a real verifier.
type TestAuthenticator struct{}

func NewTestAuthenticator() AccountVerifier {
	return &TestAuthenticator{}
}

func (ta *TestAuthenticator) VerifyAccountOwnership(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) (bool, error) {
	return true, nil
}

// Test-only NonceStore that never blocks reservations. Useful for tests that
// need a working store without a Redis dependency.
type TestNonceStore struct{}

func NewTestNonceStore() NonceStore {
	return &TestNonceStore{}
}

func (tns *TestNonceStore) IsNonceUsed(address string, nonce Nonce) (bool, error) {
	return false, nil
}

func (tns *TestNonceStore) ReserveNonce(address string, nonce Nonce) (bool, error) {
	return true, nil
}

func (tns *TestNonceStore) CleanupExpiredNonces(maxAge time.Duration) error {
	return nil
}

// Bypass authenticator that returns success for any well-formed request.
type TestFullAuthenticator struct{}

func NewTestFullAuthenticator() *TestFullAuthenticator {
	return &TestFullAuthenticator{}
}

func (tfa *TestFullAuthenticator) ValidateAuthentication(req *AuthRequest) (*AuthResult, error) {
	accountId, _, err := ExtractAuthData(req.TypedData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract auth data: %w", err)
	}
	return &AuthResult{
		EthereumAddress: "0x1234567890123456789012345678901234567890",
		SubAccountId:    accountId,
		Valid:           true,
	}, nil
}

// Deterministic EIP-712 signing wallet for tests. Pure crypto; no infra deps.
type TestWallet struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

// Returns a wallet derived from a fixed seed table so tests get stable addresses.
func NewTestWalletWithSeed(seed byte) *TestWallet {
	var privateKeyHex string
	switch seed {
	case 0:
		privateKeyHex = "7a9dce8c0e3421f45db16b44d3b5e3e9c4b9e1cb4a33f73f25c3c3c2c5beec00"
	case 1:
		privateKeyHex = "7a9dce8c0e3421f45db16b44d3b5e3e9c4b9e1cb4a33f73f25c3c3c2c5beec12"
	case 2:
		privateKeyHex = "7a9dce8c0e3421f45db16b44d3b5e3e9c4b9e1cb4a33f73f25c3c3c2c5beec13"
	case 3:
		privateKeyHex = "7a9dce8c0e3421f45db16b44d3b5e3e9c4b9e1cb4a33f73f25c3c3c2c5beec14"
	case 99:
		privateKeyHex = "7a9dce8c0e3421f45db16b44d3b5e3e9c4b9e1cb4a33f73f25c3c3c2c5beec99"
	default:
		privateKeyHex = "7a9dce8c0e3421f45db16b44d3b5e3e9c4b9e1cb4a33f73f25c3c3c2c5beec12"
	}

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		panic(fmt.Sprintf("Failed to decode private key hex: %v", err))
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		panic(fmt.Sprintf("Failed to create ECDSA private key: %v", err))
	}

	return &TestWallet{
		privateKey: privateKey,
		address:    crypto.PubkeyToAddress(privateKey.PublicKey),
	}
}

// Builds a serialized EIP-712 typed-data payload + 65-byte signature suitable
// for a WebSocket auth handshake.
func (w *TestWallet) GenerateAuthMessage(
	subAccountId snx_lib_core.SubAccountId,
) (message string, signature string, err error) {
	timestamp := time.Now().UTC().Unix()

	typedData := CreateEIP712TypedData(subAccountId, timestamp, ActionWebSocketAuth, DefaultDomainName, "1", 1)

	typedDataJSON, err := SerializeTypedData(typedData)
	if err != nil {
		return "", "", fmt.Errorf("failed to serialize typed data: %w", err)
	}

	digest, err := GetEIP712MessageHash(typedData)
	if err != nil {
		return "", "", fmt.Errorf("failed to get message hash: %w", err)
	}

	signatureBytes, err := crypto.Sign(digest, w.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign message: %w", err)
	}

	if signatureBytes[64] < 27 {
		signatureBytes[64] += 27
	}

	return typedDataJSON, "0x" + hex.EncodeToString(signatureBytes), nil
}

func (w *TestWallet) GetAddress() string {
	return w.address.Hex()
}

// Signs arbitrary EIP-712 typed data and returns the raw 65-byte signature.
func (w *TestWallet) SignTypedData(typedData apitypes.TypedData) ([]byte, error) {
	digest, err := GetEIP712MessageHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("failed to get typed data hash: %w", err)
	}

	signatureBytes, err := crypto.Sign(digest, w.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign typed data: %w", err)
	}

	if len(signatureBytes) != 65 {
		return nil, fmt.Errorf("unexpected signature length: %d", len(signatureBytes))
	}

	return signatureBytes, nil
}
