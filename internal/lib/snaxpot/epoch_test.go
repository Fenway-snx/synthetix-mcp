package snaxpot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_EpochForTime(t *testing.T) {
	anchor := time.Date(2026, time.April, 5, 18, 0, 0, 0, time.UTC)

	assert.Equal(t, uint64(1), EpochForTime(anchor, anchor))
	assert.Equal(t, uint64(0), EpochForTime(anchor.Add(-time.Second), anchor))
	assert.Equal(t, uint64(2), EpochForTime(anchor.Add(EpochDuration), anchor))
}

func Test_EpochBounds_ROUND_TRIP(t *testing.T) {
	anchor := time.Date(2026, time.April, 5, 18, 0, 0, 0, time.UTC)

	start, end := EpochBounds(3, anchor)

	assert.Equal(t, anchor.Add(2*EpochDuration), start)
	assert.Equal(t, anchor.Add(3*EpochDuration), end)
	assert.Equal(t, uint64(3), EpochForTime(start, anchor))
}

func Test_EpochBounds_ZERO_EPOCH_ID_RETURNS_EMPTY_ANCHOR_INTERVAL(t *testing.T) {
	anchor := time.Date(2026, time.April, 5, 18, 0, 0, 0, time.UTC)

	start, end := EpochBounds(0, anchor)

	assert.Equal(t, anchor, start)
	assert.Equal(t, anchor, end)
}

func Test_NextCutoffAfter(t *testing.T) {
	testCases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "monday advances to same week sunday",
			now:  time.Date(2026, time.April, 6, 12, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.April, 12, 18, 0, 0, 0, time.UTC),
		},
		{
			name: "sunday before cutoff stays same day",
			now:  time.Date(2026, time.April, 12, 17, 59, 59, 0, time.UTC),
			want: time.Date(2026, time.April, 12, 18, 0, 0, 0, time.UTC),
		},
		{
			name: "sunday at cutoff advances one week",
			now:  time.Date(2026, time.April, 12, 18, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.April, 19, 18, 0, 0, 0, time.UTC),
		},
		{
			name: "non utc input normalizes before computing cutoff",
			now:  time.Date(2026, time.April, 12, 13, 0, 0, 0, time.FixedZone("EDT", -4*60*60)),
			want: time.Date(2026, time.April, 12, 18, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, NextCutoffAfter(tc.now))
		})
	}
}
