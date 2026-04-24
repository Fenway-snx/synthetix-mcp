package snaxpot

import "time"

const EpochDuration = 7 * 24 * time.Hour

// EpochForTime returns the 1-based epoch ID that contains t, given a
// configured anchor that marks the start of epoch 1.
func EpochForTime(
	t time.Time,
	anchor time.Time,
) uint64 {
	if t.Before(anchor) {
		return 0
	}

	return uint64(t.Sub(anchor)/EpochDuration) + 1
}

// EpochBounds returns the half-open [start, end) bounds for epochID.
func EpochBounds(
	epochID uint64,
	anchor time.Time,
) (start time.Time, end time.Time) {
	if epochID == 0 {
		start, end = anchor, anchor
		return
	}

	start = anchor.Add(time.Duration(epochID-1) * EpochDuration)
	end = start.Add(EpochDuration)

	return
}

// NextCutoffAfter returns the next Sunday 18:00:00 UTC strictly after now.
func NextCutoffAfter(now time.Time) time.Time {
	now = now.UTC()

	daysUntilSunday := (int(time.Sunday) - int(now.Weekday()) + 7) % 7
	candidate := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+daysUntilSunday,
		18,
		0,
		0,
		0,
		time.UTC,
	)
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}

	return candidate
}
