package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

type OrderHistory struct {
	ID                     int64                       `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID           int64                       `gorm:"column:sub_account_id;index:idx_order_history_sub_account_id_updated_at,priority:1" json:"sub_account_id"`
	VenueOrderId           uint64                      `gorm:"column:order_id;index" json:"order_id"`
	ClientOrderId          string                      `gorm:"column:client_order_id" json:"client_order_id"`
	TPVenueOrderId         uint64                      `gorm:"column:tp_order_id" json:"tp_order_id"`
	TPClientOrderId        string                      `gorm:"column:tp_client_order_id" json:"tp_client_order_id"`
	SLVenueOrderId         uint64                      `gorm:"column:sl_order_id" json:"sl_order_id"`
	SLClientOrderId        string                      `gorm:"column:sl_client_order_id" json:"sl_client_order_id"`
	Symbol                 string                      `gorm:"column:symbol" json:"symbol"`
	Side                   string                      `gorm:"column:side" json:"side"`
	Type                   string                      `gorm:"column:type" json:"type"`           // make first letter uppercase
	Direction              string                      `gorm:"column:direction" json:"direction"` // make first letter uppercase of each word
	Price                  string                      `gorm:"column:price" json:"price"`         // if order is market return "Market" else return price
	Quantity               string                      `gorm:"column:quantity" json:"quantity"`
	Value                  string                      `gorm:"column:value" json:"value"` // if order is market return "Market" else return value
	FilledQuantity         string                      `gorm:"column:filled_quantity" json:"filled_quantity"`
	FilledPrice            string                      `gorm:"column:filled_price" json:"filled_price"`
	FilledValue            string                      `gorm:"column:filled_value" json:"filled_value"`
	Status                 string                      `gorm:"column:status" json:"status"`
	TriggeredByLiquidation bool                        `gorm:"column:triggered_by_liquidation;default:false" json:"triggered_by_liquidation"`
	ReduceOnly             bool                        `gorm:"column:reduce_only;default:false" json:"reduce_only"`
	PostOnly               bool                        `gorm:"column:post_only;default:false" json:"post_only"`
	TriggerPrice           *shopspring_decimal.Decimal `gorm:"column:trigger_price;type:decimal(20,8)" json:"trigger_price"`
	TriggerPriceType       *int32                      `gorm:"column:trigger_price_type" json:"trigger_price_type"`
	TimeInForce            int32                       `gorm:"column:time_in_force;default:0" json:"time_in_force"`
	CancelReason           string                      `gorm:"column:cancel_reason" json:"cancel_reason,omitempty"`
	Source                 string                      `gorm:"column:source;type:varchar(100)" json:"source,omitempty"`
	CancelledAt            *time.Time                  `gorm:"column:cancelled_at" json:"cancelled_at"`
	CreatedAt              *time.Time                  `gorm:"column:created_at; autoCreateTime:false" json:"created_at"`
	ExpiresAt              *time.Time                  `gorm:"column:expires_at" json:"expires_at"`
	ModifiedAt             *time.Time                  `gorm:"column:modified_at" json:"modified_at"`
	PlacedAt               *time.Time                  `gorm:"column:placed_at" json:"placed_at"`
	RejectedAt             *time.Time                  `gorm:"column:rejected_at" json:"rejected_at"`
	TradedAt               *time.Time                  `gorm:"column:traded_at" json:"traded_at"`
	UpdatedAt              time.Time                   `gorm:"column:updated_at;index;index:idx_order_history_sub_account_id_updated_at,priority:2,sort:desc;autoUpdateTime:false" json:"updated_at"`
}

// TableName returns the table name for GORM
func (OrderHistory) TableName() string {
	return "order_history"
}
