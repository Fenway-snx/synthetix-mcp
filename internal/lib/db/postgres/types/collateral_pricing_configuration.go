package types

import "time"

// Represents collateral pricing configuration in the database.
type CollateralPricingConfiguration struct {
	ID                        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Collateral                string    `gorm:"column:collateral;type:varchar(20);uniqueIndex;not null"`
	CoinmetricsSymbol         string    `gorm:"column:coinmetrics_symbol;type:varchar(50);not null"`
	Status                    string    `gorm:"column:status;type:varchar(20);not null"`
	PriceConversionPair       string    `gorm:"column:price_conversion_pair;type:varchar(20);not null"`
	PriceStalenessThresholdMS int32     `gorm:"column:price_staleness_threshold_ms;not null"`
	CreatedAt                 time.Time `gorm:"column:created_at;not null"`
	UpdatedAt                 time.Time `gorm:"column:updated_at;not null"`
}

func (CollateralPricingConfiguration) TableName() string {
	return "collateral_pricing_configuration"
}
