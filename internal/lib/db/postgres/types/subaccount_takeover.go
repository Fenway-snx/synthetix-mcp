package types

import (
	"encoding/json"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

type TakeoverPosition struct {
	Symbol          string `json:"symbol"`
	Quantity        string `json:"quantity"`
	Side            string `json:"side"`
	EntryPrice      string `json:"entry_price"`
	MarkPrice       string `json:"mark_price"`
	SLPSubAccountID int64  `json:"slp_sub_account_id"`
	TradeID         int64  `json:"trade_id"`
}

type TakeoverCollateral struct {
	Symbol          string `json:"symbol"`
	Quantity        string `json:"quantity"`
	SLPSubAccountID int64  `json:"slp_sub_account_id"`
}

type SubaccountTakeover struct {
	ID                      int64                      `gorm:"primaryKey;column:id" json:"id"`
	LiquidationID           int64                      `gorm:"column:liquidation_id;index:idx_subaccount_takeovers_liquidation_id" json:"liquidation_id"`
	SubAccountID            int64                      `gorm:"column:sub_account_id;index:idx_subaccount_takeovers_sub_account_id" json:"sub_account_id"`
	Reason                  string                     `gorm:"column:reason;type:varchar(50)" json:"reason"`
	Positions               json.RawMessage            `gorm:"column:positions;type:jsonb;default:'[]'" json:"positions"`
	Collaterals             json.RawMessage            `gorm:"column:collaterals;type:jsonb;default:'[]'" json:"collaterals"`
	PreTakeoverMargin       shopspring_decimal.Decimal `gorm:"column:pre_takeover_margin;default:0" json:"pre_takeover_margin"`
	PreTakeoverAccountValue shopspring_decimal.Decimal `gorm:"column:pre_takeover_account_value;default:0" json:"pre_takeover_account_value"`
	BadDebt                 shopspring_decimal.Decimal `gorm:"column:bad_debt;default:0" json:"bad_debt"`
	CreatedAt               time.Time                  `gorm:"column:created_at;index:idx_subaccount_takeovers_created_at" json:"created_at"`
}

func (SubaccountTakeover) TableName() string {
	return "subaccount_takeovers"
}
