package info

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_whitelist "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/whitelist"
	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

// --- test helpers -----------------------------------------------------------

const testWhitelistedAddr = "0xaabbccddeeff00112233445566778899aabbccdd"

func newTestArbitrator(t *testing.T, permissions snx_lib_api_whitelist.PermissionsMap) *snx_lib_api_whitelist.WhitelistArbitrator {
	t.Helper()
	a, err := snx_lib_api_whitelist.NewWhitelistArbitrator(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		&snx_lib_db_redis.SnxClient{},
		"ignored",
		nil,
		permissions,
		time.Minute,
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	return a
}

func newTestInfoCtx(t *testing.T, arbitrator *snx_lib_api_whitelist.WhitelistArbitrator) InfoContext {
	t.Helper()
	return snx_lib_api_handlers_types.NewInfoContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		arbitrator,
		snx_lib_request.NewRequestID(),
		"req-id",
	)
}

// --- tests ------------------------------------------------------------------

func Test_HandleGetIsWhitelisted_NormalizesAddress(t *testing.T) {
	t.Parallel()

	mixedCaseAddr := "0xAaBbCcDdEeFf00112233445566778899AaBbCcDd"

	arbitrator := newTestArbitrator(t, snx_lib_api_whitelist.PermissionsMap{
		snx_lib_api_whitelist.WalletAddress(testWhitelistedAddr): true,
	})
	ic := newTestInfoCtx(t, arbitrator)

	status, resp := Handle_getIsWhitelisted(ic, map[string]any{
		"walletAddress": mixedCaseAddr,
	})

	require.NotNil(t, resp)
	assert.Equal(t, HTTPStatusCode_200_OK, status)
	assert.Equal(t, "ok", resp.Status)

	allowed, ok := resp.Response.(bool)
	require.True(t, ok, "response payload should be a boolean")
	assert.True(t, allowed)
}

func Test_HandleGetIsWhitelisted_RejectsInvalidParams(t *testing.T) {
	t.Parallel()

	arbitrator := newTestArbitrator(t, snx_lib_api_whitelist.PermissionsMap{
		snx_lib_api_whitelist.WalletAddress(testWhitelistedAddr): true,
	})
	ic := newTestInfoCtx(t, arbitrator)

	tests := []struct {
		name   string
		params HandlerParams
	}{
		{"missing walletAddress key", map[string]any{}},
		{"empty walletAddress", map[string]any{"walletAddress": ""}},
		{"walletAddress is integer", map[string]any{"walletAddress": 12345}},
		{"walletAddress is boolean", map[string]any{"walletAddress": true}},
		{"walletAddress is nil", map[string]any{"walletAddress": nil}},
		{"walletAddress is nested object", map[string]any{"walletAddress": map[string]any{"addr": "0x1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, resp := Handle_getIsWhitelisted(ic, tt.params)

			require.NotNil(t, resp)
			assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
			assert.Equal(t, "error", resp.Status)
			require.NotNil(t, resp.Error)
			assert.Equal(t, snx_lib_api_json.ErrorCodeInvalidFormat, resp.Error.Code)
			assert.Equal(t, "Invalid request body", resp.Error.Message)
		})
	}
}

func Test_HandleGetIsWhitelisted_RejectsInvalidHexAddress(t *testing.T) {
	t.Parallel()

	arbitrator := newTestArbitrator(t, snx_lib_api_whitelist.PermissionsMap{
		snx_lib_api_whitelist.WalletAddress(testWhitelistedAddr): true,
	})
	ic := newTestInfoCtx(t, arbitrator)

	tests := []struct {
		name          string
		walletAddress string
	}{
		{"plaintext string", "not-an-address"},
		{"too short with prefix", "0xaabb"},
		{"contains non-hex characters", "0xGGGGccddeeff00112233445566778899aabbccdd"},
		{"only whitespace", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, resp := Handle_getIsWhitelisted(ic, map[string]any{
				"walletAddress": tt.walletAddress,
			})

			require.NotNil(t, resp)
			assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
			assert.Equal(t, "error", resp.Status)
			require.NotNil(t, resp.Error)
			assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
			assert.Equal(t, "Invalid wallet address format", resp.Error.Message)
		})
	}
}

func Test_HandleGetIsWhitelisted_NilArbitrator(t *testing.T) {
	t.Parallel()

	ic := newTestInfoCtx(t, nil)

	status, resp := Handle_getIsWhitelisted(ic, map[string]any{
		"walletAddress": testWhitelistedAddr,
	})

	require.NotNil(t, resp)
	assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, snx_lib_api_json.ErrorCodeInternalError, resp.Error.Code)
}

func Test_HandleGetIsWhitelisted_AddressNotWhitelisted(t *testing.T) {
	t.Parallel()

	unknownAddr := "0x1111111111111111111111111111111111111111"

	arbitrator := newTestArbitrator(t, snx_lib_api_whitelist.PermissionsMap{
		snx_lib_api_whitelist.WalletAddress(testWhitelistedAddr): true,
	})
	ic := newTestInfoCtx(t, arbitrator)

	status, resp := Handle_getIsWhitelisted(ic, map[string]any{
		"walletAddress": unknownAddr,
	})

	require.NotNil(t, resp)
	assert.Equal(t, HTTPStatusCode_200_OK, status)
	assert.Equal(t, "ok", resp.Status)

	allowed, ok := resp.Response.(bool)
	require.True(t, ok, "response payload should be a boolean")
	assert.False(t, allowed)
}
