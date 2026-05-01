package time

import "time"

// Returns elapsed time using the configured time provider.
func Since(tm time.Time) time.Duration {

	return Now().Sub(tm)
}
