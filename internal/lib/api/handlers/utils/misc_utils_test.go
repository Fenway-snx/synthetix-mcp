package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
)

func Test_CoalesceTimeRange(t *testing.T) {
	t.Run("all zero returns zero", func(t *testing.T) {
		start, end, err := CoalesceTimeRange(Timestamp_Zero, Timestamp_Zero, Timestamp_Zero, Timestamp_Zero)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Timestamp_Zero, start)
		assert.Equal(t, Timestamp_Zero, end)
	})

	t.Run("only startTime/endTime provided", func(t *testing.T) {
		start, end, err := CoalesceTimeRange(Timestamp(1000), Timestamp(2000), Timestamp_Zero, Timestamp_Zero)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Timestamp(1000), start)
		assert.Equal(t, Timestamp(2000), end)
	})

	t.Run("only fromTime/toTime provided", func(t *testing.T) {
		start, end, err := CoalesceTimeRange(Timestamp_Zero, Timestamp_Zero, Timestamp(1000), Timestamp(2000))
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Timestamp(1000), start)
		assert.Equal(t, Timestamp(2000), end)
	})

	t.Run("both provided with same values succeeds", func(t *testing.T) {
		start, end, err := CoalesceTimeRange(Timestamp(1000), Timestamp(2000), Timestamp(1000), Timestamp(2000))
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Timestamp(1000), start)
		assert.Equal(t, Timestamp(2000), end)
	})

	t.Run("conflicting start values returns error", func(t *testing.T) {
		_, _, err := CoalesceTimeRange(Timestamp(1000), Timestamp_Zero, Timestamp(1500), Timestamp_Zero)
		assert.Error(t, err)
	})

	t.Run("conflicting end values returns error", func(t *testing.T) {
		_, _, err := CoalesceTimeRange(Timestamp_Zero, Timestamp(2000), Timestamp_Zero, Timestamp(2500))
		assert.Error(t, err)
	})

	t.Run("mixed: startTime with toTime", func(t *testing.T) {
		start, end, err := CoalesceTimeRange(Timestamp(1000), Timestamp_Zero, Timestamp_Zero, Timestamp(2000))
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Timestamp(1000), start)
		assert.Equal(t, Timestamp(2000), end)
	})
}

func Test_TradeDirectionToSide(t *testing.T) {
	t.Run("sell directions", func(t *testing.T) {
		for _, dir := range []string{"open short", "Open Short", "OPEN SHORT", "close long", "Close Long", "short", "Short", "sell", "Sell"} {
			side, err := TradeDirectionToSide(dir)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, "sell", side, "direction=%q", dir)
		}
	})

	t.Run("buy directions", func(t *testing.T) {
		for _, dir := range []string{"open long", "Open Long", "OPEN LONG", "close short", "Close Short", "long", "Long", "buy", "Buy"} {
			side, err := TradeDirectionToSide(dir)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, "buy", side, "direction=%q", dir)
		}
	})

	t.Run("whitespace is trimmed", func(t *testing.T) {
		side, err := TradeDirectionToSide("  sell  ")
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, "sell", side)
	})

	t.Run("unrecognized direction returns error", func(t *testing.T) {
		for _, dir := range []string{"unknown", "Unknown", "UNKNOWN", "sideways", "", "garbage", "buy sell"} {
			_, err := TradeDirectionToSide(dir)
			assert.Error(t, err, "direction=%q should return error", dir)
		}
	})
}

func Test_APIStartEndToCoreStartEndPtrs(t *testing.T) {

	{
		tb_start, tb_end, err, qual := APIStartEndToCoreStartEndPtrs(Timestamp_Zero, Timestamp_Zero, Timestamp_Zero)

		assert.Nil(t, tb_start)
		assert.Nil(t, tb_end)
		assert.Nil(t, err)
		assert.Equal(t, "", qual)
	}

	{
		ts_start, _ := snx_lib_api_types.TimestampDate(2025, time.November, 12, 9, 0, 0, 0)
		ts_end, _ := snx_lib_api_types.TimestampDate(2025, time.November, 12, 10, 0, 0, 0)
		ts_now, _ := snx_lib_api_types.TimestampDate(2025, time.November, 13, 8, 0, 0, 0)

		assert.NotEqual(t, ts_start, ts_end)
		assert.NotEqual(t, ts_start, ts_now)
		assert.NotEqual(t, ts_end, ts_now)

		tb_start, tb_end, err, qual := APIStartEndToCoreStartEndPtrs(ts_start, ts_end, ts_now)

		assert.NotNil(t, tb_start)
		assert.NotNil(t, tb_end)
		assert.Nil(t, err)
		assert.Equal(t, "", qual)

		tm_ts_start, _ := ts_start.AsTime()
		tm_ts_end, _ := ts_end.AsTime()

		tm_tb_start := tb_start.AsTime()
		tm_tb_end := tb_end.AsTime()

		assert.True(t, tm_ts_start.Equal(tm_tb_start))
		assert.True(t, tm_ts_end.Equal(tm_tb_end), "the actual time value %v is not equal to the expected time value %v", tm_tb_end, tm_ts_end)
	}

	{
		ts_start, _ := snx_lib_api_types.TimestampDate(2025, time.November, 13, 9, 0, 0, 0)
		ts_end, _ := snx_lib_api_types.TimestampDate(2025, time.November, 13, 10, 0, 0, 0)
		ts_now, _ := snx_lib_api_types.TimestampDate(2025, time.November, 13, 8, 0, 0, 0)

		assert.NotEqual(t, ts_start, ts_end)
		assert.NotEqual(t, ts_start, ts_now)
		assert.NotEqual(t, ts_end, ts_now)

		tb_start, tb_end, err, qual := APIStartEndToCoreStartEndPtrs(ts_start, ts_end, ts_now)

		assert.NotNil(t, tb_start)
		assert.NotNil(t, tb_end)
		assert.Nil(t, err)
		assert.Equal(t, "", qual)

		tm_ts_start, _ := ts_start.AsTime()
		tm_ts_now, _ := ts_now.AsTime()

		tm_tb_start := tb_start.AsTime()
		tm_tb_end := tb_end.AsTime()

		assert.True(t, tm_ts_start.Equal(tm_tb_start))
		assert.True(t, tm_ts_now.Equal(tm_tb_end), "the actual time value %v is not equal to the expected time value %v", tm_tb_end, tm_ts_now)
	}
}
