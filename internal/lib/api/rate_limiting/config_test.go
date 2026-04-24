package ratelimiting

import (
	"strings"
	"testing"

	viper "github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseHandlerTokenCosts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected HandlerTokenCosts
	}{
		{
			name:     "normal input",
			input:    "placeOrders=5,cancelOrders=2",
			expected: HandlerTokenCosts{"placeOrders": 5, "cancelOrders": 2},
		},
		{
			name:     "leading space from YAML folding",
			input:    " placeOrders=5,cancelOrders=2",
			expected: HandlerTokenCosts{"placeOrders": 5, "cancelOrders": 2},
		},
		{
			name:     "spaces around keys and values",
			input:    " placeOrders = 5 , cancelOrders = 2 ",
			expected: HandlerTokenCosts{"placeOrders": 5, "cancelOrders": 2},
		},
		{
			name:     "empty string",
			input:    "",
			expected: HandlerTokenCosts{},
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: HandlerTokenCosts{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			costs, err := parseHandlerTokenCosts(tt.input)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expected, costs)
		})
	}
}

func Test_LoadOrderRateLimiterConfig_1(t *testing.T) {

	t.Run("normative - all defaults", func(t *testing.T) {

		yaml := `
ratelimiting:
`

		v := viper.New()

		v.SetConfigType("yaml")

		err := v.ReadConfig(strings.NewReader(yaml))

		require.Nil(t, err, "expected `nil` but obtained %v", err)

		cfg, err := LoadOrderRateLimiterConfig(v)

		require.Nil(t, err)

		assert.Equal(t, int64(1000), cfg.WindowMs)
		assert.Equal(t, RateLimit(100), cfg.GeneralRateLimit)
		assert.NotNil(t, cfg.SpecificRateLimits)
	})

	t.Run("normative - without specific rate limits", func(t *testing.T) {

		yaml := `
ratelimiting:
  window_ms: 1234
  GENERAL_RATE_LIMIT: 22
`

		v := viper.New()

		v.SetConfigType("yaml")

		err := v.ReadConfig(strings.NewReader(yaml))

		require.Nil(t, err, "expected `nil` but obtained %v", err)

		cfg, err := LoadOrderRateLimiterConfig(v)

		require.Nil(t, err)

		assert.Equal(t, int64(1234), cfg.WindowMs)
		assert.Equal(t, RateLimit(22), cfg.GeneralRateLimit)
		assert.NotNil(t, cfg.SpecificRateLimits)
		assert.Equal(t, PerSubAccountRateLimits{}, cfg.SpecificRateLimits)
	})

	t.Run("normative - with specific rate limits", func(t *testing.T) {

		yaml := `
ratelimiting:
  window_ms: 2000
  GENERAL_RATE_LIMIT: 22
  SPECIFIC_RATE_LIMITS: 123456=23,123457=24
`

		v := viper.New()

		v.SetConfigType("yaml")

		err := v.ReadConfig(strings.NewReader(yaml))

		require.Nil(t, err, "expected `nil` but obtained %v", err)

		cfg, err := LoadOrderRateLimiterConfig(v)

		require.Nil(t, err)

		assert.Equal(t, int64(2000), cfg.WindowMs)
		assert.Equal(t, RateLimit(22), cfg.GeneralRateLimit)
		assert.Equal(t, PerSubAccountRateLimits{
			123456: 23,
			123457: 24,
		}, cfg.SpecificRateLimits)
	})

	t.Run("normative - with specific rate limits in separate lines", func(t *testing.T) {
		yaml := `
ratelimiting:
  window_ms: 2000
  GENERAL_RATE_LIMIT: 22
  SPECIFIC_RATE_LIMITS:
    123456: 23
    123457: "24"
`

		v := viper.New()

		v.SetConfigType("yaml")

		err := v.ReadConfig(strings.NewReader(yaml))

		require.Nil(t, err, "expected `nil` but obtained %v", err)

		cfg, err := LoadOrderRateLimiterConfig(v)

		require.Nil(t, err)

		assert.Equal(t, int64(2000), cfg.WindowMs)
		assert.Equal(t, RateLimit(22), cfg.GeneralRateLimit)
		assert.Equal(t, PerSubAccountRateLimits{
			123456: 23,
			123457: 24,
		}, cfg.SpecificRateLimits)
	})
}
