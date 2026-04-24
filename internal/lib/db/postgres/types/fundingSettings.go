package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// FundingSettings holds global, non-market-specific settings for pricing/funding
type FundingSettings struct {
	ID                        uint64                     `json:"id" gorm:"primaryKey"`
	BaseInterestRatePer8Hours shopspring_decimal.Decimal `json:"base_interest_rate_per_8_hours"`
	TargetOrderbookDepth      int                        `json:"target_orderbook_depth"`
	CreatedAt                 time.Time                  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                 time.Time                  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}
