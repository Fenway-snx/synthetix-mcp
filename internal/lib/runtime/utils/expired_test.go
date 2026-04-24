package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Catches a deadline that has already passed regardless of whether the runtime
// timer goroutine has fired.
func Test_Expired_pastDeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // ensure deadline has passed

	assert.True(t, Expired(ctx), "must detect a past deadline")
}

func Test_Expired_healthy(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assert.False(t, Expired(ctx))
}

// Returns false for an explicitly cancelled context (not deadline exceeded), so
// that shutdown cancellation is not counted as a timeout.
func Test_Expired_cancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, Expired(ctx))
}

func Test_Expired_noDeadline(t *testing.T) {
	t.Parallel()

	assert.False(t, Expired(context.Background()))
}
