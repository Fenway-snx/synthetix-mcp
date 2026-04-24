package fees

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/tier"
	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func makeTier(id tier.Id, tierType tier.Type, minVol int64, makerRate, takerRate float64) postgrestypes.Tier {
	return postgrestypes.Tier{
		TierId:         id,
		TierType:       tierType,
		TierName:       tier.Name(id),
		MinTradeVolume: shopspring_decimal.NewFromInt(minVol),
		MakerFeeRate:   shopspring_decimal.NewFromFloat(makerRate),
		TakerFeeRate:   shopspring_decimal.NewFromFloat(takerRate),
	}
}

func volumeTiers() []postgrestypes.Tier {
	return []postgrestypes.Tier{
		makeTier("tier_0", tier.Type_volume, 0, 0.001, 0.002),
		makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
		makeTier("tier_2", tier.Type_volume, 500000, 0.0006, 0.0012),
	}
}

// --- CalculateOwnerVolumes ---

func Test_CalculateOwnerVolumes(t *testing.T) {
	t.Run("aggregates multiple subaccounts per wallet", func(t *testing.T) {
		volumes := TradeVolumeBySubAccountId{
			snx_lib_core.SubAccountId(1): shopspring_decimal.NewFromInt(50000),
			snx_lib_core.SubAccountId(2): shopspring_decimal.NewFromInt(30000),
			snx_lib_core.SubAccountId(3): shopspring_decimal.NewFromInt(20000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xaaa",
			snx_lib_core.SubAccountId(2): "0xaaa",
			snx_lib_core.SubAccountId(3): "0xbbb",
		}

		result := CalculateOwnerVolumes(volumes, ownerMappings)

		assert.Len(t, result, 2)
		assert.True(t, result["0xaaa"].Equal(shopspring_decimal.NewFromInt(80000)))
		assert.True(t, result["0xbbb"].Equal(shopspring_decimal.NewFromInt(20000)))
	})

	t.Run("handles single subaccount per wallet", func(t *testing.T) {
		volumes := TradeVolumeBySubAccountId{
			snx_lib_core.SubAccountId(1): shopspring_decimal.NewFromInt(100000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xaaa",
		}

		result := CalculateOwnerVolumes(volumes, ownerMappings)

		assert.Len(t, result, 1)
		assert.True(t, result["0xaaa"].Equal(shopspring_decimal.NewFromInt(100000)))
	})

	t.Run("empty inputs return empty map", func(t *testing.T) {
		result := CalculateOwnerVolumes(
			TradeVolumeBySubAccountId{},
			map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{},
		)
		assert.Empty(t, result)
	})

	t.Run("subaccounts with volume but no owner mapping use zero-value wallet", func(t *testing.T) {
		volumes := TradeVolumeBySubAccountId{
			snx_lib_core.SubAccountId(1): shopspring_decimal.NewFromInt(50000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{}

		result := CalculateOwnerVolumes(volumes, ownerMappings)
		assert.Len(t, result, 1)
		assert.True(t, result[""].Equal(shopspring_decimal.NewFromInt(50000)))
	})
}

// --- CalculateFeesForOwners ---

func Test_CalculateFeesForOwners(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	t.Run("assigns correct volume tiers based on volume", func(t *testing.T) {
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xlow":  shopspring_decimal.NewFromInt(50000),
			"0xmid":  shopspring_decimal.NewFromInt(200000),
			"0xhigh": shopspring_decimal.NewFromInt(600000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xlow",
			snx_lib_core.SubAccountId(2): "0xmid",
			snx_lib_core.SubAccountId(3): "0xhigh",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, nil, nil, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("tier_0"), result["0xlow"].TierId)
		assert.Equal(t, tier.Id("tier_1"), result["0xmid"].TierId)
		assert.Equal(t, tier.Id("tier_2"), result["0xhigh"].TierId)
	})

	t.Run("exact boundary volume matches tier", func(t *testing.T) {
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xexact": shopspring_decimal.NewFromInt(100000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xexact",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, nil, nil, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("tier_1"), result["0xexact"].TierId)
	})

	t.Run("custom tier overrides volume tier", func(t *testing.T) {
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xcustom": shopspring_decimal.NewFromInt(10000),
		}
		customTier := makeTier("market_maker", tier.Type_custom, 0, 0.0001, 0.0002)
		customAssignments := WalletAssignments{"0xcustom": "market_maker"}
		customRates := TierMap{"market_maker": customTier}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xcustom",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, customAssignments, customRates, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("market_maker"), result["0xcustom"].TierId)
	})

	t.Run("custom tier missing definition falls back to volume tier", func(t *testing.T) {
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xorphan": shopspring_decimal.NewFromInt(200000),
		}
		customAssignments := WalletAssignments{"0xorphan": "nonexistent_tier"}
		customRates := TierMap{}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xorphan",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, customAssignments, customRates, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("tier_1"), result["0xorphan"].TierId)
	})

	t.Run("wallets without volume get lowest tier", func(t *testing.T) {
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xnewbie",
		}

		result, err := CalculateFeesForOwners(logger, nil, nil, nil, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("tier_0"), result["0xnewbie"].TierId)
	})

	t.Run("wallet without volume but with custom tier gets custom", func(t *testing.T) {
		customTier := makeTier("vip", tier.Type_custom, 0, 0, 0)
		customAssignments := WalletAssignments{"0xvip": "vip"}
		customRates := TierMap{"vip": customTier}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xvip",
		}

		result, err := CalculateFeesForOwners(logger, nil, customAssignments, customRates, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("vip"), result["0xvip"].TierId)
	})

	t.Run("empty volume tiers returns error", func(t *testing.T) {
		_, err := CalculateFeesForOwners(logger, nil, nil, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no volume tiers available")
	})

	t.Run("unsorted volume tiers are handled correctly", func(t *testing.T) {
		unsorted := []postgrestypes.Tier{
			makeTier("tier_2", tier.Type_volume, 500000, 0.0006, 0.0012),
			makeTier("tier_0", tier.Type_volume, 0, 0.001, 0.002),
			makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
		}
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xmid": shopspring_decimal.NewFromInt(200000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xmid",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, nil, nil, unsorted, ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("tier_1"), result["0xmid"].TierId)
	})
}

