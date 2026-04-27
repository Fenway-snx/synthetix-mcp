package time

import "time"

type realTimeProvider struct{}

var (
	_realTimeProvider = realTimeProvider{}
)

func (realTimeProvider) Now() time.Time {
	return time.Now().UTC()
}

// Obtains a time provider that provides the current time (in UTC).
//
// Note:
// The current implementation always returns a reference to the same
// instance. This is subject to change in the future, and callers should not
// rely on that.
func NewRealTimeProvider() TimeProvider {
	return &_realTimeProvider
}
