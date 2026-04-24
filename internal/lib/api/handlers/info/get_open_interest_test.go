package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_OpenInterest_Structure(t *testing.T) {
	t.Run("open interest has correct fields including non-zero timestamp", func(t *testing.T) {
		oi := OpenInterest{
			Symbol:            "BTC-USD",
			OpenInterest:      "150.5",
			LongOpenInterest:  "100.3",
			ShortOpenInterest: "50.2",
			Timestamp:         1704067200000,
		}

		assert.Equal(t, Symbol("BTC-USD"), oi.Symbol)
		assert.Equal(t, "150.5", oi.OpenInterest)
		assert.Equal(t, "100.3", oi.LongOpenInterest)
		assert.Equal(t, "50.2", oi.ShortOpenInterest)
		assert.Equal(t, Timestamp(1704067200000), oi.Timestamp)
	})

	t.Run("zero timestamp is detected", func(t *testing.T) {
		oi := OpenInterest{
			Symbol:            "ETH-USD",
			OpenInterest:      "1000",
			LongOpenInterest:  "600",
			ShortOpenInterest: "400",
		}

		assert.Equal(t, Timestamp(0), oi.Timestamp,
			"omitting Timestamp should produce zero value")
	})
}

func Test_OpenInterests_AllEntriesShareTimestamp(t *testing.T) {
	ts := Timestamp(1704067200000)

	entries := OpenInterests{
		{Symbol: "BTC-USD", OpenInterest: "100", LongOpenInterest: "60", ShortOpenInterest: "40", Timestamp: ts},
		{Symbol: "ETH-USD", OpenInterest: "200", LongOpenInterest: "120", ShortOpenInterest: "80", Timestamp: ts},
		{Symbol: "SOL-USD", OpenInterest: "50", LongOpenInterest: "30", ShortOpenInterest: "20", Timestamp: ts},
	}

	for _, entry := range entries {
		assert.Equal(t, ts, entry.Timestamp,
			"all entries should share the same timestamp, got mismatch for %s", entry.Symbol)
	}
}
