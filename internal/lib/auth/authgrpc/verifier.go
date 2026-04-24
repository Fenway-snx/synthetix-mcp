// Package authgrpc adapts the v4 SubaccountServiceClient to the grpc-free
// snx_lib_auth.SubaccountVerifier interface that lib/auth's Authenticator now
// consumes. Production services that talk to the subaccount service over gRPC
// import this package; lib/auth itself stays free of google.golang.org/grpc.
package authgrpc

import (
	"context"

	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// NewVerifier wraps a SubaccountServiceClient so it satisfies
// snx_lib_auth.SubaccountVerifier. Pass the result directly to
// snx_lib_auth.NewAuthenticator.
func NewVerifier(client v4grpc.SubaccountServiceClient) snx_lib_auth.SubaccountVerifier {
	if client == nil {
		return nil
	}
	return &verifier{client: client}
}

type verifier struct {
	client v4grpc.SubaccountServiceClient
}

func (v *verifier) VerifySubaccountAuthorization(
	ctx context.Context,
	req snx_lib_auth.VerifySubaccountAuthorizationRequest,
) (snx_lib_auth.VerifySubaccountAuthorizationResponse, error) {
	resp, err := v.client.VerifySubaccountAuthorization(ctx, &v4grpc.VerifySubaccountAuthorizationRequest{
		TimestampMs:  req.TimestampMs,
		TimestampUs:  req.TimestampUs,
		SubAccountId: req.SubAccountID,
		Address:      req.Address,
		Permissions:  req.Permissions,
	})
	if err != nil {
		return snx_lib_auth.VerifySubaccountAuthorizationResponse{}, err
	}
	return snx_lib_auth.VerifySubaccountAuthorizationResponse{
		IsAuthorized:      resp.GetIsAuthorized(),
		AuthorizationType: snx_lib_auth.AuthType(resp.GetAuthorizationType()),
	}, nil
}
