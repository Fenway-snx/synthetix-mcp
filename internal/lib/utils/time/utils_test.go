package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type timeRangeTestCase struct {
	name     string
	tm       time.Time
	tmFrom   time.Time
	tmTo     time.Time
	expected bool
	desc     string
}

func Test_TimeWithinRangeExclusiveEnd(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	tmFrom := baseTime
	tmTo := baseTime.Add(10 * time.Second)

	tests := []timeRangeTestCase{
		{
			name:     "time before range",
			tm:       baseTime.Add(-1 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: false,
			desc:     "time before range should return false",
		},
		{
			name:     "time at start of range",
			tm:       tmFrom,
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: true,
			desc:     "time at start of range should return true (inclusive start)",
		},
		{
			name:     "time in middle of range",
			tm:       baseTime.Add(5 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: true,
			desc:     "time in middle of range should return true",
		},
		{
			name:     "time just before end of range",
			tm:       tmTo.Add(-1 * time.Nanosecond),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: true,
			desc:     "time just before end should return true (exclusive end)",
		},
		{
			name:     "time at end of range",
			tm:       tmTo,
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: false,
			desc:     "time at end of range should return false (exclusive end)",
		},
		{
			name:     "time after range",
			tm:       baseTime.Add(11 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: false,
			desc:     "time after range should return false",
		},
		{
			name:     "empty range (tmFrom == tmTo)",
			tm:       tmFrom,
			tmFrom:   tmFrom,
			tmTo:     tmFrom,
			expected: false,
			desc:     "empty range [x, x) should return false for any time",
		},
		{
			name:     "empty range with different time",
			tm:       baseTime.Add(1 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmFrom,
			expected: false,
			desc:     "empty range [x, x) should return false for any other time",
		},
		{
			name:     "nanosecond precision - time just before end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 999999999, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			expected: true,
			desc:     "time 1 nanosecond before end should return true",
		},
		{
			name:     "nanosecond precision - time at end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			expected: false,
			desc:     "time exactly at end should return false (exclusive end)",
		},
		{
			name:     "different time zones",
			tm:       time.Date(2024, 1, 15, 7, 0, 5, 0, time.FixedZone("EST", -5*3600)),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 10, 0, time.UTC),
			expected: true,
			desc:     "should work correctly with different time zones",
		},
		{
			name:     "very small range (1 nanosecond) - time at start",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1, time.UTC),
			expected: true,
			desc:     "time at start of 1-nanosecond range should return true",
		},
		{
			name:     "very small range (1 nanosecond) - time at end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 1, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1, time.UTC),
			expected: false,
			desc:     "time at end of 1-nanosecond range should return false",
		},
		{
			name:     "nanosecond precision - time after end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 1000000001, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			expected: false,
			desc:     "time 1 nanosecond after end should return false",
		},
		{
			name:     "large time range",
			tm:       time.Date(2050, 6, 15, 12, 30, 45, 0, time.UTC),
			tmFrom:   time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: true,
			desc:     "should work correctly with large time ranges",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimeWithinRangeExclusiveEnd(tt.tm, tt.tmFrom, tt.tmTo)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

func Test_TimeWithinRangeInclusiveEnd(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	tmFrom := baseTime
	tmTo := baseTime.Add(10 * time.Second)

	tests := []timeRangeTestCase{
		{
			name:     "time before range",
			tm:       baseTime.Add(-1 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: false,
			desc:     "time before range should return false",
		},
		{
			name:     "time at start of range",
			tm:       tmFrom,
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: true,
			desc:     "time at start of range should return true (inclusive start)",
		},
		{
			name:     "time in middle of range",
			tm:       baseTime.Add(5 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: true,
			desc:     "time in middle of range should return true",
		},
		{
			name:     "time just before end of range",
			tm:       tmTo.Add(-1 * time.Nanosecond),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: true,
			desc:     "time just before end should return true",
		},
		{
			name:     "time at end of range",
			tm:       tmTo,
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: true,
			desc:     "time at end of range should return true (inclusive end)",
		},
		{
			name:     "time after range",
			tm:       baseTime.Add(11 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmTo,
			expected: false,
			desc:     "time after range should return false",
		},
		{
			name:     "empty range (tmFrom == tmTo)",
			tm:       tmFrom,
			tmFrom:   tmFrom,
			tmTo:     tmFrom,
			expected: true,
			desc:     "empty range [x, x] should return true for the single point",
		},
		{
			name:     "empty range with different time",
			tm:       baseTime.Add(1 * time.Second),
			tmFrom:   tmFrom,
			tmTo:     tmFrom,
			expected: false,
			desc:     "empty range [x, x] should return false for any other time",
		},
		{
			name:     "nanosecond precision - time just before end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 999999999, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			expected: true,
			desc:     "time 1 nanosecond before end should return true",
		},
		{
			name:     "nanosecond precision - time at end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			expected: true,
			desc:     "time exactly at end should return true (inclusive end)",
		},
		{
			name:     "nanosecond precision - time after end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 1000000001, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1000000000, time.UTC),
			expected: false,
			desc:     "time 1 nanosecond after end should return false",
		},
		{
			name:     "different time zones",
			tm:       time.Date(2024, 1, 15, 7, 0, 10, 0, time.FixedZone("EST", -5*3600)),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 10, 0, time.UTC),
			expected: true,
			desc:     "should work correctly with different time zones",
		},
		{
			name:     "very small range (1 nanosecond) - time at start",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1, time.UTC),
			expected: true,
			desc:     "time at start of 1-nanosecond range should return true",
		},
		{
			name:     "very small range (1 nanosecond) - time at end",
			tm:       time.Date(2024, 1, 15, 12, 0, 0, 1, time.UTC),
			tmFrom:   time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2024, 1, 15, 12, 0, 0, 1, time.UTC),
			expected: true,
			desc:     "time at end of 1-nanosecond range should return true (inclusive end)",
		},
		{
			name:     "large time range",
			tm:       time.Date(2050, 6, 15, 12, 30, 45, 0, time.UTC),
			tmFrom:   time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			tmTo:     time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: true,
			desc:     "should work correctly with large time ranges",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimeWithinRangeInclusiveEnd(tt.tm, tt.tmFrom, tt.tmTo)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}
