package bootstrap

import (
	"context"
	"fmt"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Loads all open limit orders for orderbook recovery (Matching service).
// Filters to OrderTypeLimit (type=0) and orders by creation time ascending.
func (c *Client) LoadOpenLimitOrders(ctx context.Context) ([]postgrestypes.OpenOrder, error) {
	var orders []postgrestypes.OpenOrder
	if err := c.db.WithContext(ctx).
		Where("type = ?", int32(snx_lib_core.OrderTypeLimit)).
		Order("created_at ASC").
		Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("failed to load open limit orders: %w", err)
	}
	return orders, nil
}

// Loads all open orders of every type for actor hydration (Trading service).
// Orders by creation time ascending.
func (c *Client) LoadAllOpenOrders(ctx context.Context) ([]postgrestypes.OpenOrder, error) {
	var orders []postgrestypes.OpenOrder
	if err := c.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("failed to load all open orders: %w", err)
	}
	return orders, nil
}

// Loads all TWAP execution state rows keyed by order ID.
func (c *Client) LoadAllOpenOrderTwaps(ctx context.Context) (map[uint64]postgrestypes.OpenOrderTwap, error) {
	var rows []postgrestypes.OpenOrderTwap
	if err := c.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load open order TWAP rows: %w", err)
	}
	out := make(map[uint64]postgrestypes.OpenOrderTwap, len(rows))
	for _, r := range rows {
		out[r.OrderID] = r
	}
	return out, nil
}
