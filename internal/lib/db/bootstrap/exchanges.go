package bootstrap

import (
	"context"
	"fmt"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Loads all Futures Exchange configuration rows.
func (c *Client) LoadAllFuturesExchanges(ctx context.Context) ([]postgrestypes.FuturesExchange, error) {
	var exchanges []postgrestypes.FuturesExchange
	if err := c.db.WithContext(ctx).Find(&exchanges).Error; err != nil {
		return nil, fmt.Errorf("failed to load futures exchanges: %w", err)
	}
	return exchanges, nil
}

// Loads all spot exchange configuration rows.
func (c *Client) LoadAllSpotExchanges(ctx context.Context) ([]postgrestypes.SpotExchange, error) {
	var exchanges []postgrestypes.SpotExchange
	if err := c.db.WithContext(ctx).Find(&exchanges).Error; err != nil {
		return nil, fmt.Errorf("failed to load spot exchanges: %w", err)
	}
	return exchanges, nil
}

// Loads all aggregate config rows (keyed by config_type: "price" and "index").
func (c *Client) LoadAggregateConfigs(ctx context.Context) ([]postgrestypes.AggregateConfig, error) {
	var configs []postgrestypes.AggregateConfig
	if err := c.db.WithContext(ctx).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to load aggregate configs: %w", err)
	}
	return configs, nil
}

// Loads all collateral pricing configuration rows.
func (c *Client) LoadCollateralPricingConfigs(ctx context.Context) ([]postgrestypes.CollateralPricingConfiguration, error) {
	var configs []postgrestypes.CollateralPricingConfiguration
	if err := c.db.WithContext(ctx).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to load collateral pricing configs: %w", err)
	}
	return configs, nil
}
