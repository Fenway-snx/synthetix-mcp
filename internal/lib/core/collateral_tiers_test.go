package core

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

func makeCollateralTiers() []postgrestypes.CollateralHaircutTier {
	return []postgrestypes.CollateralHaircutTier{
		{
			TierName:                "Tier 1",
			MinAmountUSDT:           shopspring_decimal.Zero,
			MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(2500000)),
			CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.01"),
			CollateralValueRatio:    shopspring_decimal.RequireFromString("0.99"),
			CollateralValueAddition: shopspring_decimal.Zero,
		},
		{
			TierName:                "Tier 2",
			MinAmountUSDT:           shopspring_decimal.NewFromInt(2500000),
			MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(10000000)),
			CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.03"),
			CollateralValueRatio:    shopspring_decimal.RequireFromString("0.97"),
			CollateralValueAddition: shopspring_decimal.NewFromInt(50000),
		},
		{
			TierName:                "Tier 3",
			MinAmountUSDT:           shopspring_decimal.NewFromInt(10000000),
			MaxAmountUSDT:           nil,
			CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
			CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
			CollateralValueAddition: shopspring_decimal.NewFromInt(250000),
		},
	}
}

func Test_sortCollateralHaircutTiers(t *testing.T) {

	t.Run("sorts by MinAmountUSDT ascending", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{TierName: "Tier 3", MinAmountUSDT: shopspring_decimal.NewFromInt(10000000)},
			{TierName: "Tier 1", MinAmountUSDT: shopspring_decimal.Zero},
			{TierName: "Tier 2", MinAmountUSDT: shopspring_decimal.NewFromInt(2500000)},
		}

		sortCollateralHaircutTiers(tiers)

		assert.Equal(t, "Tier 1", tiers[0].TierName)
		assert.Equal(t, "Tier 2", tiers[1].TierName)
		assert.Equal(t, "Tier 3", tiers[2].TierName)
	})

	t.Run("already sorted is no-op", func(t *testing.T) {
		tiers := makeCollateralTiers()

		sortCollateralHaircutTiers(tiers)

		assert.Equal(t, "Tier 1", tiers[0].TierName)
		assert.Equal(t, "Tier 2", tiers[1].TierName)
		assert.Equal(t, "Tier 3", tiers[2].TierName)
	})
}

