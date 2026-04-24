package core

import (
	"errors"
	"fmt"
	"sort"

	shopspring_decimal "github.com/shopspring/decimal"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

var (
	errAtLeastOneMaintenanceMarginTierRequired = errors.New("at least one maintenance margin tier is required")
)

// SortMaintenanceMarginTiers sorts tiers by MinPositionSize in ascending order
func SortMaintenanceMarginTiers(tiers []postgrestypes.MaintenanceMarginTier) {
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].MinPositionSize.LessThan(tiers[j].MinPositionSize)
	})
}

// ValidateMaintenanceMarginTiers validates the tier configuration
func ValidateMaintenanceMarginTiers(tiers []postgrestypes.MaintenanceMarginTier) error {
	if len(tiers) == 0 {
		return errAtLeastOneMaintenanceMarginTierRequired
	}

	// Sort tiers by MinPositionSize to handle out-of-order input
	SortMaintenanceMarginTiers(tiers)

	// Check that exactly one tier has nil MaxPositionSize (the last tier)
	var nilMaxCount int
	for _, tier := range tiers {
		if tier.MaxPositionSize == nil {
			nilMaxCount++
		}
	}
	if nilMaxCount != 1 {
		return fmt.Errorf("exactly one tier must have unlimited max position size (nil), found %d", nilMaxCount)
	}

	// Check that the first tier starts at 0
	if !tiers[0].MinPositionSize.IsZero() {
		return fmt.Errorf("first tier must start at position size 0, got '%s'", tiers[0].MinPositionSize.String())
	}

	// Validate each tier and check continuity
	for i := range len(tiers) {
		tier := &tiers[i]

		// Validate max leverage
		if tier.MaxLeverage <= 0 {
			return fmt.Errorf("tier %d: max leverage must be greater than 0", i+1)
		}

		// For tiers with a cap, validate min < max
		if tier.MaxPositionSize != nil {
			if tier.MinPositionSize.GreaterThanOrEqual(*tier.MaxPositionSize) {
				return fmt.Errorf("tier %d: min position size (%s) must be less than max position size (%s)",
					i+1, tier.MinPositionSize.String(), tier.MaxPositionSize.String())
			}
		}

		// Check continuity with next tier (except for the last tier)
		if i < len(tiers)-1 {
			nextTier := &tiers[i+1]
			if tier.MaxPositionSize == nil {
				return fmt.Errorf("tier %d: only the last tier can have unlimited max position size", i+1)
			}
			// Check for overlap
			if nextTier.MinPositionSize.LessThanOrEqual(*tier.MaxPositionSize) {
				return fmt.Errorf("tier %d and %d overlap: tier %d ends at %s but tier %d starts at %s",
					i+1, i+2, i+1, tier.MaxPositionSize.String(), i+2, nextTier.MinPositionSize.String())
			}

			// Check that next tier starts exactly 1 unit after current tier ends
			expectedNextMin := tier.MaxPositionSize.Add(shopspring_decimal.NewFromInt(1))
			if !nextTier.MinPositionSize.Equal(expectedNextMin) {
				return fmt.Errorf("tier %d and %d must have a gap of 1: tier %d ends at %s but tier %d starts at %s (expected %s)",
					i+1, i+2, i+1, tier.MaxPositionSize.String(), i+2, nextTier.MinPositionSize.String(), expectedNextMin.String())
			}
		}

		// Verify that maintenance margin rate increases with tiers (lower leverage = higher margin requirement)
		if i > 0 {
			prevTier := &tiers[i-1]
			prevRate := prevTier.GetMaintenanceMarginRate()
			currentRate := tier.GetMaintenanceMarginRate()
			if currentRate.LessThanOrEqual(prevRate) {
				return fmt.Errorf("tier %d: maintenance margin rate (%s) must be greater than previous tier (%s)",
					i+1, currentRate.String(), prevRate.String())
			}
			// Also verify leverage decreases
			if tier.MaxLeverage >= prevTier.MaxLeverage {
				return fmt.Errorf("tier %d: max leverage (%d) must be less than previous tier (%d)",
					i+1, tier.MaxLeverage, prevTier.MaxLeverage)
			}
		}
	}

	return nil
}

