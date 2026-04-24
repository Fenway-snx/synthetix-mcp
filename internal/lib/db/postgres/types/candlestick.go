package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Candlestick represents pre-aggregated OHLCV data for a specific timeframe
type Candlestick struct {
	ID          uint64                       `gorm:"primaryKey;column:id" json:"id"`
	Symbol      string                       `gorm:"column:symbol;index:idx_symbol_timeframe_time;not null" json:"symbol"`          // Trading pair, e.g., "BTC-USD"
	Timeframe   snx_lib_utils_time.Timeframe `gorm:"column:timeframe;index:idx_symbol_timeframe_time;not null" json:"timeframe"`    // Timeframe, e.g., "1m", "5m", "1h", "1d"
	OpenTime    time.Time                    `gorm:"column:open_time;index:idx_symbol_timeframe_time;not null" json:"open_time"`    // Candle open time
	CloseTime   time.Time                    `gorm:"column:close_time;not null" json:"close_time"`                                  // Candle close time
	OpenPrice   shopspring_decimal.Decimal   `gorm:"type:decimal(20,8);column:open_price;not null" json:"open_price"`               // Opening price
	HighPrice   shopspring_decimal.Decimal   `gorm:"type:decimal(20,8);column:high_price;not null" json:"high_price"`               // Highest price
	LowPrice    shopspring_decimal.Decimal   `gorm:"type:decimal(20,8);column:low_price;not null" json:"low_price"`                 // Lowest price
	ClosePrice  shopspring_decimal.Decimal   `gorm:"type:decimal(20,8);column:close_price;not null" json:"close_price"`             // Closing price
	Volume      shopspring_decimal.Decimal   `gorm:"type:decimal(20,8);column:volume;not null;default:0" json:"volume"`             // Base asset volume
	QuoteVolume shopspring_decimal.Decimal   `gorm:"type:decimal(20,8);column:quote_volume;not null;default:0" json:"quote_volume"` // Quote asset volume (price * volume)
	TradeCount  int32                        `gorm:"column:trade_count;not null;default:0" json:"trade_count"`                      // Number of trades
	CreatedAt   time.Time                    `gorm:"column:created_at;autoCreateTime" json:"created_at"`                            // Record creation time
	UpdatedAt   time.Time                    `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                            // Record update time
}

// TableName returns the table name for GORM
func (Candlestick) TableName() string {
	return "candlesticks"
}
