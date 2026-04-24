package fees

import (
	"errors"
	"slices"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/tier"
	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

var (
	errNoVolumeTiersAvailable = errors.New("no volume tiers available")
)

// TradeVolume represents a subaccount's trading volume as a decimal.
type TradeVolume = shopspring_decimal.Decimal

// TradeVolumeBySubAccountId maps subaccount IDs to their trading volume.
type TradeVolumeBySubAccountId = map[snx_lib_core.SubAccountId]TradeVolume

// WalletAssignments maps each wallet to its assigned tier ID (from wallet_tiers table).
type WalletAssignments = map[snx_lib_core.WalletAddress]tier.Id

// TierMap holds resolved tiers keyed by tier ID.
type TierMap = map[tier.Id]postgrestypes.Tier

// WalletTiers holds resolved tier for each wallet address.
type WalletTiers = map[snx_lib_core.WalletAddress]postgrestypes.Tier

// SubAccountTiers holds resolved tier for each sub account.
type SubAccountTiers = map[snx_lib_core.SubAccountId]postgrestypes.Tier

// Aggregates individual subaccount volumes by owner (wallet_address).
// All subaccounts under the same wallet share the same fee tier based on combined volume.
func CalculateOwnerVolumes(
	volumes TradeVolumeBySubAccountId,
	ownerMappings map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress,
) map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal {
	ownerVolumes := make(map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal, len(ownerMappings))

	for subAccountId, volume := range volumes {
		walletAddress := ownerMappings[subAccountId]

		if existingVolume, exists := ownerVolumes[walletAddress]; exists {
			ownerVolumes[walletAddress] = existingVolume.Add(volume)
		} else {
			ownerVolumes[walletAddress] = volume
		}
	}

	return ownerVolumes
}

// Determines the tier for each wallet based on their trading volume.
// Custom tiers override volume-based tiers. Wallets without volume or custom tier
// get the lowest volume tier (feeTiers[0]).
// feeTiers must be non-empty; they are sorted by MinTradeVolume ascending defensively.
func CalculateFeesForOwners(
	logger snx_lib_logging.Logger,
	ownerVolumes map[snx_lib_core.WalletAddress]shopspring_decimal.Decimal,
	walletCustomTiers WalletAssignments,
	customTierRates TierMap,
	feeTiers []postgrestypes.Tier,
	ownerMappings map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress,
) (WalletTiers, error) {
	if len(feeTiers) == 0 {
		return nil, errNoVolumeTiersAvailable
	}

	sorted := make([]postgrestypes.Tier, len(feeTiers))
	copy(sorted, feeTiers)
	slices.SortFunc(sorted, func(a, b postgrestypes.Tier) int {
		return a.MinTradeVolume.Cmp(b.MinTradeVolume)
	})

	ownerTiers := make(WalletTiers)
	processedWallets := make(map[snx_lib_core.WalletAddress]bool)

	for walletAddress, volume := range ownerVolumes {
		processedWallets[walletAddress] = true

		if customTierId, hasCustom := walletCustomTiers[walletAddress]; hasCustom {
			if t, exists := customTierRates[customTierId]; exists {
				ownerTiers[walletAddress] = t
				continue
			}
			logger.Warn("wallet has custom tier assignment but tier definition not found",
				"custom_tier_id", customTierId,
				"wallet_address", walletAddress.Masked(),
			)
		}

		var matched postgrestypes.Tier
		found := false
		for _, ft := range sorted {
			if volume.GreaterThanOrEqual(ft.MinTradeVolume) {
				matched = ft
				found = true
			}
		}

		if found {
			ownerTiers[walletAddress] = matched
		} else {
			ownerTiers[walletAddress] = sorted[0]
		}
	}

	for _, walletAddress := range ownerMappings {
		if processedWallets[walletAddress] {
			continue
		}
		processedWallets[walletAddress] = true

		if customTierId, hasCustom := walletCustomTiers[walletAddress]; hasCustom {
			if t, exists := customTierRates[customTierId]; exists {
				ownerTiers[walletAddress] = t
				continue
			}
			logger.Warn("wallet has custom tier assignment but tier definition not found",
				"custom_tier_id", customTierId,
				"wallet_address", walletAddress.Masked(),
			)
		}

		ownerTiers[walletAddress] = sorted[0]
	}

	return ownerTiers, nil
}

// Returns the tier with the smallest MinTradeVolume, or nil if tiers is empty.
// Callers use it as the fallback when a wallet has no entry in wallet tiers.
func LowestVolumeTier(tiers []postgrestypes.Tier) *postgrestypes.Tier {
	if len(tiers) == 0 {
		return nil
	}
	sorted := make([]postgrestypes.Tier, len(tiers))
	copy(sorted, tiers)
	slices.SortFunc(sorted, func(a, b postgrestypes.Tier) int {
		return a.MinTradeVolume.Cmp(b.MinTradeVolume)
	})
	return &sorted[0]
}

// Maps wallet-level tiers to subaccounts. When fallbackTier is non-nil,
// subaccounts whose wallet is not present in walletTiers receive fallbackTier so every
// subaccount in ownerMappings appears in the result.
func MapFeesToSubaccounts(
	walletTiers WalletTiers,
	ownerMappings map[snx_lib_core.SubAccountId]snx_lib_core.WalletAddress,
	fallbackTier *postgrestypes.Tier,
) SubAccountTiers {
	subaccountTiers := make(SubAccountTiers, len(ownerMappings))

	for subAccountId, walletAddress := range ownerMappings {
		if t, exists := walletTiers[walletAddress]; exists {
			subaccountTiers[subAccountId] = t
		} else if fallbackTier != nil {
			subaccountTiers[subAccountId] = *fallbackTier
		}
	}

	return subaccountTiers
}
