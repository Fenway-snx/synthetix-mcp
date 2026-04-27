package auth

import "context"

// AuthType is a Go-native enum for the result of a subaccount-authorization
// lookup. The numeric values are deliberately independent of any wire format.
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

// VerifySubaccountAuthorizationRequest is the native request the Authenticator
// hands to a SubaccountVerifier.
type VerifySubaccountAuthorizationRequest struct {
	TimestampMs  int64
	TimestampUs  int64
	SubAccountID int64
	Address      string
	Permissions  []string
}

// VerifySubaccountAuthorizationResponse is the native response.
type VerifySubaccountAuthorizationResponse struct {
	IsAuthorized      bool
	AuthorizationType AuthType
}

// SubaccountVerifier is the contract the Authenticator depends on.
type SubaccountVerifier interface {
	VerifySubaccountAuthorization(
		ctx context.Context,
		req VerifySubaccountAuthorizationRequest,
	) (VerifySubaccountAuthorizationResponse, error)
}
