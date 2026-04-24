package bootstrap

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Loads all collaterals with their active haircut tiers ordered by tier name.
// Mirrors the exact GORM preload pattern from MarketConfig's collateral repository.
func (c *Client) LoadCollateralConfigs(ctx context.Context) ([]postgrestypes.Collateral, error) {
	var collaterals []postgrestypes.Collateral
	if err := c.db.WithContext(ctx).
		Preload("CollateralHaircutTiers", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", 1).Order("tier_name")
		}).
		Find(&collaterals).Error; err != nil {
		return nil, fmt.Errorf("failed to load collateral configs: %w", err)
	}
	return collaterals, nil
}
