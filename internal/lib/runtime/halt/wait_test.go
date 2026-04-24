package halt

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_WaitUntil_COMPLETES_WHEN_CHECK_RETURNS_DONE(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	err := WaitUntil(context.Background(), 10*time.Millisecond, func() (bool, error) {
		return calls.Add(1) >= 2, nil
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.GreaterOrEqual(t, calls.Load(), int64(2))
}

func Test_WaitUntil_RETURNS_CONTEXT_ERROR(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := WaitUntil(ctx, 10*time.Millisecond, func() (bool, error) {
		return false, nil
	})
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func Test_WaitUntil_RETURNS_CHECK_ERROR(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("boom")
	err := WaitUntil(context.Background(), 10*time.Millisecond, func() (bool, error) {
		return false, expectedErr
	})
	require.Error(t, err)
	require.ErrorIs(t, err, expectedErr)
}