// --- LowestVolumeTier ---

func Test_LowestVolumeTier(t *testing.T) {
	t.Run("returns lowest tier from sorted list", func(t *testing.T) {
		tiers := volumeTiers()
		result := LowestVolumeTier(tiers)
		require.NotNil(t, result)
		assert.Equal(t, tier.Id("tier_0"), result.TierId)
	})

	t.Run("returns lowest tier from unsorted list", func(t *testing.T) {
		unsorted := []postgrestypes.Tier{
			makeTier("tier_2", tier.Type_volume, 500000, 0.0006, 0.0012),
			makeTier("tier_0", tier.Type_volume, 0, 0.001, 0.002),
			makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
		}
		result := LowestVolumeTier(unsorted)
		require.NotNil(t, result)
		assert.Equal(t, tier.Id("tier_0"), result.TierId)
	})

	t.Run("returns nil for empty slice", func(t *testing.T) {
		result := LowestVolumeTier(nil)
		assert.Nil(t, result)
	})

	t.Run("single tier returns that tier", func(t *testing.T) {
		tiers := []postgrestypes.Tier{
			makeTier("only", tier.Type_volume, 0, 0.001, 0.002),
		}
		result := LowestVolumeTier(tiers)
		require.NotNil(t, result)
		assert.Equal(t, tier.Id("only"), result.TierId)
	})

	t.Run("does not mutate input slice", func(t *testing.T) {
		original := []postgrestypes.Tier{
			makeTier("tier_2", tier.Type_volume, 500000, 0.0006, 0.0012),
			makeTier("tier_0", tier.Type_volume, 0, 0.001, 0.002),
		}
		firstId := original[0].TierId
		LowestVolumeTier(original)
		assert.Equal(t, firstId, original[0].TierId)
	})
}

// --- CalculateFeesForOwners (additional edge cases) ---

func Test_CalculateFeesForOwners_VolumeBelowAllTiers(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	t.Run("volume below all tiers falls back to first sorted tier", func(t *testing.T) {
		highFloorTiers := []postgrestypes.Tier{
			makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
			makeTier("tier_2", tier.Type_volume, 500000, 0.0006, 0.0012),
		}
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xtiny": shopspring_decimal.NewFromInt(50),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xtiny",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, nil, nil, highFloorTiers, ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, tier.Id("tier_1"), result["0xtiny"].TierId)
	})

	t.Run("zero volume below all non-zero tiers falls back to first sorted tier", func(t *testing.T) {
		highFloorTiers := []postgrestypes.Tier{
			makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
		}
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xzero": shopspring_decimal.Zero,
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xzero",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, nil, nil, highFloorTiers, ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, tier.Id("tier_1"), result["0xzero"].TierId)
	})
}

func Test_CalculateFeesForOwners_SecondLoopCustomTierMissing(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	t.Run("wallet without volume with missing custom tier definition gets lowest tier and logs warning", func(t *testing.T) {
		customAssignments := WalletAssignments{"0xorphan_no_vol": "ghost_tier"}
		customRates := TierMap{}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xorphan_no_vol",
		}

		result, err := CalculateFeesForOwners(logger, nil, customAssignments, customRates, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, tier.Id("tier_0"), result["0xorphan_no_vol"].TierId)
	})
}

