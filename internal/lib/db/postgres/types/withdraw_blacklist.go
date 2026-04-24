package types

import "time"

// WithdrawBlacklist represents the withdraw_blacklist table.
// Wallets listed here are disputed by the relayer's withdraw watcher (DisputeWithdrawals cast on-chain).
type WithdrawBlacklist struct {
	ID            int64     `gorm:"primaryKey;column:id" json:"id"`
	WalletAddress string    `gorm:"column:wallet_address;uniqueIndex;size:42;not null" json:"wallet_address"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName specifies the table name for WithdrawBlacklist.
func (WithdrawBlacklist) TableName() string {
	return "withdraw_blacklist"
}
