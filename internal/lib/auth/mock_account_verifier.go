package auth

import (
	"strings"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// MockAccountVerifier implements AccountVerifier for testing/development
type MockAccountVerifier struct {
	// In a real implementation, this would query the database
	// For now, we'll use a simple mock that accepts certain patterns
}

// NewMockAccountVerifier creates a new mock account verifier
func NewMockAccountVerifier() AccountVerifier {
	return &MockAccountVerifier{}
}

// VerifyAccountOwnership verifies if an Ethereum address owns a given account
func (m *MockAccountVerifier) VerifyAccountOwnership(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) (bool, error) {
	// Mock implementation - in production this would:
	// 1. Query the database for the account
	// 2. Check if the ethereum address is the owner
	// 3. Return true/false based on ownership

	// For testing, we'll accept any valid Ethereum address format
	// and any account ID that is valid

	// Validate Ethereum address format
	if !isValidEthereumAddress(ethereumAddress) {
		return false, nil
	}

	// For mock purposes, accept any non-zero account ID
	// In production, this would be a database lookup
	if accountId == 0 {
		return false, nil
	}

	return true, nil
}

// Checks if an address has valid Ethereum format.
func isValidEthereumAddress(address snx_lib_api_types.WalletAddress) bool {
	// Basic validation: should start with 0x and be 42 characters total
	if len(address) != 42 {
		return false
	}

	if !strings.HasPrefix(string(address), "0x") {
		return false
	}

	// Check if remaining characters are valid hex
	for i := 2; i < len(address); i++ {
		char := address[i]
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'f') ||
			(char >= 'A' && char <= 'F')) {
			return false
		}
	}

	return true
}
