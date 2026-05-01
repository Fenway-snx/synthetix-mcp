package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseTimeframe(t *testing.T) {
	t.Run("valid timeframes", func(t *testing.T) {
		tests := []struct {
			input    string
			expected Timeframe
		}{
			{"1m", Timeframe1Minute},
			{"5m", Timeframe5Minutes},
			{"15m", Timeframe15Minutes},
			{"30m", Timeframe30Minutes},
			{"1h", Timeframe1Hour},
			{"4h", Timeframe4Hours},
			{"8h", Timeframe8Hours},
			{"12h", Timeframe12Hours},
			{"1d", Timeframe1Day},
			{"3d", Timeframe3Days},
			{"1w", Timeframe1Week},
			{"1M", Timeframe1Month},
			{"3M", Timeframe3Months},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				tf, err := ParseTimeframe(tt.input)
				require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tt.expected, tf)
			})
		}
	})

	t.Run("invalid timeframes", func(t *testing.T) {
		invalids := []string{"", "2m", "10m", "2h", "3h", "6h", "2d", "2w", "2M", "1s", "invalid", "1H", "1D", "1W"}
		for _, input := range invalids {
			t.Run(input, func(t *testing.T) {
				_, err := ParseTimeframe(input)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported timeframe")
			})
		}
	})
}

func Test_GetTimeframeDuration(t *testing.T) {
	tests := []struct {
		timeframe Timeframe
		expected  time.Duration
	}{
		{Timeframe1Minute, time.Minute},
		{Timeframe5Minutes, 5 * time.Minute},
		{Timeframe15Minutes, 15 * time.Minute},
		{Timeframe30Minutes, 30 * time.Minute},
		{Timeframe1Hour, time.Hour},
		{Timeframe4Hours, 4 * time.Hour},
		{Timeframe8Hours, 8 * time.Hour},
		{Timeframe12Hours, 12 * time.Hour},
		{Timeframe1Day, 24 * time.Hour},
		{Timeframe3Days, 3 * 24 * time.Hour},
		{Timeframe1Week, 7 * 24 * time.Hour},
		{Timeframe1Month, 0},
		{Timeframe3Months, 0},
		{Timeframe("invalid"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.timeframe), func(t *testing.T) {
			assert.Equal(t, tt.expected, GetTimeframeDuration(tt.timeframe))
		})
	}
}

func Test_TruncateToTimeframe(t *testing.T) {
	tests := []struct {
		name      string
		input     time.Time
		timeframe Timeframe
		expected  time.Time
	}{
		{
			name:      "1m truncates seconds",
			input:     time.Date(2025, 6, 15, 14, 37, 42, 0, time.UTC),
			timeframe: Timeframe1Minute,
			expected:  time.Date(2025, 6, 15, 14, 37, 0, 0, time.UTC),
		},
		{
			name:      "5m truncates to 5-minute boundary",
			input:     time.Date(2025, 6, 15, 14, 37, 0, 0, time.UTC),
			timeframe: Timeframe5Minutes,
			expected:  time.Date(2025, 6, 15, 14, 35, 0, 0, time.UTC),
		},
		{
			name:      "15m truncates to 15-minute boundary",
			input:     time.Date(2025, 6, 15, 14, 37, 0, 0, time.UTC),
			timeframe: Timeframe15Minutes,
			expected:  time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
		},
		{
			name:      "30m truncates to 30-minute boundary",
			input:     time.Date(2025, 6, 15, 14, 45, 0, 0, time.UTC),
			timeframe: Timeframe30Minutes,
			expected:  time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
		},
		{
			name:      "30m at boundary stays unchanged",
			input:     time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
			timeframe: Timeframe30Minutes,
			expected:  time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
		},
		{
			name:      "1h truncates minutes",
			input:     time.Date(2025, 6, 15, 14, 45, 0, 0, time.UTC),
			timeframe: Timeframe1Hour,
			expected:  time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		},
		{
			name:      "4h truncates to 4-hour boundary",
			input:     time.Date(2025, 6, 15, 13, 0, 0, 0, time.UTC),
			timeframe: Timeframe4Hours,
			expected:  time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:      "8h truncates to 8-hour boundary",
			input:     time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
			timeframe: Timeframe8Hours,
			expected:  time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
		},
		{
			name:      "8h at boundary stays unchanged",
			input:     time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC),
			timeframe: Timeframe8Hours,
			expected:  time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC),
		},
		{
			name:      "12h truncates to 12-hour boundary",
			input:     time.Date(2025, 6, 15, 18, 30, 0, 0, time.UTC),
			timeframe: Timeframe12Hours,
			expected:  time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:      "12h at midnight stays unchanged",
			input:     time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe12Hours,
			expected:  time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "1d truncates to start of day",
			input:     time.Date(2025, 6, 15, 18, 30, 0, 0, time.UTC),
			timeframe: Timeframe1Day,
			expected:  time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3d truncates to 3-day epoch boundary",
			input:     time.Date(2025, 6, 15, 18, 30, 0, 0, time.UTC),
			timeframe: Timeframe3Days,
			expected:  time.Date(2025, 6, 15, 18, 30, 0, 0, time.UTC).Truncate(3 * 24 * time.Hour),
		},
		{
			name:      "1w truncates to week boundary",
			input:     time.Date(2025, 6, 18, 10, 0, 0, 0, time.UTC),
			timeframe: Timeframe1Week,
			expected:  time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "1M truncates to 1st of month",
			input:     time.Date(2025, 6, 15, 14, 37, 0, 0, time.UTC),
			timeframe: Timeframe1Month,
			expected:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "1M on 1st stays unchanged",
			input:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe1Month,
			expected:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3M truncates to start of Q1 (Jan)",
			input:     time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe3Months,
			expected:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3M truncates to start of Q2 (Apr)",
			input:     time.Date(2025, 5, 20, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe3Months,
			expected:  time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3M truncates to start of Q3 (Jul)",
			input:     time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe3Months,
			expected:  time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3M truncates to start of Q4 (Oct)",
			input:     time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC),
			timeframe: Timeframe3Months,
			expected:  time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3M at quarter start stays unchanged",
			input:     time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe3Months,
			expected:  time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "invalid timeframe returns input unchanged",
			input:     time.Date(2025, 6, 15, 14, 37, 0, 0, time.UTC),
			timeframe: Timeframe("invalid"),
			expected:  time.Date(2025, 6, 15, 14, 37, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateToTimeframe(tt.input, tt.timeframe)
			assert.True(t, tt.expected.Equal(result),
				"TruncateToTimeframe(%v, %s) = %v, want %v", tt.input, tt.timeframe, result, tt.expected,
			)
		})
	}
}

