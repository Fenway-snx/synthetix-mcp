package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

const (
	TransactionId_Zero = TransactionId(0)
)

type SubAccountTransaction struct {
	ID             int                        `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID   int64                      `gorm:"column:sub_account_id" json:"sub_account_id"`
	ToSubAccountID *int64                     `gorm:"column:to_sub_account_id" json:"to_sub_account_id"`
	Action         string                     `gorm:"column:action" json:"action"`
	Status         int                        `gorm:"column:status" json:"status"`
	Amount         shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:amount" json:"amount"`
	Metadata       datatypes.JSON             `gorm:"type:jsonb;column:metadata" json:"metadata"`
	CreatedAt      time.Time                  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time                  `gorm:"column:updated_at" json:"updated_at"`

	// Relations
	SubAccount   *SubAccount `gorm:"foreignKey:SubAccountID" json:"sub_account,omitempty"`
	ToSubAccount *SubAccount `gorm:"foreignKey:ToSubAccountID" json:"to_sub_account,omitempty"`
}

// TableName specifies the table name for SubAccountTransaction
func (SubAccountTransaction) TableName() string {
	return "sub_account_transactions"
}

type SubAccountTransaction_Metadata_Deposit struct {
	CollateralName        string `json:"collateral"`
	DepositedToMaster     bool   `json:"deposited_to_master"`
	IsNewMasterAccount    bool   `json:"is_new_master_account"`
	MarketName            string `json:"market"`
	ReceiverWalletAddress string `json:"receiver_wallet_address"`
	RequestedSubAccountID int64  `json:"requested_subaccount_id"`
	TargetSubAccountID    int64  `json:"target_subaccount_id"`
	TransactionType       string `json:"type"`
	TxHash                string `json:"tx_hash"`
}

type TransactionType = string

const (
	TransactionType_Deposit TransactionType = "deposit"
)

type TransactionId = int
