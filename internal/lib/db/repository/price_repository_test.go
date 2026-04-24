package repository

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildLuaResultRow returns a 14-element []interface{} that mirrors
// what the candlesAggregateScript Lua script returns for one kline.
func buildLuaResultRow(
	priceChange, priceChangePercent, weightedAvgPrice,
	openPrice, highPrice, lowPrice, lastPrice,
	volume, quoteVolume,
	openTime, closeTime,
	firstId, lastId,
	count string,
) []interface{} {
	return []interface{}{
		priceChange,
		priceChangePercent,
		weightedAvgPrice,
		openPrice,
		highPrice,
		lowPrice,
		lastPrice,
		volume,
		quoteVolume,
		openTime,
		closeTime,
		firstId,
		lastId,
		count,
	}
}

func Test_postprocessCandle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected Candle
	}{
		{
			name: "valid 14-element candle",
			input: []string{
				"100",           // [0] priceChange
				"0.25",          // [1] priceChangePercent
				"40050",         // [2] weightedAvgPrice
				"40000",         // [3] openPrice
				"40200",         // [4] highPrice
				"39900",         // [5] lowPrice
				"40100",         // [6] lastPrice (close)
				"150",           // [7] volume
				"6000",          // [8] quoteVolume
				"1700000000000", // [9] openTime
				"1700000059999", // [10] closeTime
				"1",             // [11] firstId
				"50",            // [12] lastId
				"50",            // [13] count
			},
			expected: Candle{
				OpenTime:       1700000000000,
				Open:           shopspring_decimal.NewFromInt(40000),
				High:           shopspring_decimal.NewFromInt(40200),
				Low:            shopspring_decimal.NewFromInt(39900),
				Close:          shopspring_decimal.NewFromInt(40100),
				Volume:         shopspring_decimal.NewFromInt(0),
				CloseTime:      1700000059999,
				NumberOfTrades: 0,
			},
		},
		{
			name: "decimal prices",
			input: []string{
				"12.5",
				"0.03",
				"42012.75",
				"42000.25",
				"42100.50",
				"41900.125",
				"42012.75",
				"0",
				"0",
				"1700000000000",
				"1700000059999",
				"0",
				"0",
				"0",
			},
			expected: Candle{
				OpenTime:       1700000000000,
				Open:           shopspring_decimal.RequireFromString("42000.25"),
				High:           shopspring_decimal.RequireFromString("42100.50"),
				Low:            shopspring_decimal.RequireFromString("41900.125"),
				Close:          shopspring_decimal.RequireFromString("42012.75"),
				Volume:         shopspring_decimal.NewFromInt(0),
				CloseTime:      1700000059999,
				NumberOfTrades: 0,
			},
		},
		{
			name:     "fewer than 14 elements returns zero candle",
			input:    []string{"100", "0.25", "40050"},
			expected: Candle{},
		},
		{
			name:     "empty slice returns zero candle",
			input:    []string{},
			expected: Candle{},
		},
		{
			name:     "nil slice returns zero candle",
			input:    nil,
			expected: Candle{},
		},
		{
			name: "invalid openTime returns zero candle",
			input: []string{
				"100", "0.25", "40050", "40000", "40200", "39900", "40100",
				"150", "6000",
				"not-a-number",
				"1700000059999",
				"1", "50", "50",
			},
			expected: Candle{},
		},
		{
			name: "invalid closeTime returns zero candle",
			input: []string{
				"100", "0.25", "40050", "40000", "40200", "39900", "40100",
				"150", "6000",
				"1700000000000",
				"not-a-number",
				"1", "50", "50",
			},
			expected: Candle{},
		},
		{
			name: "invalid openPrice returns zero candle",
			input: []string{
				"100", "0.25", "40050", "abc", "40200", "39900", "40100",
				"150", "6000",
				"1700000000000", "1700000059999",
				"1", "50", "50",
			},
			expected: Candle{},
		},
		{
			name: "invalid highPrice returns zero candle",
			input: []string{
				"100", "0.25", "40050", "40000", "abc", "39900", "40100",
				"150", "6000",
				"1700000000000", "1700000059999",
				"1", "50", "50",
			},
			expected: Candle{},
		},
		{
			name: "invalid lowPrice returns zero candle",
			input: []string{
				"100", "0.25", "40050", "40000", "40200", "abc", "40100",
				"150", "6000",
				"1700000000000", "1700000059999",
				"1", "50", "50",
			},
			expected: Candle{},
		},
		{
			name: "invalid closePrice returns zero candle",
			input: []string{
				"100", "0.25", "40050", "40000", "40200", "39900", "abc",
				"150", "6000",
				"1700000000000", "1700000059999",
				"1", "50", "50",
			},
			expected: Candle{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := postprocessCandle(tc.input)
			assert.Equal(t, tc.expected.OpenTime, result.OpenTime, "OpenTime")
			assert.Equal(t, tc.expected.CloseTime, result.CloseTime, "CloseTime")
			assert.Equal(t, tc.expected.NumberOfTrades, result.NumberOfTrades, "NumberOfTrades")
			assert.True(t, tc.expected.Open.Equal(result.Open), "Open: expected %s, got %s", tc.expected.Open, result.Open)
			assert.True(t, tc.expected.High.Equal(result.High), "High: expected %s, got %s", tc.expected.High, result.High)
			assert.True(t, tc.expected.Low.Equal(result.Low), "Low: expected %s, got %s", tc.expected.Low, result.Low)
			assert.True(t, tc.expected.Close.Equal(result.Close), "Close: expected %s, got %s", tc.expected.Close, result.Close)
			assert.True(t, tc.expected.Volume.Equal(result.Volume), "Volume: expected %s, got %s", tc.expected.Volume, result.Volume)
		})
	}
}

