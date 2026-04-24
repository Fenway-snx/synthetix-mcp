package auth

// MockAuthenticator is a simple mock that always succeeds for testing
type MockAuthenticator struct{}

// NewMockAuthenticator creates a new mock authenticator for testing
func NewMockAuthenticator() AuthenticatorInterface {
	return &MockAuthenticator{}
}

// ValidateAuthentication always returns a successful authentication for testing
func (m *MockAuthenticator) ValidateAuthentication(req *AuthRequest) (*AuthResult, error) {
	// Extract sub-account ID from the typed data
	subAccountID, _, err := ExtractAuthData(req.TypedData)
	if err != nil {
		// For testing, provide a default if extraction fails
		subAccountID = 12345
	}

	return &AuthResult{
		Valid:           true,
		SubAccountId:    subAccountID,
		EthereumAddress: "0x1234567890123456789012345678901234567890", // Mock address
	}, nil
}
