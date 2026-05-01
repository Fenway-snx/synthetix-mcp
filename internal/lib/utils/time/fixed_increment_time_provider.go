package time

import "time"

type fixedIncrementTimeProvider struct {
	tm time.Time
	d  time.Duration
}

func (fitp *fixedIncrementTimeProvider) Now() time.Time {
	fitp.tm = fitp.tm.Add(fitp.d)

	return fitp.tm
}

// Obtains a time provider that provides incrementing times from the given
// start time, adding the given duration each time `#Now()` is called.
func NewFixedIncrementTimeProvider(
	tm time.Time,
	d time.Duration,
) TimeProvider {
	return &fixedIncrementTimeProvider{
		tm,
		d,
	}
}
