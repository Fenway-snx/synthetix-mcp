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

// Takes the current time (in UTC) and obtains its value as microseconds and
// milliseconds, respectively.
//
// The purpose of this function is to provide, in a single (compound)
// assignment, the values to be used for our transport-layer timestamps on
// all request and response messages in the Core. Once we move over to use
// the core `Timestamp` type, this function will become obsolete.
//
// Postconditions:
// - `milliseconds == (microseconds / 1_000)`;
func NowMicrosAndMillis() (microseconds, milliseconds int64) {
	t := Now()

	microseconds = t.UnixMicro()
	milliseconds = t.UnixMilli()

	return
}
