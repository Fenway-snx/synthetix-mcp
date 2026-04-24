package bootstrap

import (
	"context"
	"fmt"

	shopspring_decimal "github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/tier"
	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Aggregates 14-day trading volumes per subaccount from trade_history.
// Returns a map of subaccount ID to total filled_value over the last 14 days.
// Callers should set an appropriate context timeout as this query scans trade_history
// which can be large at production scale.
func (c *Client) LoadSubaccountVolumes14Day(ctx context.Context) (map[int64]shopspring_decimal.Decimal, error) {
	type volumeRow struct {
		SubAccountId int64                      `gorm:"column:sub_account_id"`
		TotalVolume  shopspring_decimal.Decimal `gorm:"column:total_volume"`
	}

	startTime := snx_lib_utils_time.Now().AddDate(0, 0, -14)
	endTime := snx_lib_utils_time.Now()

	var rows []volumeRow
	if err := c.db.WithContext(ctx).
		Model(&postgrestypes.TradeHistory{}).
		Select("sub_account_id, COALESCE(SUM(CAST(filled_value AS DECIMAL)), 0) as total_volume").
		Where("traded_at >= ? AND traded_at < ?", startTime, endTime).
		Group("sub_account_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load 14-day subaccount volumes: %w", err)
	}

	result := make(map[int64]shopspring_decimal.Decimal, len(rows))
	for _, r := range rows {
		result[r.SubAccountId] = r.TotalVolume
	}
	return result, nil
}

// Loads the latest custom wallet-to-tier assignments. Uses MAX(id) subquery to get
// the most recent assignment per wallet_address (append-only table pattern).
// Only returns assignments that reference custom-type tiers; volume-type tier
// assignments are excluded because volume tiers are determined by trading volume.
func (c *Client) LoadWalletTierMappings(ctx context.Context) ([]postgrestypes.WalletTier, error) {
	latestWalletIds := c.db.Model(&postgrestypes.WalletTier{}).
		Select("MAX(id)").
		Group("wallet_address")

	var walletTiers []postgrestypes.WalletTier
	if err := c.db.WithContext(ctx).
		Where("wallet_tiers.id IN (?)", latestWalletIds).
		Joins("JOIN tiers_view ON tiers_view.tier_id = wallet_tiers.tier_id AND tiers_view.tier_type = ?", tier.Type_custom).
		Find(&walletTiers).Error; err != nil {
		return nil, fmt.Errorf("failed to load wallet tier mappings: %w", err)
	}
	return walletTiers, nil
}

// Loads the latest tier definitions via the tiers_view.
// Includes both volume and custom tier types.
func (c *Client) LoadAllTiers(ctx context.Context) ([]postgrestypes.Tier, error) {
	var tiers []postgrestypes.Tier
	if err := c.db.WithContext(ctx).
		Model(&postgrestypes.Tier_View{}).
		Find(&tiers).Error; err != nil {
		return nil, fmt.Errorf("failed to load tier definitions: %w", err)
	}
	return tiers, nil
}
