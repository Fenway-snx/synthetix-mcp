package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// FundingRateHistory represents the funding_rate_history table
type FundingRateHistory struct {
	ID          uint64                     `gorm:"primaryKey;column:id" json:"id"`
	Symbol      string                     `gorm:"column:symbol;index:idx_funding_rate_symbol" json:"symbol"`
	FundingRate shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:funding_rate" json:"funding_rate"`
	// TODO: change this type to time.Time (or another strong time type)
	PublishTime int64     `gorm:"column:publish_time;index:idx_funding_rate_publish_time" json:"publish_time"`
	AppliedAt   time.Time `gorm:"column:applied_at;index:idx_funding_rate_applied_at" json:"applied_at"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName returns the table name for GORM
func (FundingRateHistory) TableName() string {
	return "funding_rate_history"
}
