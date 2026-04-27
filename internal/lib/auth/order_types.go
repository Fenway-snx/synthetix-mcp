package auth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// extractSubaccountAuthData extracts common subaccount authentication data from typed data message.
// This function extracts only the fields needed for validation: subAccountId, nonce, and expiresAfter.
// Nonce extraction is lenient here (returns 0 if missing); enforcement of nonce requirements happens
// in ValidateAccountAuth based on the SkipNonceCheck option, which is set by NonceRequired().
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

// ExtractOrderAuthData extracts authentication data for order requests
func ExtractOrderAuthData(typedData apitypes.TypedData) (
	subAccountId snx_lib_core.SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	err error,
) {
	return extractSubaccountAuthData(typedData.Message)
}

// ExtractCreateSubaccountAuthData extracts authentication data for createSubaccount requests.
// For createSubaccount, the masterSubAccountId field is used for ownership verification.
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
