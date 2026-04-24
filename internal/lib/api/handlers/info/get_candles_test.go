package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

func Test_ValidateCandleRequest(t *testing.T) {
	t.Run("valid requests with all timeframes", func(t *testing.T) {
		for _, tf := range snx_lib_utils_time.SupportedTimeframes {
			t.Run(string(tf), func(t *testing.T) {
				req := &CandleRequest{
					Symbol:   "BTC-USD",
					Interval: string(tf),
				}
				err := validateCandleRequest(req)
				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			})
		}
	})

	t.Run("missing symbol", func(t *testing.T) {
		req := &CandleRequest{
			Interval: "1h",
		}
		err := validateCandleRequest(req)
		require.Error(t, err)
		assert.Equal(t, snx_lib_api_validation.ErrSymbolRequired, err)
	})

	t.Run("missing interval", func(t *testing.T) {
		req := &CandleRequest{
			Symbol: "BTC-USD",
		}
		err := validateCandleRequest(req)
		require.Error(t, err)
		assert.Equal(t, errIntervalRequired, err)
	})

	t.Run("invalid intervals", func(t *testing.T) {
		invalids := []string{"2m", "10m", "2h", "3h", "6h", "2d", "2w", "2M", "invalid", "1H", "1D"}
		for _, interval := range invalids {
			t.Run(interval, func(t *testing.T) {
				req := &CandleRequest{
					Symbol:   "BTC-USD",
					Interval: interval,
				}
				err := validateCandleRequest(req)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid interval")
			})
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		req := &CandleRequest{
			Symbol:   "BTC-USD",
			Interval: "1h",
			Limit:    -1,
		}
		err := validateCandleRequest(req)
		require.Error(t, err)
		assert.Equal(t, errLimitMustBeNonNegative, err)
	})

	t.Run("zero limit is valid", func(t *testing.T) {
		req := &CandleRequest{
			Symbol:   "BTC-USD",
			Interval: "1h",
			Limit:    0,
		}
		err := validateCandleRequest(req)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("positive limit is valid", func(t *testing.T) {
		req := &CandleRequest{
			Symbol:   "BTC-USD",
			Interval: "15m",
			Limit:    100,
		}
		err := validateCandleRequest(req)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("startTime >= endTime is invalid", func(t *testing.T) {
		req := &CandleRequest{
			Symbol:    "BTC-USD",
			Interval:  "1h",
			StartTime: 1000,
			EndTime:   500,
		}
		err := validateCandleRequest(req)
		require.Error(t, err)
		assert.Equal(t, errStartTimeMustBeLess, err)
	})

	t.Run("equal startTime and endTime is invalid", func(t *testing.T) {
		req := &CandleRequest{
			Symbol:    "BTC-USD",
			Interval:  "1h",
			StartTime: 1000,
			EndTime:   1000,
		}
		err := validateCandleRequest(req)
		require.Error(t, err)
		assert.Equal(t, errStartTimeMustBeLess, err)
	})

	t.Run("valid startTime and endTime", func(t *testing.T) {
		req := &CandleRequest{
			Symbol:    "BTC-USD",
			Interval:  "30m",
			StartTime: 500,
			EndTime:   1000,
		}
		err := validateCandleRequest(req)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("new timeframes accepted", func(t *testing.T) {
		newTimeframes := []string{"30m", "8h", "12h", "3d", "3M"}
		for _, tf := range newTimeframes {
			t.Run(tf, func(t *testing.T) {
				req := &CandleRequest{
					Symbol:   "ETH-USD",
					Interval: tf,
					Limit:    50,
				}
				err := validateCandleRequest(req)
				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			})
		}
	})
}
