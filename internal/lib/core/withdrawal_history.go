package core

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

type WithdrawalHistory struct {
	Amount             shopspring_decimal.Decimal `gorm:"type:decimal(20,8); column:amount;" json:"amount"`
	AssetName          AssetName                  `gorm:"column:asset" json:"asset"`
	CreatedAt          time.Time                  `gorm:"column:created_at; index; autoCreateTime:false" json:"created_at"`
	DestinationAddress WalletAddress              `gorm:"column:destination_address" json:"destination_address"`
	FailureReason      *string                    `gorm:"column:failure_reason" json:"failure_reason"`          // Set if withdrawal failed (expired/denied/cancelled)
	Fee                shopspring_decimal.Decimal `gorm:"type:decimal(20,8); column:fee; default:5" json:"fee"` // Withdrawal fee in native asset
	Id                 int64                      `gorm:"primaryKey;column:id" json:"id"`
	OffchainId         OffchainWithdrawalId       `gorm:"column:offchain_withdrawal_id; index;" json:"offchain_withdrawal_id"`
	OnchainId          *OnchainWithdrawalId       `gorm:"column:onchain_withdrawal_id; index;"  json:"onchain_withdrawal_id"`
	RequestId          RequestId                  `gorm:"column:request_id;" json:"request_id"`
	Status             WithdrawalStatus           `gorm:"column:status;" json:"status"`
	SubAccountId       SubAccountId               `gorm:"column:sub_account_id; index" json:"sub_account_id"`
	TxHash             *string                    `gorm:"column:tx_hash" json:"tx_hash"`
}

func (WithdrawalHistory) TableName() string {
	return "withdrawal_history"
}

type WithdrawalHistory_View struct {
	WithdrawalHistory
}

func (WithdrawalHistory_View) TableName() string {
	return "withdrawal_history_view"
}
