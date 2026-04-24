package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

type FeeRateHistory struct {
	ID            uint64                     `gorm:"primaryKey; column:id"`
	MakerRate     shopspring_decimal.Decimal `gorm:"column:maker_rate; type:decimal(10,6)"`
	SubAccountID  int64                      `gorm:"column:sub_account_id; index"`
	TakerRate     shopspring_decimal.Decimal `gorm:"column:taker_rate; type:decimal(10,6)"`
	TierId        string                     `gorm:"column:tier_id"`
	TierName      string                     `gorm:"column:tier_name"`
	TradeVolume   shopspring_decimal.Decimal `gorm:"column:trade_volume;"`
	UpdatedAt     time.Time                  `gorm:"column:updated_at; index; autoUpdateTime:false"`
	WalletAddress string                     `gorm:"column:wallet_address"`
}

func (FeeRateHistory) TableName() string {
	return "fee_rate_history"
}
