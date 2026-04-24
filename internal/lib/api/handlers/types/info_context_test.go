package types

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func Test_InfoContext_WithServiceState_SetsFields(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := NewInfoContext(
		logger,
		context.Background(),
		nil, nil, nil, nil, nil, nil, nil, nil,
		"req-id",
		"client-req-id",
	)

	assert.Equal(t, "", ctx.ServiceId)
	assert.Nil(t, ctx.HaltChecker)

	halting := func() bool { return true }
	updated := ctx.WithServiceState("my-service", halting)

	assert.Equal(t, "my-service", updated.ServiceId)
	assert.NotNil(t, updated.HaltChecker)
	assert.True(t, updated.HaltChecker())
}

func Test_InfoContext_WithServiceState_DoesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := NewInfoContext(
		logger,
		context.Background(),
		nil, nil, nil, nil, nil, nil, nil, nil,
		"req-id",
		"client-req-id",
	)

	_ = ctx.WithServiceState("changed", func() bool { return true })

	assert.Equal(t, "", ctx.ServiceId)
	assert.Nil(t, ctx.HaltChecker)
}

func Test_InfoContext_WithServiceState_NilHaltChecker(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := NewInfoContext(
		logger,
		context.Background(),
		nil, nil, nil, nil, nil, nil, nil, nil,
		"req-id",
		"client-req-id",
	)

	updated := ctx.WithServiceState("svc", nil)

	assert.Equal(t, "svc", updated.ServiceId)
	assert.Nil(t, updated.HaltChecker)
}

func Test_InfoContext_WithServiceState_PreservesOtherFields(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := NewInfoContext(
		logger,
		context.Background(),
		nil, nil, nil, nil, nil, nil, nil, nil,
		"req-id",
		"client-req-id",
	)

	updated := ctx.WithServiceState("svc", func() bool { return false })

	assert.Equal(t, RequestId("req-id"), updated.RequestId)
	assert.Equal(t, ClientRequestId("client-req-id"), updated.ClientRequestId)
	assert.NotNil(t, updated.Logger)
}
