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

// Returns a provider for random times in a duration range from a base time.
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