// Verifies that mapstructure.Decode correctly converts the nested
// []interface{} shape returned by Redis Eval (Lua script) into [][]string.
// This is the exact conversion that GetCandles relies on.
func Test_mapstructure_Decode_luaResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       interface{}
		expected    [][]string
		expectError bool
	}{
		{
			name: "single candle row",
			input: []interface{}{
				buildLuaResultRow(
					"100", "0.25", "40050",
					"40000", "40200", "39900", "40100",
					"150", "6000",
					"1700000000000", "1700000059999",
					"1", "50", "50",
				),
			},
			expected: [][]string{
				{
					"100", "0.25", "40050",
					"40000", "40200", "39900", "40100",
					"150", "6000",
					"1700000000000", "1700000059999",
					"1", "50", "50",
				},
			},
		},
		{
			name: "multiple candle rows",
			input: []interface{}{
				buildLuaResultRow(
					"100", "0.25", "40050",
					"40000", "40200", "39900", "40100",
					"150", "6000",
					"1700000000000", "1700000059999",
					"1", "50", "50",
				),
				buildLuaResultRow(
					"-50", "-0.12", "40075",
					"40100", "40150", "40000", "40050",
					"200", "8000",
					"1700000060000", "1700000119999",
					"51", "100", "50",
				),
			},
			expected: [][]string{
				{
					"100", "0.25", "40050",
					"40000", "40200", "39900", "40100",
					"150", "6000",
					"1700000000000", "1700000059999",
					"1", "50", "50",
				},
				{
					"-50", "-0.12", "40075",
					"40100", "40150", "40000", "40050",
					"200", "8000",
					"1700000060000", "1700000119999",
					"51", "100", "50",
				},
			},
		},
		{
			name:     "empty result",
			input:    []interface{}{},
			expected: [][]string{},
		},
		{
			name: "integer values are rejected (Lua always returns strings via tostring)",
			input: []interface{}{
				[]interface{}{
					int64(100), int64(0), int64(40050),
					int64(40000), int64(40200), int64(39900), int64(40100),
					int64(150), int64(6000),
					int64(1700000000000), int64(1700000059999),
					int64(1), int64(50), int64(50),
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result [][]string
			err := mapstructure.Decode(tc.input, &result)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Exercises the full pipeline: mapstructure.Decode followed by
// postprocessCandle, mirroring GetCandles.
func Test_mapstructure_Decode_thenPostprocess(t *testing.T) {
	t.Parallel()

	luaResult := []interface{}{
		buildLuaResultRow(
			"100", "0.25", "40050",
			"40000", "40200", "39900", "40100",
			"150", "6000",
			"1700000000000", "1700000059999",
			"1", "50", "50",
		),
		buildLuaResultRow(
			"-50", "-0.12", "40075",
			"40100", "40150", "40000", "40050",
			"200", "8000",
			"1700000060000", "1700000119999",
			"51", "100", "50",
		),
	}

	var valuesAsArray [][]string
	err := mapstructure.Decode(luaResult, &valuesAsArray)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, valuesAsArray, 2)

	candles := make([]Candle, 0, len(valuesAsArray))
	for _, value := range valuesAsArray {
		candles = append(candles, postprocessCandle(value))
	}

	require.Len(t, candles, 2)

	// First candle
	assert.Equal(t, uint64(1700000000000), candles[0].OpenTime)
	assert.Equal(t, uint64(1700000059999), candles[0].CloseTime)
	assert.True(t, shopspring_decimal.NewFromInt(40000).Equal(candles[0].Open))
	assert.True(t, shopspring_decimal.NewFromInt(40200).Equal(candles[0].High))
	assert.True(t, shopspring_decimal.NewFromInt(39900).Equal(candles[0].Low))
	assert.True(t, shopspring_decimal.NewFromInt(40100).Equal(candles[0].Close))

	// Second candle
	assert.Equal(t, uint64(1700000060000), candles[1].OpenTime)
	assert.Equal(t, uint64(1700000119999), candles[1].CloseTime)
	assert.True(t, shopspring_decimal.NewFromInt(40100).Equal(candles[1].Open))
	assert.True(t, shopspring_decimal.NewFromInt(40150).Equal(candles[1].High))
	assert.True(t, shopspring_decimal.NewFromInt(40000).Equal(candles[1].Low))
	assert.True(t, shopspring_decimal.NewFromInt(40050).Equal(candles[1].Close))
}
