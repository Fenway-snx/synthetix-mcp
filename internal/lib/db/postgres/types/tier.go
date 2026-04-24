package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/tier"
)

// Defines the available tier levels in the system (append-only).
// Each row is an immutable snapshot. The latest row per tier_id is the current definition.
type Tier struct {
	Id                 int64                      `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	TierId             tier.Id                    `gorm:"column:tier_id;size:50;not null" json:"tier_id"`
	TierType           tier.Type                  `gorm:"column:tier_type;size:20;not null" json:"tier_type"`
	TierName           tier.Name                  `gorm:"column:tier_name;size:100;not null" json:"tier_name"`
	MinTradeVolume     shopspring_decimal.Decimal `gorm:"type:decimal(30,2);column:min_trade_volume;not null;default:0" json:"min_trade_volume"`
	MakerFeeRate       shopspring_decimal.Decimal `gorm:"type:decimal(10,6);column:maker_fee_rate;not null" json:"maker_fee_rate"`
	TakerFeeRate       shopspring_decimal.Decimal `gorm:"type:decimal(10,6);column:taker_fee_rate;not null" json:"taker_fee_rate"`
	MaxBorrowCapacity  shopspring_decimal.Decimal `gorm:"type:decimal(30,2);column:max_borrow_capacity;not null" json:"max_borrow_capacity"`
	MaxOrdersPerMarket int64                      `gorm:"column:max_orders_per_market;not null" json:"max_orders_per_market"`
	MaxSubAccounts     int64                      `gorm:"column:max_sub_accounts;not null;default:1" json:"max_sub_accounts"`
	MaxTotalOrders     int64                      `gorm:"column:max_total_orders;not null" json:"max_total_orders"`
	CreatedAt          time.Time                  `gorm:"column:created_at;not null" json:"created_at"`
}

func (Tier) TableName() string { return "tiers" }

type Tier_View struct {
	Tier
}

func (Tier_View) TableName() string { return "tiers_view" }