func Test_SortAndValidateCollateralHaircutTiers(t *testing.T) {

	t.Run("valid 3-tier config with nil max", func(t *testing.T) {
		tiers := makeCollateralTiers()

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("valid unsorted tiers are sorted then validated", func(t *testing.T) {
		// Provide tiers in reverse order — sort+validate should handle it
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 3",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(10000000),
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(250000),
			},
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(2500000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.01"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.99"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(2500000),
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(10000000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.03"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.97"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(50000),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		// Verify sorted in-place
		assert.Equal(t, "Tier 1", tiers[0].TierName)
		assert.Equal(t, "Tier 2", tiers[1].TierName)
		assert.Equal(t, "Tier 3", tiers[2].TierName)
	})

	t.Run("valid single tier (stablecoin, zero haircut)", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Stablecoin",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.Zero,
				CollateralValueRatio:    shopspring_decimal.NewFromInt(1),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("valid all-bounded tiers (no nil max)", func(t *testing.T) {
		// t1: 0-1000@5%, t2: 1000-5000@15%
		// VA: 0, 1000×(0.15-0.05)=100
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(1000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(5000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.15"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.85"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(100),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("valid sUSDe production tiers", func(t *testing.T) {
		// sUSDe: t1=0-5M@1%, t2=5M-20M@2%, t3=20M-50M@5%
		// VA: 0, 5000000×(0.02-0.01)=50000, 50000+20000000×(0.05-0.02)=650000
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "t1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(5000000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.01"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.99"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "t2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(5000000),
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(20000000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.02"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.98"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(50000),
			},
			{
				TierName:                "t3",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(20000000),
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(50000000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(650000),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("valid fractional haircuts with non-integer VA", func(t *testing.T) {
		// t1: 0-750@0.3%, t2: 750+@0.7%
		// VA[1] = 750 × (0.007 - 0.003) = 3
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(750)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.003"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.997"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(750),
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.007"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.993"),
				CollateralValueAddition: shopspring_decimal.RequireFromString("3"),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("error: empty tiers", func(t *testing.T) {
		err := SortAndValidateCollateralHaircutTiers([]postgrestypes.CollateralHaircutTier{})

		assert.ErrorIs(t, err, errAtLeastOneCollateralHaircutTierRequired)
	})

	t.Run("error: first tier does not start at 0", func(t *testing.T) {
		// VR/VA intentionally omitted — validation short-circuits at the
		// start-at-zero check before reaching the per-tier loop.
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:               "Tier 1",
				MinAmountUSDT:          shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:          nil,
				CollateralValueHaircut: shopspring_decimal.RequireFromString("0.05"),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.ErrorIs(t, err, errFirstCollateralTierMustStartAtZero)
	})

	t.Run("error: gap between tiers", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(1000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(1001), // gap-of-1
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.15"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.85"),
				CollateralValueAddition: shopspring_decimal.RequireFromString("100.1"),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not contiguous")
	})

	t.Run("error: overlapping tiers", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(2000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.15"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.85"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(100),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not contiguous")
	})

	t.Run("error: haircut not monotonically increasing", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(1000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.10"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.90"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"), // lower than prev
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(-50),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be greater than previous")
	})

	t.Run("error: equal haircuts on adjacent tiers", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(1000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"), // equal to prev
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be greater than previous")
	})

	t.Run("error: haircut equals 1", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.NewFromInt(1),
				CollateralValueRatio:    shopspring_decimal.Zero,
				CollateralValueAddition: shopspring_decimal.Zero,
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.ErrorIs(t, err, errHaircutMustBeInZeroToOneRange)
	})

	t.Run("error: haircut greater than 1", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("1.5"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("-0.5"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.ErrorIs(t, err, errHaircutMustBeInZeroToOneRange)
	})

	t.Run("error: negative haircut", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("-0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("1.05"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.ErrorIs(t, err, errHaircutMustBeInZeroToOneRange)
	})

	t.Run("error: multiple nil max tiers", func(t *testing.T) {
		// VR/VA intentionally omitted — validation short-circuits at the
		// nil-max count check before reaching the per-tier loop.
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:               "Tier 1",
				MinAmountUSDT:          shopspring_decimal.Zero,
				MaxAmountUSDT:          nil,
				CollateralValueHaircut: shopspring_decimal.RequireFromString("0.05"),
			},
			{
				TierName:               "Tier 2",
				MinAmountUSDT:          shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:          nil,
				CollateralValueHaircut: shopspring_decimal.RequireFromString("0.10"),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at most one tier")
	})

	t.Run("error: min >= max", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(0),
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.Zero),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be less than")
	})

	t.Run("error: nil max not on last tier", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(5000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.10"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.90"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(50),
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.ErrorIs(t, err, errOnlyLastTierCanHaveNilMax)
	})

	t.Run("error: wrong value ratio", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.90"), // should be 0.95
				CollateralValueAddition: shopspring_decimal.Zero,
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.ErrorIs(t, err, errValueRatioMismatch)
	})

	t.Run("error: wrong value addition", func(t *testing.T) {
		tiers := []postgrestypes.CollateralHaircutTier{
			{
				TierName:                "Tier 1",
				MinAmountUSDT:           shopspring_decimal.Zero,
				MaxAmountUSDT:           snx_lib_utils_test.MakePointerOf(shopspring_decimal.NewFromInt(1000)),
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.05"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.95"),
				CollateralValueAddition: shopspring_decimal.Zero,
			},
			{
				TierName:                "Tier 2",
				MinAmountUSDT:           shopspring_decimal.NewFromInt(1000),
				MaxAmountUSDT:           nil,
				CollateralValueHaircut:  shopspring_decimal.RequireFromString("0.15"),
				CollateralValueRatio:    shopspring_decimal.RequireFromString("0.85"),
				CollateralValueAddition: shopspring_decimal.NewFromInt(999), // should be 100
			},
		}

		err := SortAndValidateCollateralHaircutTiers(tiers)

		assert.ErrorIs(t, err, errValueAdditionMismatch)
	})
}

func Test_ProgressiveHaircut_BoundaryProof(t *testing.T) {
	// Proof that progressive formula produces continuous values at tier boundaries.
	// cbBTC: t1=0-2.5M@1%, t2=2.5M-10M@3%, t3=10M+@5%
	// VA: 0, 50000, 250000 | VR: 0.99, 0.97, 0.95

	t.Run("$15M yields $14.5M (not $14.25M from flat 5%)", func(t *testing.T) {
		// Progressive: 15M × 0.95 + 250000 = 14,500,000
		usdtValue := shopspring_decimal.NewFromInt(15000000)
		vr := shopspring_decimal.RequireFromString("0.95")
		va := shopspring_decimal.NewFromInt(250000)

		adjusted := usdtValue.Mul(vr).Add(va)
		expected := shopspring_decimal.NewFromInt(14500000)
		assert.True(t, adjusted.Equal(expected), "expected %s, got %s", expected.String(), adjusted.String())

		// Verify via iterative calculation
		tier1 := shopspring_decimal.NewFromInt(2500000).Mul(shopspring_decimal.RequireFromString("0.99"))
		tier2 := shopspring_decimal.NewFromInt(7500000).Mul(shopspring_decimal.RequireFromString("0.97"))
		tier3 := shopspring_decimal.NewFromInt(5000000).Mul(shopspring_decimal.RequireFromString("0.95"))
		iterative := tier1.Add(tier2).Add(tier3)
		assert.True(t, iterative.Equal(expected), "iterative: expected %s, got %s", expected.String(), iterative.String())
	})

	t.Run("continuity at $2.5M boundary (tier 1 → tier 2)", func(t *testing.T) {
		boundary := shopspring_decimal.NewFromInt(2500000)

		// Using tier 1 formula: 2.5M × 0.99 + 0 = 2,475,000
		tier1Result := boundary.Mul(shopspring_decimal.RequireFromString("0.99")).Add(shopspring_decimal.Zero)

		// Using tier 2 formula: 2.5M × 0.97 + 50000 = 2,475,000
		tier2Result := boundary.Mul(shopspring_decimal.RequireFromString("0.97")).Add(shopspring_decimal.NewFromInt(50000))

		assert.True(t, tier1Result.Equal(tier2Result),
			"boundary discontinuity: tier1=%s, tier2=%s", tier1Result.String(), tier2Result.String())
	})

	t.Run("continuity at $10M boundary (tier 2 → tier 3)", func(t *testing.T) {
		boundary := shopspring_decimal.NewFromInt(10000000)

		// Using tier 2 formula: 10M × 0.97 + 50000 = 9,750,000
		tier2Result := boundary.Mul(shopspring_decimal.RequireFromString("0.97")).Add(shopspring_decimal.NewFromInt(50000))

		// Using tier 3 formula: 10M × 0.95 + 250000 = 9,750,000
		tier3Result := boundary.Mul(shopspring_decimal.RequireFromString("0.95")).Add(shopspring_decimal.NewFromInt(250000))

		assert.True(t, tier2Result.Equal(tier3Result),
			"boundary discontinuity: tier2=%s, tier3=%s", tier2Result.String(), tier3Result.String())
	})

	t.Run("sUSDe continuity at $5M boundary (tier 1 → tier 2)", func(t *testing.T) {
		boundary := shopspring_decimal.NewFromInt(5000000)

		// tier 1: 5M × 0.99 + 0 = 4,950,000
		tier1Result := boundary.Mul(shopspring_decimal.RequireFromString("0.99")).Add(shopspring_decimal.Zero)

		// tier 2: 5M × 0.98 + 50000 = 4,950,000
		tier2Result := boundary.Mul(shopspring_decimal.RequireFromString("0.98")).Add(shopspring_decimal.NewFromInt(50000))

		assert.True(t, tier1Result.Equal(tier2Result),
			"sUSDe boundary discontinuity at 5M: tier1=%s, tier2=%s", tier1Result.String(), tier2Result.String())
	})

	t.Run("sUSDe continuity at $20M boundary (tier 2 → tier 3)", func(t *testing.T) {
		boundary := shopspring_decimal.NewFromInt(20000000)

		// tier 2: 20M × 0.98 + 50000 = 19,650,000
		tier2Result := boundary.Mul(shopspring_decimal.RequireFromString("0.98")).Add(shopspring_decimal.NewFromInt(50000))

		// tier 3: 20M × 0.95 + 650000 = 19,650,000
		tier3Result := boundary.Mul(shopspring_decimal.RequireFromString("0.95")).Add(shopspring_decimal.NewFromInt(650000))

		assert.True(t, tier2Result.Equal(tier3Result),
			"sUSDe boundary discontinuity at 20M: tier2=%s, tier3=%s", tier2Result.String(), tier3Result.String())
	})
}
