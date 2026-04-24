package time

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Now_PROVIDES_UTC(t *testing.T) {

	now := Now()

	// string forms
	{
		expected := time.UTC.String()
		actual := now.Location().String()

		assert.Equal(t, expected, actual)
		assert.Equal(t, expected, actual)
	}

	// instances (via pointers)
	{
		expected := time.UTC
		actual := now.Location()

		assert.Equal(t, expected, actual)
		assert.Equal(t, expected, actual)
	}
}

func Test_NowMicrosAndMillis(t *testing.T) {

	tm_before := time.Now()

	runtime.Gosched()

	microseconds, milliseconds := NowMicrosAndMillis()

	runtime.Gosched()

	tm_after := time.Now()

	assert.LessOrEqual(t, tm_before.UnixMicro(), microseconds)
	assert.LessOrEqual(t, tm_before.UnixMilli(), milliseconds)

	assert.Equal(t, milliseconds, microseconds/1_000)

	assert.GreaterOrEqual(t, tm_after.UnixMicro(), microseconds)
	assert.GreaterOrEqual(t, tm_after.UnixMilli(), milliseconds)
}

func Test_SetTimeProvider(t *testing.T) {
	// Capture the original time before we change anything
	originalNow := Now()

	// Set a fixed time provider
	fixedTime := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	cleanup := SetTimeProvider(NewFixedTimeProvider(fixedTime))

	// Now() should return the fixed time
	assert.Equal(t, fixedTime, Now())
	assert.NotEqual(t, originalNow, Now())

	// Call cleanup to restore original provider
	cleanup()

	// Now() should return real time again (close to current time)
	restoredNow := Now()
	timeDiff := time.Since(restoredNow)
	assert.Less(t, timeDiff.Abs(), time.Second, "After cleanup, Now() should return real time")
}

func Test_SetTimeProvider_NestedCleanup(t *testing.T) {
	// Test that nested SetTimeProvider calls restore correctly
	time1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	cleanup1 := SetTimeProvider(NewFixedTimeProvider(time1))
	assert.Equal(t, time1, Now())

	cleanup2 := SetTimeProvider(NewFixedTimeProvider(time2))
	assert.Equal(t, time2, Now())

	// Cleanup in reverse order
	cleanup2()
	assert.Equal(t, time1, Now(), "Should restore to first fixed time")

	cleanup1()
	// Should be back to real time
	restoredNow := Now()
	timeDiff := time.Since(restoredNow)
	assert.Less(t, timeDiff.Abs(), time.Second, "Should restore to real time")
}
