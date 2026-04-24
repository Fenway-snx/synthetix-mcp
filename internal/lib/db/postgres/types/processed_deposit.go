package types

import (
	"time"
)

// ProcessedDeposit tracks processed deposits to prevent duplicate processing.
// The combination of TxHash and LogIndex uniquely identifies a blockchain deposit event.
type ProcessedDeposit struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	TxHash    string    `gorm:"column:tx_hash;type:varchar(66);not null;uniqueIndex:idx_processed_deposits_nonce"`
	LogIndex  uint      `gorm:"column:log_index;not null;uniqueIndex:idx_processed_deposits_nonce"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName specifies the table name for ProcessedDeposit
func (ProcessedDeposit) TableName() string {
	return "processed_deposits"
}
