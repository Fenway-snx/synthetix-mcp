package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
)

func knownDate() time.Time {
	return time.Date(
		2025,
		time.November,
		11,
		10,
		42,
		41,
		123_456_789,
		time.UTC,
	)
}

func Test_CONSTANTS(t *testing.T) {

	// FundingIntervalMillis
	{
		assert.Equal(t, int64(3_600_000), snx_lib_authtest.FundingInterval.Milliseconds())
	}
}

// Used to verify the existing logic before fixing, and then ensure that the
// new tallies correctly.
func Test_GetLatestFundingRate_CALCULATION(t *testing.T) {

	// original
	{
		const fundingIntervalMillis = 3_600_000

		var PublishTime int64       // Publish timestamp
		var NextFundingTime int64   // Next funding time timestamp
		var FundingIntervalMs int64 // Funding interval in milliseconds

		now := knownDate().UnixMilli() // Use milliseconds instead of seconds

		PublishTime = now
		NextFundingTime = now + fundingIntervalMillis
		FundingIntervalMs = fundingIntervalMillis

		assert.Equal(t, int64(1_762_857_761_123), PublishTime)
		assert.Equal(t, int64(1_762_861_361_123), NextFundingTime)
		assert.Equal(t, int64(3_600_000), FundingIntervalMs)
	}

	// new
	{
		const fundingInterval = time.Hour

		var PublishTime int64       // Publish timestamp
		var NextFundingTime int64   // Next funding time timestamp
		var FundingIntervalMs int64 // Funding interval in milliseconds

		now := knownDate().UnixMilli() // Use milliseconds instead of seconds

		PublishTime = now
		NextFundingTime = now + fundingInterval.Milliseconds()
		FundingIntervalMs = fundingInterval.Milliseconds()

		assert.Equal(t, int64(1_762_857_761_123), PublishTime)
		assert.Equal(t, int64(1_762_861_361_123), NextFundingTime)
		assert.Equal(t, int64(3_600_000), FundingIntervalMs)
	}
}
