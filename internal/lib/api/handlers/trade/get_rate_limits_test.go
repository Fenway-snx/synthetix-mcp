package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func testTradeContextForGetRateLimits(t *testing.T) snx_lib_api_handlers_types.TradeContext {
	t.Helper()

	logger := snx_lib_logging_doubles.NewStubLogger()
	return snx_lib_api_handlers_types.NewTradeContext(
		logger,
		t.Context(),
		nil,
		nil,
		nil,
		nil,
		snx_lib_authtest.NewMockSubaccountServiceClient(),
		nil,
		nil,
		nil,
		"req-id",
		"client-req",
		snx_lib_api_handlers_types.WalletAddress("0xwallet"),
		snx_lib_core.SubAccountId(1),
	)
}

func Test_Handle_getRateLimits_RateLimitResponse_FROM_SUBACCOUNT_SNAPSHOT(t *testing.T) {
	t.Parallel()

	params := HandlerParams{"user": "0xabc", "type": "getRateLimits"}

	tests := []struct {
		name             string
		snapshot         *snx_lib_api_handlers_types.GetRateLimitsSubaccountSnapshot
		wantRequestsUsed int
		wantRequestsCap  int
	}{
		{
			name:             "NIL_SNAPSHOT_ZEROS",
			snapshot:         nil,
			wantRequestsUsed: 0,
			wantRequestsCap:  0,
		},
		{
			name: "TYPICAL_AFTER_DEBIT",
			snapshot: &snx_lib_api_handlers_types.GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 1155,
				Limit:           1200,
			},
			wantRequestsUsed: 45,
			wantRequestsCap:  1200,
		},
		{
			name: "FULL_BUCKET",
			snapshot: &snx_lib_api_handlers_types.GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 100,
				Limit:           100,
			},
			wantRequestsUsed: 0,
			wantRequestsCap:  100,
		},
		{
			name: "CLAMPS_NEGATIVE_USED",
			snapshot: &snx_lib_api_handlers_types.GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 15,
				Limit:           10,
			},
			wantRequestsUsed: 0,
			wantRequestsCap:  10,
		},
		{
			name: "ZERO_LIMIT_FROM_LIMITER",
			snapshot: &snx_lib_api_handlers_types.GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 0,
				Limit:           0,
			},
			wantRequestsUsed: 0,
			wantRequestsCap:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := testTradeContextForGetRateLimits(t)
			if tt.snapshot != nil {
				ctx = ctx.WithGetRateLimitsSubaccountSnapshot(tt.snapshot)
			}

			code, resp := Handle_getRateLimits(ctx, params)
			assert.Equal(t, HTTPStatusCode_200_OK, code)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Response)

			parsed, ok := resp.Response.(RateLimitResponse)
			require.True(t, ok)
			assert.Equal(t, tt.wantRequestsUsed, parsed.RequestsUsed)
			assert.Equal(t, tt.wantRequestsCap, parsed.RequestsCap)
		})
	}
}
