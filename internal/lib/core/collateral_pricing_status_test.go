package core

import "testing"

func Test_NormalizeCollateralPricingStatus(t *testing.T) {
	t.Parallel()

	if got := NormalizeCollateralPricingStatus(""); got != CollateralPricingStatusDisabled {
		t.Fatalf("empty -> disabled, got %q", got)
	}
	if got := NormalizeCollateralPricingStatus("  ACTIVE  "); got != CollateralPricingStatusActive {
		t.Fatalf("trim and lower active, got %q", got)
	}
	if got := NormalizeCollateralPricingStatus("future_state"); got != "future_state" {
		t.Fatalf("unknown preserved lowercase, got %q", got)
	}
}

func Test_CollateralPricingStatusIsActive(t *testing.T) {
	t.Parallel()

	if !CollateralPricingStatusIsActive(CollateralPricingStatusActive) {
		t.Fatal("active constant should be active")
	}
	if CollateralPricingStatusIsActive(CollateralPricingStatusDisabled) {
		t.Fatal("disabled should not be active")
	}
	if CollateralPricingStatusIsActive("unknown") {
		t.Fatal("unknown should not be active")
	}
}
