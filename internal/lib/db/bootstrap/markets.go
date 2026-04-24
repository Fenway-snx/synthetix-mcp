package bootstrap

import (
	"context"
	"fmt"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Loads a single market by symbol with its maintenance margin tiers.
// Returns gorm.ErrRecordNotFound if the symbol does not exist.
func (c *Client) LoadMarketBySymbol(ctx context.Context, symbol string) (*postgrestypes.Market, error) {
	var market postgrestypes.Market
	if err := c.db.WithContext(ctx).
		Preload("MaintenanceMarginTiers").
		Where("symbol = ?", symbol).
		First(&market).Error; err != nil {
		return nil, fmt.Errorf("failed to load market %s: %w", symbol, err)
	}
	return &market, nil
}

// Loads all open markets with their maintenance margin tiers.
func (c *Client) LoadActiveMarkets(ctx context.Context) ([]postgrestypes.Market, error) {
	var markets []postgrestypes.Market
	if err := c.db.WithContext(ctx).
		Preload("MaintenanceMarginTiers").
		Where("is_open = ?", true).
		Find(&markets).Error; err != nil {
		return nil, fmt.Errorf("failed to load markets: %w", err)
	}
	return markets, nil
}

// Loads all markets with their maintenance margin tiers, including inactive markets.
func (c *Client) LoadAllMarkets(ctx context.Context) ([]postgrestypes.Market, error) {
	var markets []postgrestypes.Market
	if err := c.db.WithContext(ctx).
		Preload("MaintenanceMarginTiers").
		Find(&markets).Error; err != nil {
		return nil, fmt.Errorf("failed to load all markets: %w", err)
	}
	return markets, nil
}

// Loads all market price entries.
func (c *Client) LoadAllMarketPrices(ctx context.Context) ([]postgrestypes.MarketPrice, error) {
	var prices []postgrestypes.MarketPrice
	if err := c.db.WithContext(ctx).Find(&prices).Error; err != nil {
		return nil, fmt.Errorf("failed to load market prices: %w", err)
	}
	return prices, nil
}

// Loads all SLP exposure limits with their associated market data.
func (c *Client) LoadSLPExposureLimits(ctx context.Context) ([]postgrestypes.SLPExposureLimit, error) {
	var limits []postgrestypes.SLPExposureLimit
	if err := c.db.WithContext(ctx).
		Preload("Market").
		Find(&limits).Error; err != nil {
		return nil, fmt.Errorf("failed to load SLP exposure limits: %w", err)
	}
	return limits, nil
}
