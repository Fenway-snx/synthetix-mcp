package time

import (
	"math/rand/v2"
	"time"
)

// Obtains an offset of duration from the current time.
func NowPlusDuration(duration time.Duration) time.Time {
	return Now().Add(duration)
}

// Obtains a random duration in the half-open range [min, exclusiveMax).
//
// Preconditions:
//   - `min < exclusiveMax`
//
// Note:
// The durations are evaluated as nanoseconds internally.
func RandomDurationInRange(min, exclusiveMax time.Duration) time.Duration {

	min_ms := min.Nanoseconds()
	xmax_ms := exclusiveMax.Nanoseconds()

	n := xmax_ms - min_ms

	r := rand.Int64N(n)

	return min + time.Duration(r)*time.Nanosecond
}
