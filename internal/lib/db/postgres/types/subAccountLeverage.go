package types

import "time"

type SubAccountLeverage struct {
	ID           int       `gorm:"primaryKey"`
	SubAccountID int64     `gorm:"not null; index; uniqueIndex:idx_subaccount_symbol_lev,unique;"`
	Symbol       string    `gorm:"not null; uniqueIndex:idx_subaccount_symbol_lev,unique;"`
	Leverage     uint32    `gorm:"not null;"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`

	// Relations
	SubAccount *SubAccount `gorm:"foreignKey:SubAccountId"`
}

// TableName specifies the table name for SubAccountLeverage
func (SubAccountLeverage) TableName() string {
	return "sub_account_leverages"
}
