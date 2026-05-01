package auth

import "context"

// Go-native enum for subaccount authorization lookup results.
type AuthType uint8

const (
	AuthTypeUnspecified AuthType = 0
	AuthTypeNone        AuthType = 1
	AuthTypeOwner       AuthType = 2
	AuthTypeDelegate    AuthType = 3
)

func (t AuthType) String() string {
	switch t {
	case AuthTypeOwner:
		return "AUTHORIZATION_TYPE_OWNER"
	case AuthTypeDelegate:
		return "AUTHORIZATION_TYPE_DELEGATE"
	case AuthTypeNone:
		return "AUTHORIZATION_TYPE_NONE"
	default:
		return "AUTHORIZATION_TYPE_UNSPECIFIED"
	}
}

// Native request for subaccount authorization checks.
type VerifySubaccountAuthorizationRequest struct {
	TimestampMs  int64
	TimestampUs  int64
	SubAccountID int64
	Address      string
	Permissions  []string
}

// Native response for subaccount authorization checks.
type VerifySubaccountAuthorizationResponse struct {
	IsAuthorized      bool
	AuthorizationType AuthType
}

// Contract for subaccount authorization checks.
type SubaccountVerifier interface {
	VerifySubaccountAuthorization(
		ctx context.Context,
		req VerifySubaccountAuthorizationRequest,
	) (VerifySubaccountAuthorizationResponse, error)
}
