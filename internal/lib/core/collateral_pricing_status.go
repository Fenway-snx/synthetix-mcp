package core

import "strings"

// Allowed collateral_pricing_configuration.status values (database + gRPC).
// Add new tokens here and in the proto comment on CollateralPricingConfig before use.
const (
	CollateralPricingStatusActive   = "active"
	CollateralPricingStatusDisabled = "disabled"
)

// CollateralPricingStatusIsActive reports whether pricing should run for this status.
func CollateralPricingStatusIsActive(status string) bool {
	return NormalizeCollateralPricingStatus(status) == CollateralPricingStatusActive
}

// NormalizeCollateralPricingStatus returns a canonical lowercase status or
// CollateralPricingStatusDisabled when empty. Unknown non-empty values are
// returned lowercased and trimmed; they are not active unless equal to
// CollateralPricingStatusActive.
func NormalizeCollateralPricingStatus(s string) string {
	n := strings.ToLower(strings.TrimSpace(s))
	switch n {
	case CollateralPricingStatusActive, CollateralPricingStatusDisabled:
		return n
	case "":
		return CollateralPricingStatusDisabled
	default:
		return n
	}
}
