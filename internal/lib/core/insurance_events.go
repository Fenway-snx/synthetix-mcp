package core

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// InsurancePositionIncreasedEvent is published when a position is increased
// This resets the exclusion period for wick insurance
type InsurancePositionIncreasedEvent struct {
	SubAccountId SubAccountId               `json:"sub_account_id"`
	Symbol       string                     `json:"symbol"`
	OldQuantity  shopspring_decimal.Decimal `json:"old_quantity"`
	NewQuantity  shopspring_decimal.Decimal `json:"new_quantity"`
	Timestamp    time.Time                  `json:"timestamp"`
}

// InsuranceProtectionActivatedEvent is published when wick insurance protection is activated
type InsuranceProtectionActivatedEvent struct {
	SubAccountId SubAccountId `json:"sub_account_id"`
	Timestamp    time.Time    `json:"timestamp"`    // TODO: This should be called triggered at or started at
	ExpiredTime  time.Time    `json:"expired_time"` // TODO: Keeping the same pattern happening here but this name does not feel right
}

// InsuranceProtectionCompletedEvent is published when wick insurance protection completes
type InsuranceProtectionCompletedEvent struct {
	SubAccountId SubAccountId `json:"sub_account_id"`
	ProtectionID int64        `json:"protection_id"`
	ExpiredTime  time.Time    `json:"expired_time"`
	Timestamp    time.Time    `json:"timestamp"`
}
