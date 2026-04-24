package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Loads the global funding settings. Returns nil, nil if no funding settings row exists.
func (c *Client) LoadFundingSettings(ctx context.Context) (*postgrestypes.FundingSettings, error) {
	var settings postgrestypes.FundingSettings
	if err := c.db.WithContext(ctx).First(&settings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load funding settings: %w", err)
	}
	return &settings, nil
}
