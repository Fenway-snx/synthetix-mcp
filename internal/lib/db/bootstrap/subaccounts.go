package bootstrap

import (
	"context"
	"fmt"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Loads all subaccount records.
func (c *Client) LoadAllSubaccounts(ctx context.Context) ([]postgrestypes.SubAccount, error) {
	var subaccounts []postgrestypes.SubAccount
	if err := c.db.WithContext(ctx).Find(&subaccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to load subaccounts: %w", err)
	}
	return subaccounts, nil
}

// Loads all subaccount collateral balances with their collateral configuration.
func (c *Client) LoadAllSubaccountCollaterals(ctx context.Context) ([]postgrestypes.SubAccountCollateral, error) {
	var collaterals []postgrestypes.SubAccountCollateral
	if err := c.db.WithContext(ctx).
		Preload("Collateral").
		Find(&collaterals).Error; err != nil {
		return nil, fmt.Errorf("failed to load subaccount collaterals: %w", err)
	}
	return collaterals, nil
}

// Loads all per-symbol leverage overrides for subaccounts.
func (c *Client) LoadAllSubaccountLeverages(ctx context.Context) ([]postgrestypes.SubAccountLeverage, error) {
	var leverages []postgrestypes.SubAccountLeverage
	if err := c.db.WithContext(ctx).Find(&leverages).Error; err != nil {
		return nil, fmt.Errorf("failed to load subaccount leverages: %w", err)
	}
	return leverages, nil
}
