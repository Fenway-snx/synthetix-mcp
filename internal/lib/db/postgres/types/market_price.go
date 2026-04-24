package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// Represents a market's price data in the database.
//
// Used by:
// - Market Config Service;
// - Subaccount Service;
type MarketPrice struct {
	ID         uint64                     `gorm:"primarykey" json:"id"`
	Symbol     string                     `gorm:"uniqueIndex;not null" json:"symbol"`
	IndexPrice shopspring_decimal.Decimal `gorm:"type:decimal(20,8)" json:"index_price"`
	LastPrice  shopspring_decimal.Decimal `gorm:"type:decimal(20,8)" json:"last_price"`
	MarkPrice  shopspring_decimal.Decimal `gorm:"type:decimal(20,8)" json:"mark_price"`
	CreatedAt  time.Time                  `json:"created_at"`
	UpdatedAt  time.Time                  `json:"updated_at"`
}

func (MarketPrice) TableName() string {
	return "market_prices"
}
