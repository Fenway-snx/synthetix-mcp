package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

type PositionCloseReason string

const (
	PositionCloseReasonClose       PositionCloseReason = "close"
	PositionCloseReasonFlip        PositionCloseReason = "flip"
	PositionCloseReasonLiquidation PositionCloseReason = "liquidation"
)

// PositionHistory represents an append-only event log for position lifecycle events.
// Each row captures a snapshot of the position at the time of an event (open, update, close).
// A position's full history is reconstructed by querying all rows with the same position_id ordered by event_at.
type PositionHistory struct {
	ID                       uint64                     `gorm:"primaryKey;column:id" json:"id"`
	PositionId               uint64                     `gorm:"column:position_id;index:idx_position_history_position_id" json:"position_id"`
	SubAccountID             int64                      `gorm:"column:sub_account_id;index:idx_position_history_subaccount_event,priority:1" json:"sub_account_id"`
	Symbol                   string                     `gorm:"column:symbol" json:"symbol"`
	Side                     int32                      `gorm:"column:side" json:"side"`
	Action                   int32                      `gorm:"column:action" json:"action"`
	EntryPrice               shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:entry_price" json:"entry_price"`
	Quantity                 shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:quantity" json:"quantity"`
	UPNL                     shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:upnl;default:'0'" json:"upnl"`
	UsedMargin               shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:used_margin;default:'0'" json:"used_margin"`
	MaintenanceMargin        shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:maintenance_margin;default:'0'" json:"maintenance_margin"`
	LiquidationPrice         shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:liquidation_price;default:'0'" json:"liquidation_price"`
	NetPositionFundingPnl    shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:net_position_funding_pnl;default:'0'" json:"net_position_funding_pnl"`
	AccumulatedRealizedPnl   shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:accumulated_realized_pnl;default:'0'" json:"accumulated_realized_pnl"`
	AccumulatedFees          shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:accumulated_fees;default:'0'" json:"accumulated_fees"`
	AccumulatedCloseValue    shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:accumulated_close_value;default:'0'" json:"accumulated_close_value"`
	AccumulatedCloseQuantity shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:accumulated_close_quantity;default:'0'" json:"accumulated_close_quantity"`
	ClosePrice               shopspring_decimal.Decimal `gorm:"type:numeric(20,8);column:close_price" json:"close_price"`
	CloseReason              PositionCloseReason        `gorm:"column:close_reason;type:varchar(50)" json:"close_reason"`
	Leverage                 *uint32                    `gorm:"column:leverage" json:"leverage"`
	TradeId                  int64                      `gorm:"column:trade_id" json:"trade_id"`
	ClosedAt                 *time.Time                 `gorm:"column:closed_at" json:"closed_at"`
	CreatedAt                *time.Time                 `gorm:"column:created_at;autoCreateTime:false" json:"created_at"`
	ModifiedAt               *time.Time                 `gorm:"column:modified_at" json:"modified_at"`
	EventAt                  time.Time                  `gorm:"column:event_at;index:idx_position_history_subaccount_event,priority:2" json:"event_at"`
}

// TableName returns the table name for GORM
func (PositionHistory) TableName() string {
	return "position_history"
}
