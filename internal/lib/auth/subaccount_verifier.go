package auth

import "context"

// AuthType is a Go-native enum for the result of a subaccount-authorization
// lookup. The numeric values are deliberately independent of any wire
// format; adapters in lib/auth/authgrpc and lib/auth/authtest translate
// explicitly between this enum and their underlying transport values, so
// lib/auth stays free of google.golang.org/grpc and protobuf.
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
// hands to a SubaccountVerifier. Adapters at the service boundary translate
// this into whatever wire format their backend speaks (gRPC today, REST in
// mcp-service, etc.).
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

// SubaccountVerifier is the contract the Authenticator depends on. Replacing
// the previous gRPC-typed dependency with this Go-native interface lets
// lib/auth stay free of google.golang.org/grpc; adapters live in
// lib/auth/authtest (for tests) and at each service's main() (for prod).
type SubaccountVerifier interface {
	VerifySubaccountAuthorization(
		ctx context.Context,
		req VerifySubaccountAuthorizationRequest,
	) (VerifySubaccountAuthorizationResponse, error)
}
