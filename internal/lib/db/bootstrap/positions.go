package bootstrap

import (
	"context"
	"fmt"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// OITotals holds aggregated open interest for a single symbol.
type OITotals struct {
	Symbol  string
	LongOI  shopspring_decimal.Decimal
	ShortOI shopspring_decimal.Decimal
}

// Loads all position records.
func (c *Client) LoadAllPositions(ctx context.Context) ([]postgrestypes.Position, error) {
	var positions []postgrestypes.Position
	if err := c.db.WithContext(ctx).Find(&positions).Error; err != nil {
		return nil, fmt.Errorf("failed to load positions: %w", err)
	}
	return positions, nil
}

// Loads aggregated open interest totals for all symbols in a single query.
// Returns a map keyed by symbol. Replaces per-symbol gRPC calls with one GROUP BY query.
func (c *Client) LoadAllOITotals(ctx context.Context) (map[string]*OITotals, error) {
	type oiRow struct {
		Symbol  string
		LongOI  shopspring_decimal.Decimal `gorm:"column:long_oi"`
		ShortOI shopspring_decimal.Decimal `gorm:"column:short_oi"`
	}

	var rows []oiRow
	if err := c.db.WithContext(ctx).
		Model(&postgrestypes.Position{}).
		Select(`symbol,
			COALESCE(SUM(CASE WHEN side = ? THEN CAST(quantity AS NUMERIC) ELSE 0 END), 0) as long_oi,
			COALESCE(SUM(CASE WHEN side = ? THEN CAST(quantity AS NUMERIC) ELSE 0 END), 0) as short_oi`,
			int32(snx_lib_core.PositionSideLong),
			int32(snx_lib_core.PositionSideShort)).
		Group("symbol").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to load OI totals: %w", err)
	}

	result := make(map[string]*OITotals, len(rows))
	for _, r := range rows {
		result[r.Symbol] = &OITotals{
			Symbol:  r.Symbol,
			LongOI:  r.LongOI,
			ShortOI: r.ShortOI,
		}
	}
	return result, nil
}
