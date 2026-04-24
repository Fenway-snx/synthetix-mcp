package time

import "time"

type randomTimeProvider struct {
	tm    time.Time
	dFrom time.Duration
	dTo   time.Duration
}

func (rtp randomTimeProvider) Now() time.Time {
	d := RandomDurationInRange(rtp.dFrom, rtp.dTo)

	return rtp.tm.Add(d)
}

// Obtains a time provider that provides random times within a duration
// range relative to a base time.
//
// Preconditions:
//   - `dFrom < dTo`;
func NewRandomTimeProvider(
	tm time.Time,
	dFrom time.Duration,
	dTo time.Duration,
) TimeProvider {
	return &randomTimeProvider{
		tm:    tm,
		dFrom: dFrom,
		dTo:   dTo,
	}
}
