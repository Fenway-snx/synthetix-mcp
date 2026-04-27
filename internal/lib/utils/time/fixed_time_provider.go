package time

import "time"

type fixedTimeProvider struct {
	t time.Time
}

func (ftp fixedTimeProvider) Now() time.Time {
	return ftp.t
}

// Obtains a time provider that provides a preprogrammed (unchanging) time
// (in UTC).
func NewFixedTimeProvider(t time.Time) TimeProvider {

	return &fixedTimeProvider{
		t,
	}
}
