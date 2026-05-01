package auth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

var (
	errSignatureMustBe65Bytes = errors.New("signature must be 65 bytes")
)

// EIP712 action types
const (
	ActionWebSocketAuth = "websocket_auth" // Action type for WebSocket authentication
)

// Standard EIP-712 domain name for Synthetix.
const DefaultDomainName = "Synthetix"

// Returns the common EIP-712 domain fields.
func getEIP712DomainFields() []apitypes.Type {
	return []apitypes.Type{
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
	}
}

// Returns the domain separator for WebSocket authentication.
func GetEIP712Domain(name, version string, chainID int) apitypes.TypedDataDomain {
	return apitypes.TypedDataDomain{
		Name:              name,
		Version:           version,
		ChainId:           math.NewHexOrDecimal256(int64(chainID)),
		VerifyingContract: "0x0000000000000000000000000000000000000000",
	}
}

// Returns type definitions for the authentication message.
func GetEIP712Types() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"AuthMessage": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "timestamp", Type: "uint256"},
			{Name: "action", Type: "string"},
		},
	}
}

// Creates typed data for signing.
func CreateEIP712TypedData(
	subAccountId snx_lib_core.SubAccountId,
	timestamp int64,
	action string,
	domainName string,
	domainVersion string,
	chainID int,
) apitypes.TypedData {
	return apitypes.TypedData{
		Types:       GetEIP712Types(),
		PrimaryType: "AuthMessage",
		Domain:      GetEIP712Domain(domainName, domainVersion, chainID),
		Message: apitypes.TypedDataMessage{
			"subAccountId": math.NewHexOrDecimal256(int64(subAccountId)), // TODO: create specific converter function for this
			"timestamp":    math.NewHexOrDecimal256(timestamp),
			"action":       action,
		},
	}
}

// Recovers the signer address from an EIP-712 signature.
// Recovery IDs are normalized from Ethereum's 27/28 convention to 0/1.
func VerifyEIP712Signature(typedData apitypes.TypedData, signatureHex string) (common.Address, error) {
	// Remove 0x prefix if present
	if len(signatureHex) >= 2 && signatureHex[:2] == "0x" {
		signatureHex = signatureHex[2:]
	}

	// Decode signature
	signature := common.FromHex(signatureHex)
	if len(signature) != 65 {
		return common.Address{}, errSignatureMustBe65Bytes
	}

	digest, err := GetEIP712MessageHash(typedData)
	if err != nil {
		return common.Address{}, err
	}

	// Adjust recovery ID for Ethereum
	if signature[64] >= 27 {
		signature[64] -= 27
	}

	// Recover public key
	pubKey, err := crypto.SigToPub(digest, signature)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Get address from public key
	address := crypto.PubkeyToAddress(*pubKey)
	return address, nil
}

// Computes the EIP-712 digest used for signing or verification.
// The digest is keccak256("\x19\x01" || domainSeparator || hashStruct(message)).
func GetEIP712MessageHash(typedData apitypes.TypedData) ([]byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash message: %w", err)
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	return crypto.Keccak256(rawData), nil
}

// Converts typed data to JSON for client-side signing.
func SerializeTypedData(typedData apitypes.TypedData) (string, error) {
	// Create a structure that matches what wallets expect
	data := map[string]any{
		API_WKS_types:       typedData.Types,
		API_WKS_primaryType: typedData.PrimaryType,
		API_WKS_domain:      typedData.Domain.Map(),
		API_WKS_message:     typedData.Message,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize typed data: %w", err)
	}

	return string(jsonBytes), nil
}
