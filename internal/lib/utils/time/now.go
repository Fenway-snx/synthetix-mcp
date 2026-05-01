package time

import "time"

var (
	_timeProvider TimeProvider
)

func init() {
	_timeProvider = NewRealTimeProvider()
}

// SetTimeProvider sets the time provider for testing purposes.
// Returns a cleanup function that restores the original provider.
func SetTimeProvider(tp TimeProvider) func() {
	original := _timeProvider
	_timeProvider = tp
	return func() {
		_timeProvider = original
	}
}

// Obtains the current time, in UTC.
func Now() time.Time {
	return _timeProvider.Now()
}

// Returns current UTC time as microseconds and milliseconds.
func NowMicrosAndMillis() (microseconds, milliseconds int64) {
	t := Now()

	microseconds = t.UnixMicro()
	milliseconds = t.UnixMilli()

	return
}
