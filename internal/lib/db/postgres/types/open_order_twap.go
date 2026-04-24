package types

import "time"

// OpenOrderTwap stores TWAP parent execution state in table open_order_twap.
// order_id is the parent TWAP row in open_orders (FK, ON DELETE CASCADE).
type OpenOrderTwap struct {
	OrderID           uint64    `gorm:"primaryKey;column:order_id" json:"order_id"`
	SubAccountID      int64     `gorm:"column:sub_account_id;index:idx_open_order_twap_sub_account_id" json:"sub_account_id"`
	ChunkIntervalMs   int64     `gorm:"column:chunk_interval_ms" json:"chunk_interval_ms"`
	ChunkQuantity     string    `gorm:"type:varchar(255);column:chunk_quantity" json:"chunk_quantity"`
	ChunksTotal       int       `gorm:"column:chunks_total" json:"chunks_total"`
	Direction         int32     `gorm:"column:direction" json:"direction"`
	PriceLimit        string    `gorm:"type:varchar(255);column:price_limit" json:"price_limit"`
	ReduceOnly        bool      `gorm:"column:reduce_only" json:"reduce_only"`
	Side              int32     `gorm:"column:side" json:"side"`
	Source            string    `gorm:"column:source" json:"source"`
	StartedAtMs       int64     `gorm:"column:started_at_ms" json:"started_at_ms"`
	Symbol            string    `gorm:"column:symbol" json:"symbol"`
	TotalQuantity     string    `gorm:"type:varchar(255);column:total_quantity" json:"total_quantity"`
	ChunksFilled      int       `gorm:"column:chunks_filled" json:"chunks_filled"`
	ChunksSubmitted   int       `gorm:"column:chunks_submitted" json:"chunks_submitted"`
	FilledNotional    string    `gorm:"type:varchar(255);column:filled_notional" json:"filled_notional"`
	NextChunkAtMs     int64     `gorm:"column:next_chunk_at_ms" json:"next_chunk_at_ms"`
	QuantityFilled    string    `gorm:"type:varchar(255);column:quantity_filled" json:"quantity_filled"`
	QuantitySubmitted string    `gorm:"type:varchar(255);column:quantity_submitted" json:"quantity_submitted"`
	State             int32     `gorm:"column:state" json:"state"`
	TotalFees         string    `gorm:"type:varchar(255);column:total_fees" json:"total_fees"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime:false" json:"updated_at"`
}

// TableName returns the table name for GORM.
func (OpenOrderTwap) TableName() string {
	return "open_order_twap"
}
