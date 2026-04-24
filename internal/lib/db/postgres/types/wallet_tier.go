package types

import (
	"time"

	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/tier"
)

// Maps each wallet to its assigned tier (append-only).
// Each row is an immutable snapshot. The latest row per wallet_address is the current assignment.
type WalletTier struct {
	Id            int64     `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	WalletAddress string    `gorm:"column:wallet_address;size:42;not null" json:"wallet_address"`
	TierId        tier.Id   `gorm:"column:tier_id;size:50;not null" json:"tier_id"`
	CreatedAt     time.Time `gorm:"column:created_at;not null" json:"created_at"`
}

func (WalletTier) TableName() string { return "wallet_tiers" }
