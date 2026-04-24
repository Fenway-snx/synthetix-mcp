package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

type Liquidation struct {
	ID                         int64                      `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID               int64                      `gorm:"column:sub_account_id;index:idx_liquidations_sub_account_id" json:"sub_account_id"`
	Reason                     string                     `gorm:"column:reason;type:varchar(50)" json:"reason"`
	Stage                      string                     `gorm:"column:stage;type:varchar(50)" json:"stage"`
	TotalPositions             int64                      `gorm:"column:total_positions;default:0" json:"total_positions"`
	TotalCollateralSymbols     int64                      `gorm:"column:total_collateral_symbols;default:0" json:"total_collateral_symbols"`
	PreLiquidationMargin       shopspring_decimal.Decimal `gorm:"column:pre_liquidation_margin;default:0" json:"pre_liquidation_margin"`
	PreLiquidationAccountValue shopspring_decimal.Decimal `gorm:"column:pre_liquidation_account_value;default:0" json:"pre_liquidation_account_value"`
	BadDebt                    shopspring_decimal.Decimal `gorm:"column:bad_debt;default:0" json:"bad_debt"`
	CreatedAt                  time.Time                  `gorm:"column:created_at;index:idx_liquidations_created_at" json:"created_at"`
}

func (Liquidation) TableName() string {
	return "liquidations"
}
