package types

import (
	"time"

	"gorm.io/datatypes"
)

// Row in exchanges_futures: aggregate-mark venue settings for one exchange key.
// PK is id (surrogate key); key has a UNIQUE constraint.
type FuturesExchange struct {
	ID                uint64                                `gorm:"column:id;primaryKey;autoIncrement"`
	Key               string                                `gorm:"column:key;type:varchar(30);uniqueIndex;not null"`
	Name              string                                `gorm:"type:varchar(50);not null;"`
	Enabled           bool                                  `gorm:"not null"`
	AggregateWeight   int                                   `gorm:"not null"`
	URL               string                                `gorm:"column:url;type:varchar(255)"`
	CoinmetricsPrefix string                                `gorm:"type:varchar(50)"`
	CoinmetricsSuffix string                                `gorm:"type:varchar(50)"`
	SymbolPattern     string                                `gorm:"type:varchar(100)"`
	SymbolOverrides   datatypes.JSONType[map[string]string] `gorm:"type:jsonb"`
	Source            string                                `gorm:"type:varchar(50)"`
	APIKeyEnv         string                                `gorm:"type:varchar(100)"`
	CreatedAt         time.Time                             `gorm:"not null"`
	UpdatedAt         time.Time                             `gorm:"not null"`
}

func (FuturesExchange) TableName() string {
	return "exchanges_futures"
}

// Row in exchanges_spot: aggregate-index venue settings for one exchange key.
// PK is id (surrogate key); key has a UNIQUE constraint.
type SpotExchange struct {
	ID                   uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Key                  string    `gorm:"column:key;type:varchar(30);uniqueIndex;not null"`
	Name                 string    `gorm:"type:varchar(50);not null"`
	Enabled              bool      `gorm:"not null"`
	AggregateIndexWeight int       `gorm:"not null"`
	URL                  string    `gorm:"column:url;type:varchar(255)"`
	CoinmetricsPrefix    string    `gorm:"type:varchar(50)"`
	CoinmetricsSuffix    string    `gorm:"type:varchar(50)"`
	CreatedAt            time.Time `gorm:"not null"`
	UpdatedAt            time.Time `gorm:"not null"`
}

func (SpotExchange) TableName() string {
	return "exchanges_spot"
}
