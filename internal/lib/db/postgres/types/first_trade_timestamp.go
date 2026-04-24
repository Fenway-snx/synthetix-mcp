package types

import (
	"time"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Tracks when the first real trade candle was created for each symbol/timeframe.
// This is used to determine the lower bound for historical candle backfilling —
// we do not backfill before the first trade timestamp for a given market/timeframe.
type FirstTradeTimestamp struct {
	ID             uint64                       `gorm:"primaryKey;column:id" json:"id"`
	Symbol         string                       `gorm:"column:symbol;uniqueIndex:idx_symbol_timeframe;not null" json:"symbol"`       // Trading pair, e.g., "BTC-USD"
	Timeframe      snx_lib_utils_time.Timeframe `gorm:"column:timeframe;uniqueIndex:idx_symbol_timeframe;not null" json:"timeframe"` // Timeframe, e.g., "1m", "5m", "1h", "1d"
	FirstTradeTime time.Time                    `gorm:"column:first_trade_time;not null" json:"first_trade_time"`                    // Timestamp of first candle with trades
	CreatedAt      time.Time                    `gorm:"column:created_at;autoCreateTime" json:"created_at"`                          // Record creation time
	UpdatedAt      time.Time                    `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                          // Record update time
}

// TableName returns the table name for GORM
func (FirstTradeTimestamp) TableName() string {
	return "first_trade_timestamps"
}
