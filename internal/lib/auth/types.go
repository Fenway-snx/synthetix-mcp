package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

var (
	errRequestTooOld       = errors.New("request too old")
	errTimestampFromFuture = errors.New("timestamp from future")
	errValueExceedsInt64   = errors.New("value exceeds 64-bit integer")
)

// AuthRequest represents the authentication data sent by the client
type AuthRequest struct {
	TypedData apitypes.TypedData // The parsed EIP-712 typed data
	Signature string             // The signature in hex format
}

// AuthResult represents the result of authentication
type AuthResult struct {
	EthereumAddress string
	SubAccountId    snx_lib_core.SubAccountId
	Valid           bool
}

// Interface for verifying account ownership.
type AccountVerifier interface {
	VerifyAccountOwnership(
		ethereumAddress snx_lib_api_types.WalletAddress,
		accountId snx_lib_core.SubAccountId,
	) (bool, error)
}

// Interface for tracking used nonces.
type NonceStore interface {
	IsNonceUsed(address string, nonce Nonce) (bool, error)
	// ReserveNonce atomically reserves a nonce if it hasn't been used yet
	// Returns true if the nonce was successfully reserved, false if already used
	ReserveNonce(address string, nonce Nonce) (bool, error)
	CleanupExpiredNonces(maxAge time.Duration) error
}

// AuthenticatorInterface defines the interface for authentication validators
type AuthenticatorInterface interface {
	ValidateAuthentication(req *AuthRequest) (*AuthResult, error)
}

// AccountAuthenticatorInterface defines the interface for account authentication
type AccountAuthenticatorInterface interface {
	Enabled() bool
	DomainName() string
	DomainVersion() string
	ChainID() int
	ValidateAccountAuth(
		typedData apitypes.TypedData,
		signature string,
		extractFunc DataExtractor,
		opts *AuthOptions,
	) (*AuthResult, error)
	VerifyAccountOwner(
		ethereumAddress snx_lib_api_types.WalletAddress,
		accountId snx_lib_core.SubAccountId,
	) (bool, error)
}

// ParseAuthMessage parses the authentication message and creates an auth request
func ParseAuthMessage(authMessage, signature string) (*AuthRequest, error) {
	// Parse the typed data from JSON
	var typedData apitypes.TypedData
	if err := json.Unmarshal([]byte(authMessage), &typedData); err != nil {
		return nil, fmt.Errorf("failed to parse typed data: %w", err)
	}

	return &AuthRequest{
		TypedData: typedData,
		Signature: signature,
	}, nil
}

// ExtractAuthData extracts authentication data from typed data message
func ExtractAuthData(typedData apitypes.TypedData) (
	subAccountId snx_lib_core.SubAccountId,
	timestamp int64,
	err error,
) {
	message := typedData.Message

	// Extract sub-account ID
	var subAccountIdInt int64
	subAccountIdInt, err = extractInt64Value(message, "subAccountId")
	if err != nil {
		return 0, 0, err
	}
	subAccountId = snx_lib_core.SubAccountId(subAccountIdInt)

	// Extract timestamp
	timestamp, err = extractInt64Value(message, "timestamp")
	if err != nil {
		return 0, 0, err
	}

	return subAccountId, timestamp, nil
}

// ValidateTimestamp checks if the request timestamp is recent
func ValidateTimestamp(timestamp int64) error {
	now := snx_lib_utils_time.Now().Unix()
	maxAge := int64(60) // 1 minute

	// Convert millisecond timestamp to seconds if needed
	// Detect if timestamp is in milliseconds (> year 2038 when expressed as seconds)
	timestampSeconds := timestamp
	if timestamp > 2147483647 { // Unix timestamp for 2038-01-19, beyond this it must be milliseconds
		timestampSeconds = timestamp / 1000
	}

	if timestampSeconds < now-maxAge {
		return errRequestTooOld
	}

	if timestampSeconds > now+60 { // Allow 1 minute clock skew
		return errTimestampFromFuture
	}

	return nil
}

// GenerateTimestamp creates a new timestamp for authentication
func GenerateTimestamp() int64 {
	// Use current Unix timestamp as the nonce
	// This provides replay protection while being meaningful to users
	return snx_lib_utils_time.Now().Unix()
}

// extractNumericValue extracts a numeric value from a typed data message and returns it as a *big.Int
// This handles the common parsing logic for different input formats
func extractNumericValue(message apitypes.TypedDataMessage, key string) (*big.Int, error) {
	value, exists := message[key]
	if !exists {
		return nil, fmt.Errorf("missing %s in message", key)
	}

	switch v := value.(type) {
	case float64:
		return big.NewInt(int64(v)), nil
	case string:
		// Handle string representation (both hex and decimal)
		if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
			// Parse as hex
			bigInt := new(big.Int)
			_, parseOk := bigInt.SetString(v[2:], 16)
			if !parseOk {
				return nil, fmt.Errorf("invalid hex %s format", key)
			}
			return bigInt, nil
		}
		// Parse as decimal
		bigInt := new(big.Int)
		_, parseOk := bigInt.SetString(v, 10)
		if !parseOk {
			return nil, fmt.Errorf("invalid decimal %s format", key)
		}
		return bigInt, nil
	case *math.HexOrDecimal256:
		// Handle HexOrDecimal256 format used in typed data
		return (*big.Int)(v), nil
	default:
		return nil, fmt.Errorf("invalid %s type: %T", key, v)
	}
}

// Extracts a uint256 value from a typed data message.
//
// Deprecated: remove this dead code soon (once happy with `int64` SIDs)
func extractUint256Value(message apitypes.TypedDataMessage, key string) (uint64, error) {
	bigInt, err := extractNumericValue(message, key)
	if err != nil {
		return 0, err
	}
	return bigInt.Uint64(), nil
}

// Extracts a signed 64-bit value from a typed data message.
func extractInt64Value(message apitypes.TypedDataMessage, key string) (int64, error) {
	bigInt, err := extractNumericValue(message, key)
	if err != nil {
		return 0, err
	}

	if !bigInt.IsInt64() {
		return 0, errValueExceedsInt64
	}

	return bigInt.Int64(), nil
}

func extractNonce(message apitypes.TypedDataMessage) (r Nonce, err error) {
	var i int64
	if i, err = extractInt64Value(message, "nonce"); err == nil {
		r = Nonce(i)
	}

	return
}

// AuthConfig represents EIP-712 authentication configuration
type AuthConfig struct {
	DomainName    string
	DomainVersion string
	ChainID       int
}

// ValidateAuthConfig validates authentication configuration and returns validation errors
func ValidateAuthConfig(config AuthConfig) []string {
	var errors []string

	if config.DomainName == "" {
		errors = append(errors, "domain_name is required")
	}
	if config.DomainVersion == "" {
		errors = append(errors, "domain_version is required")
	}
	if config.ChainID <= 0 {
		errors = append(errors, "chain_id must be greater than 0")
	}

	return errors
}