func Test_NextTimeframeBoundary(t *testing.T) {
	tests := []struct {
		name      string
		start     time.Time
		timeframe Timeframe
		expected  time.Time
	}{
		{
			name:      "1m advances by 1 minute",
			start:     time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
			timeframe: Timeframe1Minute,
			expected:  time.Date(2025, 6, 15, 14, 31, 0, 0, time.UTC),
		},
		{
			name:      "5m advances by 5 minutes",
			start:     time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
			timeframe: Timeframe5Minutes,
			expected:  time.Date(2025, 6, 15, 14, 35, 0, 0, time.UTC),
		},
		{
			name:      "15m advances by 15 minutes",
			start:     time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
			timeframe: Timeframe15Minutes,
			expected:  time.Date(2025, 6, 15, 14, 45, 0, 0, time.UTC),
		},
		{
			name:      "30m advances by 30 minutes",
			start:     time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
			timeframe: Timeframe30Minutes,
			expected:  time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC),
		},
		{
			name:      "1h advances by 1 hour",
			start:     time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
			timeframe: Timeframe1Hour,
			expected:  time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC),
		},
		{
			name:      "4h advances by 4 hours",
			start:     time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
			timeframe: Timeframe4Hours,
			expected:  time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC),
		},
		{
			name:      "8h advances by 8 hours",
			start:     time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC),
			timeframe: Timeframe8Hours,
			expected:  time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC),
		},
		{
			name:      "8h crossing midnight",
			start:     time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC),
			timeframe: Timeframe8Hours,
			expected:  time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "12h advances by 12 hours",
			start:     time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe12Hours,
			expected:  time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:      "12h crossing midnight",
			start:     time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
			timeframe: Timeframe12Hours,
			expected:  time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "1d advances by 1 day",
			start:     time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe1Day,
			expected:  time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3d advances by 3 days",
			start:     time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe3Days,
			expected:  time.Date(2025, 6, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "1w advances by 7 days",
			start:     time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe1Week,
			expected:  time.Date(2025, 6, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "1M advances by 1 calendar month",
			start:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe1Month,
			expected:  time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "1M crossing year boundary",
			start:     time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe1Month,
			expected:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3M advances by 3 calendar months",
			start:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe3Months,
			expected:  time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "3M crosses year boundary",
			start:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
			timeframe: Timeframe3Months,
			expected:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "invalid timeframe returns start unchanged",
			start:     time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
			timeframe: Timeframe("invalid"),
			expected:  time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NextTimeframeBoundary(tt.start, tt.timeframe)
			assert.True(t, tt.expected.Equal(result),
				"NextTimeframeBoundary(%v, %s) = %v, want %v", tt.start, tt.timeframe, result, tt.expected,
			)
		})
	}
}

