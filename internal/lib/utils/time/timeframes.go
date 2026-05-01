package time

import (
	"fmt"
	"time"
)

type Timeframe string

// Timeframe constants
const (
	Timeframe1Minute   Timeframe = "1m"  // 1 minute
	Timeframe5Minutes  Timeframe = "5m"  // 5 minutes
	Timeframe15Minutes Timeframe = "15m" // 15 minutes
	Timeframe30Minutes Timeframe = "30m" // 30 minutes
	Timeframe1Hour     Timeframe = "1h"  // 1 hour
	Timeframe4Hours    Timeframe = "4h"  // 4 hours
	Timeframe8Hours    Timeframe = "8h"  // 8 hours
	Timeframe12Hours   Timeframe = "12h" // 12 hours
	Timeframe1Day      Timeframe = "1d"  // 1 day
	Timeframe3Days     Timeframe = "3d"  // 3 days
	Timeframe1Week     Timeframe = "1w"  // 1 week
	Timeframe1Month    Timeframe = "1M"  // 1 month
	Timeframe3Months   Timeframe = "3M"  // 3 months
)

// SupportedTimeframes defines the valid timeframe values
var SupportedTimeframes = []Timeframe{
	Timeframe1Minute,
	Timeframe5Minutes,
	Timeframe15Minutes,
	Timeframe30Minutes,
	Timeframe1Hour,
	Timeframe4Hours,
	Timeframe8Hours,
	Timeframe12Hours,
	Timeframe1Day,
	Timeframe3Days,
	Timeframe1Week,
	Timeframe1Month,
	Timeframe3Months,
}

// Provides O(1) membership tests for supported timeframes. Built
// from SupportedTimeframes at init time.
var supportedSet map[Timeframe]struct{}

// Maps each non-base timeframe to the shortest base timeframe
// from which it can be aggregated. Timeframes absent from this map
// are themselves base timeframes (i.e. self-referential).
var baseTimeframes = map[Timeframe]Timeframe{
	Timeframe30Minutes: Timeframe15Minutes,
	Timeframe8Hours:    Timeframe4Hours,
	Timeframe12Hours:   Timeframe4Hours,
	Timeframe3Days:     Timeframe1Day,
	Timeframe1Week:     Timeframe1Day,
	Timeframe1Month:    Timeframe1Day,
	Timeframe3Months:   Timeframe1Day,
}

func init() {
	supportedSet = make(map[Timeframe]struct{}, len(SupportedTimeframes))
	for _, tf := range SupportedTimeframes {
		supportedSet[tf] = struct{}{}
	}
}

// Checks whether a timeframe is a recognised supported value.
func (tf Timeframe) IsSupported() bool {
	_, ok := supportedSet[tf]
	return ok
}

// Returns the base timeframe from which tf can be aggregated. Base
// timeframes return themselves. Unsupported timeframes return the
// zero value and false.
func (tf Timeframe) Base() (base Timeframe, ok bool) {
	if b, derived := baseTimeframes[tf]; derived {
		base = b
		ok = true
		return
	}

	if tf.IsSupported() {
		base = tf
		ok = true
		return
	}

	return
}

// Reports whether the timeframe requires aggregation from a shorter
// base timeframe.
func (tf Timeframe) IsDerived() bool {
	_, derived := baseTimeframes[tf]
	return derived
}

// ParseTimeframe checks if a timeframe is supported
func ParseTimeframe(timeframe string) (Timeframe, error) {
	tf := Timeframe(timeframe)
	if tf.IsSupported() {
		return tf, nil
	}
	return "", fmt.Errorf("unsupported timeframe: %q", timeframe)
}

// Returns the fixed duration for a timeframe.
func GetTimeframeDuration(timeframe Timeframe) time.Duration {
	switch timeframe {
	case Timeframe1Minute:
		return time.Minute
	case Timeframe5Minutes:
		return 5 * time.Minute
	case Timeframe15Minutes:
		return 15 * time.Minute
	case Timeframe30Minutes:
		return 30 * time.Minute
	case Timeframe1Hour:
		return time.Hour
	case Timeframe4Hours:
		return 4 * time.Hour
	case Timeframe8Hours:
		return 8 * time.Hour
	case Timeframe12Hours:
		return 12 * time.Hour
	case Timeframe1Day:
		return 24 * time.Hour
	case Timeframe3Days:
		return 3 * 24 * time.Hour
	case Timeframe1Week:
		return 7 * 24 * time.Hour
	case Timeframe1Month:
		// Monthly timeframes require special handling - return 0 to indicate special case
		return 0
	case Timeframe3Months:
		// Quarterly timeframes require special handling - return 0 to indicate special case
		return 0
	default:
		return 0
	}
}

// TruncateToTimeframe truncates a time to the appropriate timeframe boundary
// For monthly timeframes, this truncates to the 1st day of the month at 00:00:00
// For other timeframes, this uses the standard duration-based truncation
func TruncateToTimeframe(t time.Time, timeframe Timeframe) time.Time {
	switch timeframe {
	case Timeframe1Month:
		// For monthly timeframes, truncate to the 1st day of the month
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case Timeframe3Months:
		// For quarterly timeframes, truncate to the 1st day of the quarter
		quarterMonth := t.Month() - (t.Month()-1)%3
		return time.Date(t.Year(), quarterMonth, 1, 0, 0, 0, 0, t.Location())
	default:
		// For all other timeframes, use standard duration-based truncation
		duration := GetTimeframeDuration(timeframe)
		if duration == 0 {
			return t // Return original time if invalid timeframe
		}
		return t.Truncate(duration)
	}
}

// Returns the start of the next period after the given time.
// For monthly timeframes, this advances by calendar month(s).
// For all others, it adds the fixed duration.
func NextTimeframeBoundary(startTime time.Time, timeframe Timeframe) time.Time {
	switch timeframe {
	case Timeframe1Month:
		// For monthly timeframes, end at the start of the next month (standard boundary)
		return startTime.AddDate(0, 1, 0)
	case Timeframe3Months:
		// For quarterly timeframes, end at the start of the next quarter
		return startTime.AddDate(0, 3, 0)
	default:
		// For all other timeframes, add the duration
		duration := GetTimeframeDuration(timeframe)
		if duration == 0 {
			return startTime // Return start time if invalid timeframe
		}
		return startTime.Add(duration)
	}
}

// Returns the parent timeframe for hierarchical aggregation.
func getParentTimeframe(timeframe Timeframe) Timeframe {
	switch timeframe {
	case Timeframe5Minutes:
		return Timeframe1Minute
	case Timeframe15Minutes:
		return Timeframe5Minutes
	case Timeframe30Minutes:
		return Timeframe15Minutes
	case Timeframe1Hour:
		return Timeframe5Minutes
	case Timeframe4Hours:
		return Timeframe1Hour
	case Timeframe8Hours:
		return Timeframe4Hours
	case Timeframe12Hours:
		return Timeframe4Hours
	case Timeframe1Day:
		return Timeframe1Hour
	case Timeframe3Days:
		return Timeframe1Day
	case Timeframe1Week:
		return Timeframe1Day
	case Timeframe1Month:
		return Timeframe1Day
	case Timeframe3Months:
		return Timeframe1Month
	default:
		return ""
	}
}
