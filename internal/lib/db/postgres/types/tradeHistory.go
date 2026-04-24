package types

import (
	"time"
)

type TradeHistory struct {
	ID                          uint64    `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID                int64     `gorm:"column:sub_account_id;index:idx_trade_history_subaccount_time,priority:1;index" json:"sub_account_id"`
	VenueOrderId                uint64    `gorm:"column:order_id;index" json:"order_id"`
	ClientOrderId               string    `gorm:"column:client_order_id" json:"client_order_id"`
	Symbol                      string    `gorm:"column:symbol;index:idx_trade_history_symbol_time,priority:1;index:idx_trade_history_symbol_time_fill,priority:1;index" json:"symbol"`                                                          // trading pair, e.g., "BTCUSD", "ETHUSDT"
	OrderType                   string    `gorm:"column:order_type" json:"order_type"`                                                                                                                                                           // e.g., "Limit", "Market"
	Direction                   string    `gorm:"column:direction" json:"direction"`                                                                                                                                                             // e.g., "Long", "Short"
	FilledPrice                 string    `gorm:"column:filled_price" json:"filled_price"`                                                                                                                                                       // actual execution price
	FilledQuantity              string    `gorm:"column:filled_quantity" json:"filled_quantity"`                                                                                                                                                 // actual execution quantity
	FilledValue                 string    `gorm:"column:filled_value" json:"filled_value"`                                                                                                                                                       // filled_price * filled_quantity
	FillType                    string    `gorm:"column:fill_type;index:idx_trade_history_symbol_time_fill,priority:3" json:"fill_type"`                                                                                                         // FillTypeMaker or FillTypeTaker (see lib/types constants)
	Fee                         string    `gorm:"column:fee" json:"fee"`                                                                                                                                                                         // trading fee
	ClosedPNL                   string    `gorm:"column:closed_pnl" json:"closed_pnl"`                                                                                                                                                           // realized PnL from position closure
	MarkPrice                   string    `gorm:"type:text;column:mark_price;default:'0'" json:"mark_price"`                                                                                                                                     // mark price at time of trade (TEXT to support long decimal strings)
	EntryPrice                  string    `gorm:"type:text;column:entry_price;default:'0'" json:"entry_price"`                                                                                                                                   // position (average) entry price at time of trade (TEXT to support long decimal strings)
	TradedAt                    time.Time `gorm:"column:traded_at;index:idx_trade_history_symbol_time,priority:2;index:idx_trade_history_symbol_time_fill,priority:2;index:idx_trade_history_subaccount_time,priority:2;index" json:"traded_at"` // execution timestamp
	TriggeredByLiquidation      bool      `gorm:"column:triggered_by_liquidation;default:false" json:"triggered_by_liquidation"`
	FeeRate                     string    `gorm:"column:fee_rate;default:'0'" json:"fee_rate"`
	LiquidationClearanceFee     string    `gorm:"column:liquidation_clearance_fee;default:'0'" json:"liquidation_clearance_fee"`           // fee charged on liquidation (separate from trading fee)
	LiquidationClearanceFeeRate string    `gorm:"column:liquidation_clearance_fee_rate;default:'0'" json:"liquidation_clearance_fee_rate"` // rate used to calculate liquidation clearance fee
	PostOnly                    bool      `gorm:"column:post_only;default:false" json:"post_only"`
	ReduceOnly                  bool      `gorm:"column:reduce_only;default:false" json:"reduce_only"`
	Source                      string    `gorm:"column:source;type:varchar(100)" json:"source,omitempty"`
	TradeId                     int64     `gorm:"column:trade_id;index:idx_trade_history_trade_id;default:0" json:"trade_id"`
	LiquidationReason           string    `gorm:"column:liquidation_reason;type:varchar(50);default:''" json:"liquidation_reason,omitempty"`
	LiquidationId               int64     `gorm:"column:liquidation_id;default:0" json:"liquidation_id,omitempty"`
}

// TableName returns the table name for GORM
func (TradeHistory) TableName() string {
	return "trade_history"
}