func Test_GetParentTimeframe(t *testing.T) {
	tests := []struct {
		timeframe Timeframe
		expected  Timeframe
	}{
		{Timeframe1Minute, ""},
		{Timeframe5Minutes, Timeframe1Minute},
		{Timeframe15Minutes, Timeframe5Minutes},
		{Timeframe30Minutes, Timeframe15Minutes},
		{Timeframe1Hour, Timeframe5Minutes},
		{Timeframe4Hours, Timeframe1Hour},
		{Timeframe8Hours, Timeframe4Hours},
		{Timeframe12Hours, Timeframe4Hours},
		{Timeframe1Day, Timeframe1Hour},
		{Timeframe3Days, Timeframe1Day},
		{Timeframe1Week, Timeframe1Day},
		{Timeframe1Month, Timeframe1Day},
		{Timeframe3Months, Timeframe1Month},
		{Timeframe("invalid"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.timeframe), func(t *testing.T) {
			assert.Equal(t, tt.expected, getParentTimeframe(tt.timeframe))
		})
	}
}

func Test_SupportedTimeframes_ContainsAllConstants(t *testing.T) {
	expected := []Timeframe{
		Timeframe1Minute, Timeframe5Minutes, Timeframe15Minutes, Timeframe30Minutes,
		Timeframe1Hour, Timeframe4Hours, Timeframe8Hours, Timeframe12Hours,
		Timeframe1Day, Timeframe3Days, Timeframe1Week, Timeframe1Month, Timeframe3Months,
	}

	assert.Equal(t, expected, SupportedTimeframes)

	for _, tf := range SupportedTimeframes {
		parsed, err := ParseTimeframe(string(tf))
		require.NoError(t, err, "SupportedTimeframes entry %q should be parseable", tf)
		assert.Equal(t, tf, parsed)
	}
}

func Test_Timeframe_IsSupported(t *testing.T) {
	for _, tf := range SupportedTimeframes {
		assert.True(t, tf.IsSupported(), "%s should be supported", tf)
	}

	assert.False(t, Timeframe("2h").IsSupported())
	assert.False(t, Timeframe("invalid").IsSupported())
	assert.False(t, Timeframe("").IsSupported())
}

func Test_Timeframe_Base(t *testing.T) {
	tests := []struct {
		timeframe    Timeframe
		expectedBase Timeframe
		expectedOK   bool
	}{
		{Timeframe1Minute, Timeframe1Minute, true},
		{Timeframe5Minutes, Timeframe5Minutes, true},
		{Timeframe15Minutes, Timeframe15Minutes, true},
		{Timeframe30Minutes, Timeframe15Minutes, true},
		{Timeframe1Hour, Timeframe1Hour, true},
		{Timeframe4Hours, Timeframe4Hours, true},
		{Timeframe8Hours, Timeframe4Hours, true},
		{Timeframe12Hours, Timeframe4Hours, true},
		{Timeframe1Day, Timeframe1Day, true},
		{Timeframe3Days, Timeframe1Day, true},
		{Timeframe1Week, Timeframe1Day, true},
		{Timeframe1Month, Timeframe1Day, true},
		{Timeframe3Months, Timeframe1Day, true},
		{Timeframe("2h"), "", false},
		{Timeframe("invalid"), "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.timeframe), func(t *testing.T) {
			base, ok := tt.timeframe.Base()
			assert.Equal(t, tt.expectedOK, ok)
			assert.Equal(t, tt.expectedBase, base)
		})
	}
}

func Test_Timeframe_IsDerived(t *testing.T) {
	derived := []Timeframe{
		Timeframe30Minutes,
		Timeframe8Hours,
		Timeframe12Hours,
		Timeframe3Days,
		Timeframe1Week,
		Timeframe1Month,
		Timeframe3Months,
	}

	for _, tf := range derived {
		assert.True(t, tf.IsDerived(), "%s should be derived", tf)
	}

	notDerived := []Timeframe{
		Timeframe1Minute,
		Timeframe5Minutes,
		Timeframe15Minutes,
		Timeframe1Hour,
		Timeframe4Hours,
		Timeframe1Day,
	}

	for _, tf := range notDerived {
		assert.False(t, tf.IsDerived(), "%s should not be derived", tf)
	}

	assert.False(t, Timeframe("invalid").IsDerived())
}

// Inputs covering best, middle, worst, and invalid cases.
var benchInputs = []string{
	"1m",      // first in list (best case for linear)
	"4h",      // middle of list
	"1M",      // last in list (worst case for linear)
	"invalid", // miss — both approaches must exhaust all cases
}

func Benchmark_ParseTimeframe(b *testing.B) {
	for _, input := range benchInputs {
		b.Run(input, func(b *testing.B) {
			for b.Loop() {
				ParseTimeframe(input)
			}
		})
	}
}
