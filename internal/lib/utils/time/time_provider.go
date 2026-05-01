package time

import "time"

// Provides current time, with test implementations for determinism.
type TimeProvider interface {
	Now() time.Time
}
