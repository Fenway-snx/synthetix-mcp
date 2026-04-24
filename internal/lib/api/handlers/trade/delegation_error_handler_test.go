package trade

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

// Documented client-facing behavior: for the three 400-class gRPC codes
// (AlreadyExists, InvalidArgument, FailedPrecondition), `handleGRPCError`
// only forwards an ErrorInfo Reason/Metadata pair when the Reason matches a
// registered ErrorCode in lib/core/status_codes. Unregistered reasons (and
// any metadata attached alongside them) are discarded so upstream gRPC
// services cannot inject arbitrary codes or leak internal metadata through
// the API boundary.

func statusErrorWithReason(
	t *testing.T,
	code codes.Code,
	message string,
	reason string,
	metadata map[string]string,
) error {
	t.Helper()
	st := status.New(code, message)
	detailed, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason:   reason,
		Metadata: metadata,
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	return detailed.Err()
}

func Test_handleGRPCError_FailedPrecondition_RegisteredReason_ForwardsMetadata(t *testing.T) {
	metadata := map[string]string{
		"currentCount": "5",
		"maxAllowed":   "5",
		"tierName":     "Tier 1",
	}
	err := statusErrorWithReason(
		t,
		codes.FailedPrecondition,
		"sub-account limit exceeded",
		string(snx_lib_api_json.ErrorCodeMaxSubAccountsExceeded),
		metadata,
	)

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeMaxSubAccountsExceeded, resp.Error.Code)
	assert.Equal(t, metadata, resp.Error.Details)
}

func Test_handleGRPCError_FailedPrecondition_UnregisteredReason_DropsDetails(t *testing.T) {
	err := statusErrorWithReason(
		t,
		codes.FailedPrecondition,
		"failed precondition",
		"internal.db.connection_lost",
		map[string]string{
			"dbHost":    "pg-subaccount-01.internal",
			"requestId": "abc-123",
		},
	)

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code,
		"unregistered reason must not become a client-facing ErrorCode",
	)
	assert.Nil(t, resp.Error.Details,
		"metadata attached to an unregistered reason must not leak to clients",
	)
}

func Test_handleGRPCError_InvalidArgument_RegisteredReason_ForwardsMetadata(t *testing.T) {
	metadata := map[string]string{"field": "permissions"}
	err := statusErrorWithReason(
		t,
		codes.InvalidArgument,
		"invalid input",
		string(snx_lib_api_json.ErrorCodeInvalidValue),
		metadata,
	)

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInvalidValue, resp.Error.Code)
	assert.Equal(t, metadata, resp.Error.Details)
}

func Test_handleGRPCError_InvalidArgument_UnregisteredReason_DropsDetails(t *testing.T) {
	err := statusErrorWithReason(
		t,
		codes.InvalidArgument,
		"invalid input",
		"subaccount-service/validator",
		map[string]string{"serviceName": "subaccount"},
	)

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
	assert.Nil(t, resp.Error.Details)
}

func Test_handleGRPCError_AlreadyExists_NoErrorInfo_UsesDefaultCode(t *testing.T) {
	err := status.Error(codes.AlreadyExists, "already exists")

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
	assert.Nil(t, resp.Error.Details)
}

func Test_handleGRPCError_AlreadyExists_UnregisteredReason_DropsDetails(t *testing.T) {
	err := statusErrorWithReason(
		t,
		codes.AlreadyExists,
		"conflict",
		"DUPLICATE_ROW",
		map[string]string{"table": "delegations", "pk": "42"},
	)

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code,
		"server-supplied DUPLICATE_ROW is not a registered code",
	)
	assert.Nil(t, resp.Error.Details,
		"internal table/pk metadata must not leak",
	)
}

func Test_handleGRPCError_NotFound(t *testing.T) {
	err := status.Error(codes.NotFound, "delegation not found")

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_404_NotFound, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeNotFound, resp.Error.Code)
}

func Test_handleGRPCError_NotFound_IgnoresErrorInfo(t *testing.T) {
	err := statusErrorWithReason(
		t,
		codes.NotFound,
		"not found",
		string(snx_lib_api_json.ErrorCodeOrderNotFound),
		map[string]string{"tenant": "internal-ops"},
	)

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_404_NotFound, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeNotFound, resp.Error.Code,
		"NotFound always returns the canonical ErrorCodeNotFound",
	)
	assert.Nil(t, resp.Error.Details,
		"NotFound does not forward details",
	)
}

func Test_handleGRPCError_PermissionDenied(t *testing.T) {
	err := status.Error(codes.PermissionDenied, "forbidden")

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_403_Forbidden, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeForbidden, resp.Error.Code)
}

func Test_handleGRPCError_Unauthenticated(t *testing.T) {
	err := status.Error(codes.Unauthenticated, "unauthenticated")

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_401_Unauthorized, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeUnauthorized, resp.Error.Code)
}

func Test_handleGRPCError_Internal(t *testing.T) {
	err := status.Error(codes.Internal, "database error")

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_500_InternalServerError, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
}

func Test_handleGRPCError_Unimplemented(t *testing.T) {
	err := status.Error(codes.Unimplemented, "not implemented")

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_501_StatusNotImplemented, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
}

func Test_handleGRPCError_NonStatusError(t *testing.T) {
	err := errors.New("not a gRPC status error")

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_500_InternalServerError, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
}

func Test_handleGRPCError_EmptyReason_DropsDetails(t *testing.T) {
	err := statusErrorWithReason(
		t,
		codes.InvalidArgument,
		"invalid",
		"",
		map[string]string{"internalField": "internal-value"},
	)

	httpStatus, resp := handleGRPCError(err, "req-id")

	assert.Equal(t, HTTPStatusCode_400_BadRequest, httpStatus)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
	assert.Nil(t, resp.Error.Details,
		"an empty reason string is not a registered ErrorCode",
	)
}
