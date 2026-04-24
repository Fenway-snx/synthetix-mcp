package ratelimiting

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewPerSubAccountRateLimiter(t *testing.T) {

	t.Run("negative general", func(t *testing.T) {

		ctx := context.Background()
		window := time.Minute
		generalRateLimit := RateLimit(-1)
		var specificRateLimits PerSubAccountRateLimits

		r, err := NewPerSubAccountRateLimiter(
			ctx,
			nil,
			window,
			generalRateLimit,
			specificRateLimits,
		)

		assert.Nil(t, r)
		require.NotNil(t, err)

		assert.Contains(t, err.Error(), "rate limits may not be negative")
	})

	t.Run("negative specific", func(t *testing.T) {

		ctx := context.Background()
		window := time.Minute
		generalRateLimit := RateLimit(0)
		specificRateLimits := PerSubAccountRateLimits{
			123: 0,
			456: +1,
			789: -1,
		}

		r, err := NewPerSubAccountRateLimiter(
			ctx,
			nil,
			window,
			generalRateLimit,
			specificRateLimits,
		)

		assert.Nil(t, r)
		require.NotNil(t, err)

		assert.Contains(t, err.Error(), "rate limits may not be negative")
	})
}

func Test_OrderRateLimiter_IN_ACTION_NO_SPECIFICS(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	window := time.Millisecond * 100
	rateLimit := RateLimit(2)

	orl, err := NewPerSubAccountRateLimiter(
		ctx,
		nil,
		window,
		rateLimit,
		nil,
	)

	require.Nil(t, err)
	require.NotNil(t, orl)

	assert.Equal(t, 100*time.Millisecond, orl.Window())

	sid1 := SubAccountId(1)
	sid2 := SubAccountId(2)

	_, exists1 := orl.GetSpecificRateLimit(sid1)
	_, exists2 := orl.GetSpecificRateLimit(sid2)

	assert.False(t, exists1)
	assert.False(t, exists2)

	// Now we have run a couple of cycles ...

	for i := 0; i != 2; i++ {

		// ... and in each cycle we:
		//
		// - "do" the two allowed events; and then
		// - attempt to "do" 10 disallowed events

		var allowed bool
		var count int
		var limit RateLimit

		// "do" first allowed event (SID=1)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 1, count)
		assert.Equal(t, rateLimit, limit)

		// "do" second allowed event (SID=1)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 0, count)
		assert.Equal(t, rateLimit, limit)

		for k := 0; k != 10; k++ {

			allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1+k)

			assert.Nil(t, err)
			assert.False(t, allowed)
			assert.Equal(t, 0, count)
			assert.Equal(t, rateLimit, limit)
		}

		// sleep for the next cycle

		time.Sleep(window)
	}

	cancel()

	time.Sleep(time.Millisecond * 5)
}

func Test_OrderRateLimiter_IN_ACTION_WITH_SPECIFICS(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	window := time.Millisecond * 100
	generalRateLimit := RateLimit(2)
	rateLimit2 := RateLimit(3)

	orl, err := NewPerSubAccountRateLimiter(
		ctx,
		nil,
		window,
		generalRateLimit,
		map[SubAccountId]RateLimit{
			2: rateLimit2,
		},
	)

	require.Nil(t, err)
	require.NotNil(t, orl)

	assert.Equal(t, 100*time.Millisecond, orl.Window())

	sid1 := SubAccountId(1)
	sid2 := SubAccountId(2)

	_, exists1 := orl.GetSpecificRateLimit(sid1)
	rl2, exists2 := orl.GetSpecificRateLimit(sid2)

	assert.False(t, exists1)
	assert.True(t, exists2)
	assert.Equal(t, rateLimit2, rl2)

	// Now we have run a couple of cycles ...

	for i := 0; i != 2; i++ {

		// ... and in each cycle we:
		//
		// - "do" the two / three allowed events; and then
		// - attempt to "do" 10 disallowed events

		var allowed bool
		var count int
		var limit RateLimit

		// "do" first allowed events (SID=1, SID=2)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 1, count)
		assert.Equal(t, generalRateLimit, limit)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 2, count)
		assert.Equal(t, rateLimit2, limit)

		// "do" second allowed event (SID=2, SID=1)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 1, count)
		assert.Equal(t, rateLimit2, limit)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 0, count)
		assert.Equal(t, generalRateLimit, limit)

		// "do" third allowed event (SID=2)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 0, count)
		assert.Equal(t, rateLimit2, limit)

		for k := 0; k != 10; k++ {

			allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1+k)

			assert.Nil(t, err)
			assert.False(t, allowed)
			assert.Equal(t, 0, count)
			assert.Equal(t, generalRateLimit, limit)

			allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1+k)

			assert.Nil(t, err)
			assert.False(t, allowed)
			assert.Equal(t, 0, count)
			assert.Equal(t, rateLimit2, limit)
		}

		// sleep for the next cycle

		time.Sleep(window)
	}

	cancel()

	time.Sleep(time.Millisecond * 5)
}

