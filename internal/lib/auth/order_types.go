package auth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Extracts common account-auth fields from typed-data messages.
// Missing nonces return zero; callers decide whether zero is allowed.
func extractSubaccountAuthData(message map[string]any) (
	subAccountId snx_lib_core.SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	err error,
) {
	// Extract subAccountId
	var subAccountInt int64
	if subAccountInt, err = extractInt64Value(message, "subAccountId"); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to extract subAccountId: %w", err)
	}
	subAccountId = snx_lib_core.SubAccountId(subAccountInt)

	// Extract nonce (optional for SubAccountAction/get* requests)
	if _, hasNonce := message["nonce"]; hasNonce {
		if nonce, err = extractNonce(message); err != nil {
			return 0, 0, 0, fmt.Errorf("failed to extract nonce: %w", err)
		}
	}

	// Extract expiresAfter
	if expiresAfter, err = extractInt64Value(message, "expiresAfter"); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to extract expiresAfter: %w", err)
	}

	return subAccountId, nonce, expiresAfter, nil
}

// Extracts authentication data for order requests.
func ExtractOrderAuthData(typedData apitypes.TypedData) (
	subAccountId snx_lib_core.SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	err error,
) {
	return extractSubaccountAuthData(typedData.Message)
}

// Extracts authentication data for subaccount creation requests.
// The master subaccount field is used for ownership verification.
func ExtractCreateSubaccountAuthData(typedData apitypes.TypedData) (
	subAccountId snx_lib_core.SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	err error,
) {
	message := typedData.Message

	// Extract masterSubAccountId (used as subAccountId for ownership verification)
	var subAccountInt int64
	if subAccountInt, err = extractInt64Value(message, "masterSubAccountId"); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to extract masterSubAccountId: %w", err)
	}
	subAccountId = snx_lib_core.SubAccountId(subAccountInt)

	// Extract nonce (required for createSubaccount)
	if nonce, err = extractNonce(message); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to extract nonce: %w", err)
	}

	// Extract expiresAfter
	if expiresAfter, err = extractInt64Value(message, "expiresAfter"); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to extract expiresAfter: %w", err)
	}

	return subAccountId, nonce, expiresAfter, nil
}
