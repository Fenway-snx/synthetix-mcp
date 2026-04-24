package core

import (
	"errors"
	"fmt"
	"slices"

	shopspring_decimal "github.com/shopspring/decimal"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
	snx_lib_utils_decimal "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/decimal"
)

var (
	errAtLeastOneCollateralHaircutTierRequired = errors.New("at least one collateral haircut tier is required")
	errFirstCollateralTierMustStartAtZero      = errors.New("first tier must start at min_amount_usdt 0")
	errHaircutMustBeInZeroToOneRange           = errors.New("haircut must be in [0, 1) range — haircut >= 1 would produce zero/negative collateral value")
	errOnlyLastTierCanHaveNilMax               = errors.New("only the last tier can have nil max_amount_usdt")
	errValueAdditionMismatch                   = errors.New("collateral_value_addition does not match boundary-continuity formula")
	errValueRatioMismatch                      = errors.New("collateral_value_ratio does not match 1 - haircut")
)

// Sorts tiers by MinAmountUSDT in ascending order.
func sortCollateralHaircutTiers(tiers []postgrestypes.CollateralHaircutTier) {
	slices.SortFunc(tiers, func(a, b postgrestypes.CollateralHaircutTier) int {
		return a.MinAmountUSDT.Cmp(b.MinAmountUSDT)
	})
}

// Sorts tiers in-place by MinAmountUSDT ascending, then validates the tier
// configuration for progressive bracket calculation.
//
// Structural checks (contiguity, monotonicity, bounds) run first so that
// errors report the root cause rather than a derived-field mismatch.
//
// Rules:
//   - At least one tier required
//   - First tier must start at 0
//   - All haircuts must be in [0, 1) — haircut >= 1 would zero/negate collateral value
//   - At most one nil-max tier, and it must be the last
//   - For bounded tiers, min < max
//   - Contiguous ranges: nextMin == prevMax (uses [min, max) half-open semantics)
//   - Haircuts must increase monotonically
//   - ValueRatio must equal 1 - haircut
//   - ValueAddition must follow boundary-continuity formula:
//     VA[0] = 0, VA[i] = VA[i-1] + min[i] × (haircut[i] - haircut[i-1])
func SortAndValidateCollateralHaircutTiers(tiers []postgrestypes.CollateralHaircutTier) error {
	if len(tiers) == 0 {
		return errAtLeastOneCollateralHaircutTierRequired
	}

	sortCollateralHaircutTiers(tiers)

	// First tier must start at 0
	if !tiers[0].MinAmountUSDT.IsZero() {
		return errFirstCollateralTierMustStartAtZero
	}

	// Check nil-max count — at most one, must be last
	var nilMaxCount int
	for _, tier := range tiers {
		if tier.MaxAmountUSDT == nil {
			nilMaxCount++
		}
	}
	if nilMaxCount > 1 {
		return fmt.Errorf("at most one tier can have nil max_amount_usdt (unbounded), found %d", nilMaxCount)
	}

	// --- Structural checks first (root-cause errors) ---

	for i := range tiers {
		tier := &tiers[i]

		// Haircut must be in [0, 1)
		if tier.CollateralValueHaircut.IsNegative() || tier.CollateralValueHaircut.GreaterThanOrEqual(snx_lib_utils_decimal.Decimal_1) {
			return fmt.Errorf("tier %d (%s): %w (got %s)",
				i+1, tier.TierName, errHaircutMustBeInZeroToOneRange, tier.CollateralValueHaircut.String())
		}

		// For bounded tiers, validate min < max
		if tier.MaxAmountUSDT != nil {
			if tier.MinAmountUSDT.GreaterThanOrEqual(*tier.MaxAmountUSDT) {
				return fmt.Errorf("tier %d (%s): min_amount_usdt (%s) must be less than max_amount_usdt (%s)",
					i+1, tier.TierName, tier.MinAmountUSDT.String(), tier.MaxAmountUSDT.String())
			}
		}

		// Only last tier can have nil max
		if tier.MaxAmountUSDT == nil && i < len(tiers)-1 {
			return errOnlyLastTierCanHaveNilMax
		}

		// Check contiguity with next tier: nextMin must equal prevMax ([min, max) semantics).
		if i < len(tiers)-1 {
			nextTier := &tiers[i+1]
			if !nextTier.MinAmountUSDT.Equal(*tier.MaxAmountUSDT) {
				return fmt.Errorf("tier %d and %d are not contiguous: tier %d ends at %s but tier %d starts at %s (expected equal for [min, max) semantics)",
					i+1, i+2, i+1, tier.MaxAmountUSDT.String(), i+2, nextTier.MinAmountUSDT.String())
			}
		}

		// Haircut must increase monotonically
		if i > 0 {
			prevTier := &tiers[i-1]
			if tier.CollateralValueHaircut.LessThanOrEqual(prevTier.CollateralValueHaircut) {
				return fmt.Errorf("tier %d (%s): haircut (%s) must be greater than previous tier (%s)",
					i+1, tier.TierName, tier.CollateralValueHaircut.String(), prevTier.CollateralValueHaircut.String())
			}
		}
	}

	// --- Derived-field checks (VR, VA) ---

	expectedVA := shopspring_decimal.Zero

	for i := range tiers {
		tier := &tiers[i]

		// VR must equal 1 - haircut
		expectedVR := snx_lib_utils_decimal.Decimal_1.Sub(tier.CollateralValueHaircut)
		if !tier.CollateralValueRatio.Equal(expectedVR) {
			return fmt.Errorf("tier %d (%s): %w (expected %s, got %s)",
				i+1, tier.TierName, errValueRatioMismatch, expectedVR.String(), tier.CollateralValueRatio.String())
		}

		// VA must follow boundary-continuity formula
		if i > 0 {
			prevTier := &tiers[i-1]
			haircutDiff := tier.CollateralValueHaircut.Sub(prevTier.CollateralValueHaircut)
			expectedVA = expectedVA.Add(tier.MinAmountUSDT.Mul(haircutDiff))
		}
		if !tier.CollateralValueAddition.Equal(expectedVA) {
			return fmt.Errorf("tier %d (%s): %w (expected %s, got %s)",
				i+1, tier.TierName, errValueAdditionMismatch, expectedVA.String(), tier.CollateralValueAddition.String())
		}
	}

	return nil
}