func Test_OrderRateLimiter_IN_ACTION_WITH_SPECIFICS_AND_DYNAMIC_UPDATING(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	window := time.Millisecond * 100
	generalRateLimit := RateLimit(2)
	rateLimit2 := RateLimit(3)
	rateLimit2b := RateLimit(4)

	orl, err := NewPerSubAccountRateLimiter(
		ctx,
		nil,
		window,
		generalRateLimit,
		map[SubAccountId]RateLimit{},
	)

	require.Nil(t, err)
	require.NotNil(t, orl)

	sid1 := SubAccountId(1)
	sid2 := SubAccountId(2)

	_, exists1 := orl.GetSpecificRateLimit(sid1)
	_, exists2 := orl.GetSpecificRateLimit(sid2)

	assert.False(t, exists1)
	assert.False(t, exists2)

	orl.AddSpecificRateLimit(sid2, rateLimit2b)

	_, exists1 = orl.GetSpecificRateLimit(sid1)
	rl2b, exists2b := orl.GetSpecificRateLimit(sid2)

	assert.False(t, exists1)
	assert.True(t, exists2b)
	assert.Equal(t, rateLimit2b, rl2b)

	_, exists2b = orl.RemoveSpecificRateLimit(sid2)

	assert.True(t, exists2b)

	orl.AddSpecificRateLimit(sid2, rateLimit2)

	// Now we have run a couple of cycles ...

	for i := 0; i != 2; i++ {

		// ... and in each cycle we:
		//
		// - "do" the two / three allowed events; and then
		// - attempt to "do" 10 disallowed events

		var allowed bool
		var count int
		var limit RateLimit

		// "do" first allowed events (SID=1, SID=2)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 1, count)
		assert.Equal(t, generalRateLimit, limit)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 2, count)
		assert.Equal(t, rateLimit2, limit)

		// "do" second allowed event (SID=2, SID=1)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 1, count)
		assert.Equal(t, rateLimit2, limit)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 0, count)
		assert.Equal(t, generalRateLimit, limit)

		// "do" third allowed event (SID=2)

		allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1)

		assert.Nil(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 0, count)
		assert.Equal(t, rateLimit2, limit)

		for k := 0; k != 10; k++ {

			allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid1, 1+k)

			assert.Nil(t, err)
			assert.False(t, allowed)
			assert.Equal(t, 0, count)
			assert.Equal(t, generalRateLimit, limit)

			allowed, count, limit, err = orl.CheckOrderLimit(ctx, sid2, 1+k)

			assert.Nil(t, err)
			assert.False(t, allowed)
			assert.Equal(t, 0, count)
			assert.Equal(t, rateLimit2, limit)
		}

		// sleep for the next cycle

		time.Sleep(window)
	}

	cancel()

	time.Sleep(time.Millisecond * 5)
}
