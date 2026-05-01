package time

import (
	"time"
)

// Reports whether a time lies in the half-open range [from, to).
func TimeWithinRangeExclusiveEnd(
	tm time.Time,
	tmFrom time.Time,
	tmTo time.Time,
) bool {

	ns := tm.UnixNano()
	nsFrom := tmFrom.UnixNano()
	nsTo := tmTo.UnixNano()

	if ns < nsFrom {
		return false
	}

	if ns >= nsTo {
		return false
	}

	return true
}

// Reports whether a time lies in the inclusive range [from, to].
func TimeWithinRangeInclusiveEnd(
	tm time.Time,
	tmFrom time.Time,
	tmTo time.Time,
) bool {

	ns := tm.UnixNano()
	nsFrom := tmFrom.UnixNano()
	nsTo := tmTo.UnixNano()

	if ns < nsFrom {
		return false
	}

	if ns > nsTo {
		return false
	}

	return true
}