// GetMaintenanceDeduction calculates the maintenance deduction for a specific tier
// Formula: deduction = [Floor of Position Bracket on Tier n * (Difference between Maintenance Margin Rate on Tier n and Tier n-1)] + Maintenance Deduction on Tier n-1
// This ensures smooth transitions between tiers
func GetMaintenanceDeduction(tiers []postgrestypes.MaintenanceMarginTier, tierIndex int) shopspring_decimal.Decimal {
	if tierIndex == 0 {
		return shopspring_decimal.Zero
	}

	deduction := shopspring_decimal.Zero

	// Calculate deduction iteratively from tier 1 to tierIndex
	for i := 1; i <= tierIndex; i++ {
		currentTier := &tiers[i]
		prevTier := &tiers[i-1]

		// Floor of Position Bracket on Tier n (which is the MinPositionSize of current tier)
		floorPositionBracket := currentTier.MinPositionSize

		// Difference between Maintenance Margin Rate on Tier n and Tier n-1
		currentRate := currentTier.MaintenanceMarginRatio
		prevRate := prevTier.MaintenanceMarginRatio
		rateDiff := currentRate.Sub(prevRate)

		// Add to deduction: Floor * Rate Difference
		deduction = deduction.Add(floorPositionBracket.Mul(rateDiff))
	}

	return deduction
}

// FindApplicableTier finds the tier that applies to a given notional value
// Note: This function assumes tiers are already sorted by MinPositionSize in ascending order
func FindApplicableTier(tiers []postgrestypes.MaintenanceMarginTier, notionalValue shopspring_decimal.Decimal) *postgrestypes.MaintenanceMarginTier {
	for i, tier := range tiers {
		if tier.MinPositionSize.LessThanOrEqual(notionalValue) &&
			(tier.MaxPositionSize == nil || notionalValue.LessThanOrEqual(*tier.MaxPositionSize)) {
			return &tiers[i]
		}
	}
	return nil
}

// CalculateMaintenanceMargin calculates the maintenance margin for a given notional value
// using the tiered margin system
// Note: This function assumes tiers are already sorted by MinPositionSize in ascending order
func CalculateMaintenanceMargin(tiers []postgrestypes.MaintenanceMarginTier, notionalValue shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	var tier *postgrestypes.MaintenanceMarginTier

	// Find the applicable tier
	for i, t := range tiers {
		if t.MinPositionSize.LessThanOrEqual(notionalValue) &&
			(t.MaxPositionSize == nil || notionalValue.LessThanOrEqual(*t.MaxPositionSize)) {
			tier = &tiers[i]
			break
		}
	}

	if tier == nil {
		return shopspring_decimal.Zero
	}

	// Use the stored rate and deduction from the tier
	rate := tier.MaintenanceMarginRatio
	deduction := tier.MaintenanceDeductionValue

	// Formula: maintenance_margin = notional_value * maintenance_margin_rate - maintenance_deduction
	return notionalValue.Mul(rate).Sub(deduction)
}

// EnrichTiersWithCalculatedFields now just returns the tiers as-is since
// the calculated fields are stored in the database
func EnrichTiersWithCalculatedFields(tiers []postgrestypes.MaintenanceMarginTier) []postgrestypes.MaintenanceMarginTier {
	return tiers
}

// CalculateInitialMargin calculates the initial margin requirement for a given notional value
// using the tiered margin system. Initial margin is 2x the maintenance margin.
func CalculateInitialMargin(tiers []postgrestypes.MaintenanceMarginTier, notionalValue shopspring_decimal.Decimal) shopspring_decimal.Decimal {
	var tier *postgrestypes.MaintenanceMarginTier

	// Find the applicable tier
	for i, t := range tiers {
		if t.MinPositionSize.LessThanOrEqual(notionalValue) &&
			(t.MaxPositionSize == nil || notionalValue.LessThanOrEqual(*t.MaxPositionSize)) {
			tier = &tiers[i]
			break
		}
	}

	if tier == nil {
		return shopspring_decimal.Zero
	}

	// Use the stored initial margin rate and deduction from the tier
	// Initial margin uses the same deduction as maintenance margin
	rate := tier.GetInitialMarginRate()
	deduction := tier.MaintenanceDeductionValue

	// Formula: initial_margin = notional_value * initial_margin_rate - maintenance_deduction
	return notionalValue.Mul(rate).Sub(deduction)
}
