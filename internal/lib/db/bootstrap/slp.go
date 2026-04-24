package bootstrap

import (
	"context"
	"fmt"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Loads all SLP-to-subaccount mappings with their associated market and collateral data.
func (c *Client) LoadSLPMappings(ctx context.Context) ([]postgrestypes.SubAccountSLP, error) {
	var mappings []postgrestypes.SubAccountSLP
	if err := c.db.WithContext(ctx).
		Preload("Market").
		Preload("Collateral").
		Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to load SLP mappings: %w", err)
	}
	return mappings, nil
}
