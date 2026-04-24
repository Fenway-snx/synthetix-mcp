package types

import (
	"time"
)

// OpenOrder represents an open order in the database
type OpenOrder struct {
	VenueOrderId            uint64     `gorm:"primaryKey;column:order_id" json:"order_id"`
	ClientOrderId           string     `gorm:"column:client_order_id" json:"client_order_id"`
	SubAccountID            int64      `gorm:"column:sub_account_id;index:idx_open_orders_subaccount_id" json:"sub_account_id"`
	Symbol                  string     `gorm:"column:symbol;index:idx_open_orders_symbol" json:"symbol"`
	Side                    int32      `gorm:"column:side" json:"side"` // 0: Short, 1: Long (matching OrderSide enum)
	Type                    int32      `gorm:"column:type" json:"type"` // Order type enum
	OriginalType            int32      `gorm:"column:original_type" json:"original_type"`
	Direction               int32      `gorm:"column:direction" json:"direction"`                                       // Direction enum
	TimeInForce             int32      `gorm:"column:time_in_force" json:"time_in_force"`                               // Time in force enum
	Price                   string     `gorm:"type:varchar(255);column:price" json:"price"`                             // Price as string
	PriceExponent           uint8      `gorm:"column:price_exponent" json:"price_exponent"`                             // Price exponent
	Quantity                string     `gorm:"type:varchar(255);column:quantity" json:"quantity"`                       // Quantity as string
	QuantityExponent        uint8      `gorm:"column:quantity_exponent;not null" json:"quantity_exponent"`              // Quantity exponent
	RemainingQuantity       string     `gorm:"type:varchar(255);column:remaining_quantity" json:"remaining_quantity"`   // Remaining quantity as string
	ReduceOnly              bool       `gorm:"column:reduce_only;default:false" json:"reduce_only"`                     // Reduce only flag
	PostOnly                bool       `gorm:"column:post_only;default:false" json:"post_only"`                         // Post only flag
	TakeProfitVenueOrderId  uint64     `gorm:"column:take_profit_order_id;default:0" json:"take_profit_order_id"`       // Take profit order ID
	TakeProfitClientOrderId string     `gorm:"column:take_profit_client_order_id" json:"take_profit_client_order_id"`   // Take profit client order ID
	StopLossVenueOrderId    uint64     `gorm:"column:stop_loss_order_id;default:0" json:"stop_loss_order_id"`           // Stop loss order ID
	StopLossClientOrderId   string     `gorm:"column:stop_loss_client_order_id" json:"stop_loss_client_order_id"`       // Stop loss client order ID
	PositionID              uint64     `gorm:"column:position_id;default:0" json:"position_id"`                         // Position ID
	TriggerPrice            string     `gorm:"type:varchar(255);column:trigger_price;default:'0'" json:"trigger_price"` // Trigger price as string
	TriggerPriceType        int32      `gorm:"column:trigger_price_type;default:0" json:"trigger_price_type"`           // Trigger price type enum
	IsActive                bool       `gorm:"column:is_active;default:false" json:"is_active"`                         // Whether conditional order is active (monitoring for trigger)
	ClosePosition           bool       `gorm:"column:close_position;default:false" json:"close_position"`               // Close entire position when triggered (TP/SL only)
	UpdatedAt               time.Time  `gorm:"column:updated_at; index; autoUpdateTime:false" json:"updated_at"`
	CreatedAt               *time.Time `gorm:"column:created_at; autoCreateTime:false" json:"created_at"`
	ExpiresAt               *time.Time `gorm:"column:expires_at;index" json:"expires_at"`
	PlacedAt                *time.Time `gorm:"column:placed_at" json:"placed_at"`
	ModifiedAt              *time.Time `gorm:"column:modified_at" json:"modified_at"`
	TradedAt                *time.Time `gorm:"column:traded_at" json:"traded_at"`
}

// TableName returns the table name for GORM
func (OpenOrder) TableName() string {
	return "open_orders"
}
