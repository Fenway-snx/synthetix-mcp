package snaxpot

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSnaxpotConfig() Config {
	return Config{
		USDPerTicket: shopspring_decimal.NewFromInt(2),
	}
}

func Test_ValidSnaxBall(t *testing.T) {
	assert.False(t, ValidSnaxBall(0))
	assert.True(t, ValidSnaxBall(1))
	assert.True(t, ValidSnaxBall(5))
	assert.False(t, ValidSnaxBall(6))
}

func Test_ValidStandardBalls(t *testing.T) {
	assert.True(t, ValidStandardBalls([]int{1, 2, 3, 4, 5}))
	assert.False(t, ValidStandardBalls([]int{0, 2, 3, 4, 5}))
	assert.False(t, ValidStandardBalls([]int{1, 2, 2, 4, 5}))
	assert.False(t, ValidStandardBalls([]int{1, 3, 2, 4, 5}))
	assert.False(t, ValidStandardBalls([]int{1, 2, 3, 4, 33}))
}

func Test_Config_BaseTicketsFromGrossFees(t *testing.T) {
	cfg := newTestSnaxpotConfig()

	assert.Equal(t, int64(0), cfg.BaseTicketsFromGrossFees(shopspring_decimal.Zero))
	assert.Equal(
		t,
		int64(0),
		cfg.BaseTicketsFromGrossFees(shopspring_decimal.RequireFromString("-1")),
	)
	assert.Equal(
		t,
		int64(1),
		cfg.BaseTicketsFromGrossFees(shopspring_decimal.RequireFromString("2.99")),
	)
	assert.Equal(
		t,
		int64(3),
		cfg.BaseTicketsFromGrossFees(shopspring_decimal.RequireFromString("6.00")),
	)
}

func Test_Config_BaseTicketsFromGrossFees_CUSTOM_PRICE(t *testing.T) {
	cfg := newTestSnaxpotConfig()
	cfg.USDPerTicket = shopspring_decimal.RequireFromString("3")

	assert.Equal(
		t,
		int64(2),
		cfg.BaseTicketsFromGrossFees(shopspring_decimal.RequireFromString("8.99")),
	)
}

func Test_StakingMultiplierForSNX(t *testing.T) {
	testCases := []struct {
		name      string
		snxStaked string
		want      string
	}{
		{name: "negative clamps to base tier", snxStaked: "-1", want: "1.00"},
		{name: "below first tier stays base", snxStaked: "999", want: "1.00"},
		{name: "one thousand gets first boost", snxStaked: "1000", want: "1.05"},
		{name: "fifty thousand gets mid tier", snxStaked: "50000", want: "1.40"},
		{name: "one million gets max tier", snxStaked: "1000000", want: "2.00"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(
				t,
				shopspring_decimal.RequireFromString(tc.want),
				StakingMultiplierForSNX(
					shopspring_decimal.RequireFromString(tc.snxStaked),
				),
			)
		})
	}
}

func Test_PurchasedMultiplierForStakingMultiplier(t *testing.T) {
	testCases := []struct {
		name              string
		stakingMultiplier string
		want              string
	}{
		{name: "negative clamps to base", stakingMultiplier: "-1.00", want: "1.00"},
		{name: "base tier stays base", stakingMultiplier: "1.00", want: "1.00"},
		{name: "small uplift is halved", stakingMultiplier: "1.20", want: "1.10"},
		{name: "mid uplift is halved", stakingMultiplier: "1.50", want: "1.25"},
		{name: "max uplift becomes one point five", stakingMultiplier: "2.00", want: "1.50"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(
				t,
				PurchasedMultiplierForStakingMultiplier(
					shopspring_decimal.RequireFromString(tc.stakingMultiplier),
				).Equal(shopspring_decimal.RequireFromString(tc.want)),
			)
		})
	}
}

func Test_TicketsAfterMultiplier(t *testing.T) {
	assert.Equal(
		t,
		int64(0),
		TicketsAfterMultiplier(0, shopspring_decimal.RequireFromString("1.50")),
	)
	assert.Equal(
		t,
		int64(0),
		TicketsAfterMultiplier(10, shopspring_decimal.RequireFromString("0")),
	)
	assert.Equal(
		t,
		int64(4),
		TicketsAfterMultiplier(3, shopspring_decimal.RequireFromString("1.50")),
	)
	assert.Equal(
		t,
		int64(5),
		TicketsAfterMultiplier(3, shopspring_decimal.RequireFromString("1.80")),
	)
}

func Test_LoadConfigFromEnv(t *testing.T) {
	t.Setenv("SNX_SNAXPOT_USD_PER_TICKET", "3.5")

	cfg, err := LoadConfigFromEnv()

	require.NoError(
		t,
		err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)",
		err,
	)
	assert.Equal(t, shopspring_decimal.RequireFromString("3.5"), cfg.USDPerTicket)
}

func Test_LoadConfigFromEnv_REJECTS_INVALID_DECIMAL(t *testing.T) {
	t.Setenv("SNX_SNAXPOT_USD_PER_TICKET", "not-a-decimal")

	_, err := LoadConfigFromEnv()

	require.Error(t, err)
	assert.ErrorContains(t, err, "SNX_SNAXPOT_USD_PER_TICKET must be a valid decimal")
}

func Test_LoadConfigFromEnv_REQUIRES_ALL_VARS(t *testing.T) {
	_, err := LoadConfigFromEnv()

	require.ErrorIs(t, err, errSnaxpotEnvVarRequired)
	require.EqualError(t, err, "snaxpot environment variable is required: SNX_SNAXPOT_USD_PER_TICKET")
}
