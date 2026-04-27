package time

import (
	"time"
)

// Determines whether a given time value tm lies inside the exclusive time
// range [tmFrom, tmTo).
//
// Preconditions:
//   - `tmFrom <= tmTo`;
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

// Determines whether a given time value tm lies inside the inclusive time
// range [tmFrom, tmTo].
//
// Preconditions:
//   - `tmFrom <= tmTo`;
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
