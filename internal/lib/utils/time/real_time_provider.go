package time

import "time"

type realTimeProvider struct{}

var (
	_realTimeProvider = realTimeProvider{}
)

func (realTimeProvider) Now() time.Time {
	return time.Now().UTC()
}

// Returns a provider for current UTC time.
func NewRealTimeProvider() TimeProvider {
	return &_realTimeProvider
}
