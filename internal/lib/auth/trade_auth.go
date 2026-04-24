package auth

import (
	"errors"
	"fmt"
)

var (
	errAccountAuthenticatorNotAvailable = errors.New("account authenticator not available")
)

// ValidateTradeActionSignature constructs the typed data for a trade action and validates
// the EIP-712 signature using the provided account authenticator. Callers must pass a
// payload that is already validated (e.g. *validation.ValidatedPlaceOrdersAction) so that
// CreateTradeTypedData can render the correct message shape.
func ValidateTradeActionSignature(
	authenticator AccountAuthenticatorInterface,
	config AuthConfig,
	subAccountId SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	action RequestAction,
	validatedPayload any,
	signatureHex string,
	opts *AuthOptions,
) (*AuthResult, error) {
	if authenticator == nil {
		return nil, errAccountAuthenticatorNotAvailable
	}

	typedData, err := CreateTradeTypedData(
		subAccountId,
		nonce,
		expiresAfter,
		action,
		validatedPayload,
		config.DomainName,
		config.DomainVersion,
		config.ChainID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trade typed data: %w", err)
	}

	// Create local options - never mutate the passed-in opts
	localOpts := AuthOptions{
		SupportExpiration: true,
		UseTimestampNonce: false,
	}
	if opts != nil {
		localOpts = *opts
	}

	// Skip nonce check for actions that don't require it (e.g., get* read-only operations)
	if !action.NonceRequired() {
		localOpts.SkipNonceCheck = true
	}

	// Select the appropriate extractor based on the action type
	var extractFunc DataExtractor = ExtractOrderAuthData
	if action == "createSubaccount" {
		// createSubaccount uses masterSubAccountId for ownership verification
		extractFunc = ExtractCreateSubaccountAuthData
	}

	result, err := authenticator.ValidateAccountAuth(
		typedData,
		signatureHex,
		extractFunc,
		&localOpts,
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}
