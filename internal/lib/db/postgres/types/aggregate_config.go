package types

import (
	"time"
)

// Represents an aggregate price configuration in the database.
// Two rows: "price" (futures/mark) and "index" (spot).
// PK is id (surrogate key); config_type has a UNIQUE constraint.
type AggregateConfig struct {
	ID                        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ConfigType                string    `gorm:"column:config_type;type:varchar(20);uniqueIndex;not null"`
	PollIntervalMS            *int      `gorm:"column:poll_interval_ms"` // nullable, only used by price config
	MinExchangesRequired      int       `gorm:"not null"`
	PriceStalenessThresholdMS int       `gorm:"column:price_staleness_threshold_ms;not null"`
	CreatedAt                 time.Time `gorm:"not null"`
	UpdatedAt                 time.Time `gorm:"not null"`
}

func (AggregateConfig) TableName() string {
	return "aggregate_configs"
}
