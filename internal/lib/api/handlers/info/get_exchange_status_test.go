package info

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_exchange_status "github.com/Fenway-snx/synthetix-mcp/internal/lib/exchange/status"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func Test_Handle_getExchangeStatus_NotHalting_ReturnsRunning(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := InfoContext{
		ContextCommon: ContextCommon{
			Context: context.Background(),
			Logger:  logger,
		},
		ClientRequestId: "req-1",
	}
	ctx = ctx.WithServiceState("test-svc", func() bool { return false })

	status, resp := Handle_getExchangeStatus(ctx, HandlerParams{})

	assert.Equal(t, HTTPStatusCode_200_OK, status)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)

	dto, ok := resp.Response.(snx_lib_exchange_status.ExchangeStatus)
	require.True(t, ok)
	assert.True(t, dto.AcceptingOrders)
	assert.Equal(t, "RUNNING", dto.ExchangeStatus)
	assert.Greater(t, dto.TimestampMs, int64(0))
}

func Test_Handle_getExchangeStatus_Halting_ReturnsMaintenance(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := InfoContext{
		ContextCommon: ContextCommon{
			Context: context.Background(),
			Logger:  logger,
		},
		ClientRequestId: "req-2",
	}
	ctx = ctx.WithServiceState("test-svc", func() bool { return true })

	status, resp := Handle_getExchangeStatus(ctx, HandlerParams{})

	assert.Equal(t, HTTPStatusCode_200_OK, status)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)

	dto, ok := resp.Response.(snx_lib_exchange_status.ExchangeStatus)
	require.True(t, ok)
	assert.False(t, dto.AcceptingOrders)
	assert.Equal(t, "MAINTENANCE", dto.ExchangeStatus)
	assert.Equal(t, "SERVICE_DRAINING", dto.Code)
}

func Test_Handle_getExchangeStatus_NilHaltChecker_ReturnsRunning(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := InfoContext{
		ContextCommon: ContextCommon{
			Context: context.Background(),
			Logger:  logger,
		},
		ClientRequestId: "req-3",
	}

	status, resp := Handle_getExchangeStatus(ctx, HandlerParams{})

	assert.Equal(t, HTTPStatusCode_200_OK, status)
	require.NotNil(t, resp)

	dto, ok := resp.Response.(snx_lib_exchange_status.ExchangeStatus)
	require.True(t, ok)
	assert.True(t, dto.AcceptingOrders)
	assert.Equal(t, "RUNNING", dto.ExchangeStatus)
}

func Test_Handle_getExchangeStatus_EchoesClientRequestId(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := InfoContext{
		ContextCommon: ContextCommon{
			Context: context.Background(),
			Logger:  logger,
		},
		ClientRequestId: "client-req-xyz",
	}

	_, resp := Handle_getExchangeStatus(ctx, HandlerParams{})

	assert.Equal(t, ClientRequestId("client-req-xyz"), resp.ClientRequestId)
}

func Test_Handle_getExchangeStatus_IgnoresParams(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := InfoContext{
		ContextCommon: ContextCommon{
			Context: context.Background(),
			Logger:  logger,
		},
	}

	status, resp := Handle_getExchangeStatus(ctx, HandlerParams{
		"extra_field": "should_be_ignored",
		"action":      "getExchangeStatus",
	})

	assert.Equal(t, HTTPStatusCode_200_OK, status)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)
}