func Test_CalculateFeesForOwners_MixedVolumeAndNoVolume(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	t.Run("wallets in both volume and no-volume loops", func(t *testing.T) {
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xhas_vol": shopspring_decimal.NewFromInt(200000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xhas_vol",
			snx_lib_core.SubAccountId(2): "0xno_vol",
			snx_lib_core.SubAccountId(3): "0xno_vol",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, nil, nil, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("tier_1"), result["0xhas_vol"].TierId)
		assert.Equal(t, tier.Id("tier_0"), result["0xno_vol"].TierId)
	})

	t.Run("duplicate wallets in ownerMappings processed once in second loop", func(t *testing.T) {
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xdup",
			snx_lib_core.SubAccountId(2): "0xdup",
			snx_lib_core.SubAccountId(3): "0xdup",
		}

		result, err := CalculateFeesForOwners(logger, nil, nil, nil, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, tier.Id("tier_0"), result["0xdup"].TierId)
	})
}

func Test_CalculateFeesForOwners_SingleTier(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	t.Run("single volume tier works for all wallets", func(t *testing.T) {
		singleTier := []postgrestypes.Tier{
			makeTier("only_tier", tier.Type_volume, 0, 0.001, 0.002),
		}
		ownerVolumes := map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{
			"0xrich": shopspring_decimal.NewFromInt(1000000),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xrich",
			snx_lib_core.SubAccountId(2): "0xnewbie",
		}

		result, err := CalculateFeesForOwners(logger, ownerVolumes, nil, nil, singleTier, ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, tier.Id("only_tier"), result["0xrich"].TierId)
		assert.Equal(t, tier.Id("only_tier"), result["0xnewbie"].TierId)
	})
}

func Test_CalculateFeesForOwners_NilMaps(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	t.Run("nil ownerVolumes and nil ownerMappings", func(t *testing.T) {
		result, err := CalculateFeesForOwners(logger, nil, nil, nil, volumeTiers(), nil)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})

	t.Run("empty ownerVolumes with wallets only in ownerMappings", func(t *testing.T) {
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xonly_in_mappings",
		}

		result, err := CalculateFeesForOwners(logger, map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal{}, nil, nil, volumeTiers(), ownerMappings)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, tier.Id("tier_0"), result["0xonly_in_mappings"].TierId)
	})
}

// --- MapFeesToSubaccounts ---

func Test_MapFeesToSubaccounts(t *testing.T) {
	t.Run("maps wallet tiers to subaccounts", func(t *testing.T) {
		walletTiers := WalletTiers{
			"0xaaa": makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
			"0xbbb": makeTier("tier_0", tier.Type_volume, 0, 0.001, 0.002),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xaaa",
			snx_lib_core.SubAccountId(2): "0xaaa",
			snx_lib_core.SubAccountId(3): "0xbbb",
		}

		result := MapFeesToSubaccounts(walletTiers, ownerMappings, nil)

		assert.Len(t, result, 3)
		assert.Equal(t, tier.Id("tier_1"), result[snx_lib_core.SubAccountId(1)].TierId)
		assert.Equal(t, tier.Id("tier_1"), result[snx_lib_core.SubAccountId(2)].TierId)
		assert.Equal(t, tier.Id("tier_0"), result[snx_lib_core.SubAccountId(3)].TierId)
	})

	t.Run("applies fallback tier for unknown wallets", func(t *testing.T) {
		walletTiers := WalletTiers{
			"0xaaa": makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xaaa",
			snx_lib_core.SubAccountId(2): "0xunknown",
		}
		fallback := makeTier("tier_0", tier.Type_volume, 0, 0.001, 0.002)

		result := MapFeesToSubaccounts(walletTiers, ownerMappings, &fallback)

		assert.Len(t, result, 2)
		assert.Equal(t, tier.Id("tier_1"), result[snx_lib_core.SubAccountId(1)].TierId)
		assert.Equal(t, tier.Id("tier_0"), result[snx_lib_core.SubAccountId(2)].TierId)
	})

	t.Run("unknown wallets omitted when no fallback", func(t *testing.T) {
		walletTiers := WalletTiers{
			"0xaaa": makeTier("tier_1", tier.Type_volume, 100000, 0.0008, 0.0016),
		}
		ownerMappings := map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress{
			snx_lib_core.SubAccountId(1): "0xaaa",
			snx_lib_core.SubAccountId(2): "0xunknown",
		}

		result := MapFeesToSubaccounts(walletTiers, ownerMappings, nil)

		assert.Len(t, result, 1)
		_, exists := result[snx_lib_core.SubAccountId(2)]
		assert.False(t, exists)
	})

	t.Run("empty inputs return empty map", func(t *testing.T) {
		result := MapFeesToSubaccounts(nil, nil, nil)
		assert.Empty(t, result)
	})
}
