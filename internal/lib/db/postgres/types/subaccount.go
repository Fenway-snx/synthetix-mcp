package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

var (
	errAtLeastOneOfMarketIdOrCollateralIdMustBeSet = errors.New("at least one of MarketID or CollateralID must be set")
)

// StringArray custom type for PostgreSQL array handling
type StringArray []string

// Scan implements the sql.Scanner interface for database reads
func (a *StringArray) Scan(value any) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		*a = []string{}
		return nil
	}

	return json.Unmarshal(bytes, a)
}

// Value implements the driver.Valuer interface for database writes
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

// CollateralHaircutTier represents a single tier in the collateral haircut system
type CollateralHaircutTier struct {
	ID                      int64                       `gorm:"primaryKey;column:id" json:"id"`
	CollateralID            int64                       `gorm:"column:collateral_id;index:idx_collateral_tier,unique" json:"collateral_id"`
	TierName                string                      `gorm:"column:tier_name;index:idx_collateral_tier,unique" json:"tier_name"`
	MinAmountUSDT           shopspring_decimal.Decimal  `gorm:"type:decimal(20,8);column:min_amount_usdt" json:"min_amount_usdt"`
	MaxAmountUSDT           *shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:max_amount_usdt" json:"max_amount_usdt"`
	CollateralValueRatio    shopspring_decimal.Decimal  `gorm:"type:decimal(6,4);column:collateral_value_ratio" json:"collateral_value_ratio"`
	CollateralValueHaircut  shopspring_decimal.Decimal  `gorm:"type:decimal(6,4);column:collateral_value_haircut" json:"collateral_value_haircut"`
	CollateralValueAddition shopspring_decimal.Decimal  `gorm:"type:decimal(20,8);column:collateral_value_addition;default:0" json:"collateral_value_addition"`
	Status                  int                         `gorm:"column:status;default:1" json:"status"`
	CreatedAt               time.Time                   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt               time.Time                   `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the table name for GORM
func (CollateralHaircutTier) TableName() string {
	return "collateral_haircut_tiers"
}

// SubAccountType represents the type of a subaccount
type SubAccountType int32

// SubAccount Type Constants
const (
	// TODO: Either start at 1 or set 0 as unknown
	AccountTypeNormal SubAccountType = 0 // Regular trading accounts (default)
	AccountTypeSLP    SubAccountType = 1 // Synthetix Liquidity Provider accounts
	AccountTypeFee    SubAccountType = 2
)

// ValidAccountTypes contains all valid account types for validation
var ValidAccountTypes = []SubAccountType{
	AccountTypeNormal,
	AccountTypeSLP,
	AccountTypeFee,
}

// String returns the string representation of the SubAccountType
func (s SubAccountType) String() string {
	switch s {
	case AccountTypeNormal:
		return "normal"
	case AccountTypeSLP:
		return "slp"
	case AccountTypeFee:
		return "fee"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// MarshalJSON implements json.Marshaler
func (s SubAccountType) MarshalJSON() ([]byte, error) {
	return json.Marshal(int32(s))
}

// UnmarshalJSON implements json.Unmarshaler
func (s *SubAccountType) UnmarshalJSON(data []byte) error {
	var i int32
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	*s = SubAccountType(i)
	return nil
}

// IsValid checks if the SubAccountType is valid
func (s SubAccountType) IsValid() bool {
	switch s {
	case
		AccountTypeFee,
		AccountTypeNormal,
		AccountTypeSLP:
		return true

	default:
		return false
	}
}

// Scan implements the sql.Scanner interface for database operations
func (s *SubAccountType) Scan(value any) error {
	if value == nil {
		*s = AccountTypeNormal // Default to normal if NULL
		return nil
	}
	switch v := value.(type) {
	case int64:
		*s = SubAccountType(v)
	case int32:
		*s = SubAccountType(v)
	case int:
		*s = SubAccountType(v)
	default:
		return fmt.Errorf("cannot scan %T into SubAccountType", value)
	}
	return nil
}

// Value implements the driver.Valuer interface for database operations
func (s SubAccountType) Value() (driver.Value, error) {
	return int64(s), nil
}

// SubAccount represents the sub_accounts table
type SubAccount struct {
	ID                          int64                   `gorm:"primaryKey;column:id" json:"id"`
	Name                        string                  `gorm:"not null;column:name" json:"name"`
	WalletAddress               string                  `gorm:"column:wallet_address;index" json:"wallet_address"`
	MasterID                    int64                   `gorm:"column:master_id" json:"master_id"`
	Status                      v4grpc.SubaccountStatus `gorm:"column:status;type:integer;default:1;not null" json:"status"`
	Type                        SubAccountType          `gorm:"column:type;type:integer;default:0;not null" json:"type"`
	DeadManSwitchActive         bool                    `gorm:"column:dead_man_switch_active;default:false;not null" json:"dead_man_switch_active"`
	DeadManSwitchGeneration     uint64                  `gorm:"column:dead_man_switch_generation;type:bigint;default:0;not null" json:"dead_man_switch_generation"`
	DeadManSwitchTimeoutSeconds int64                   `gorm:"column:dead_man_switch_timeout_seconds;default:0;not null" json:"dead_man_switch_timeout_seconds"`
	DeadManSwitchTriggerTime    *time.Time              `gorm:"column:dead_man_switch_trigger_time" json:"dead_man_switch_trigger_time"`
	CreatedAt                   time.Time               `gorm:"column:created_at" json:"created_at"`
	UpdatedAt                   time.Time               `gorm:"column:updated_at" json:"updated_at"`
}

// TableName specifies the table name for SubAccount
func (SubAccount) TableName() string {
	return "sub_accounts"
}

// IsActive returns true if the subaccount status is ACTIVE
func (s *SubAccount) IsActive() bool {
	return s.Status == v4grpc.SubaccountStatus_ACCOUNT_STATUS_ACTIVE
}

// IsFrozen returns true if the subaccount status is FROZEN
func (s *SubAccount) IsFrozen() bool {
	return s.Status == v4grpc.SubaccountStatus_ACCOUNT_STATUS_FROZEN
}

// SubAccountMargin represents the sub_account_margins table (1:1 with sub_accounts)
type SubAccountMargin struct {
	SubAccountID         int64                      `gorm:"primaryKey;column:sub_account_id" json:"sub_account_id"`
	AdjustedAccountValue shopspring_decimal.Decimal `gorm:"column:adjusted_account_value;type:decimal(20,8);default:0" json:"adjusted_account_value"`
	AccountValue         shopspring_decimal.Decimal `gorm:"column:account_value;type:decimal(20,8);default:0" json:"account_value"`
	AvailableMargin      shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:available_margin;default:0" json:"available_margin"`
	CumulativeIMR        shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:cumulative_imr;default:0" json:"cumulative_imr"`
	CumulativeMMR        shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:cumulative_mmr;default:0" json:"cumulative_mmr"`
	CumulativeRMR        shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:cumulative_rmr;default:0" json:"cumulative_rmr"`
	RealizedPnl          shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:realized_pnl;default:0" json:"realized_pnl"`
	UnrealizedPnl        shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:unrealized_pnl;default:0" json:"unrealized_pnl"`
	Debt                 shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:debt;default:0" json:"debt"`
	Withdrawable         shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:withdrawable;default:0" json:"withdrawable"`
	UpdatedAt            time.Time                  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName specifies the table name for SubAccountMargin
func (SubAccountMargin) TableName() string {
	return "sub_account_margins"
}

// TODO: We should probably rename this one SNX-4925
// Collateral represents the collaterals table
type Collateral struct {
	ID                int64                      `gorm:"primaryKey;column:id" json:"id"`
	Collateral        string                     `gorm:"not null;column:collateral" json:"collateral"`
	Market            string                     `gorm:"not null;column:market" json:"market"`
	DepositCap        shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:deposit_cap;default:0" json:"deposit_cap"`
	LLTV              shopspring_decimal.Decimal `gorm:"type:decimal(5,2);column:lltv;not null" json:"lltv"` // Liquidation LTV (trigger for auto-exchange)
	LTV               shopspring_decimal.Decimal `gorm:"type:decimal(5,2);column:ltv;not null" json:"ltv"`   // Loan-to-Value ratio (target after auto-exchange)
	QuantityPrecision int64                      `gorm:"column:quantity_precision;not null" json:"quantity_precision"`
	WithdrawFee       shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:withdraw_fee" json:"withdraw_fee"`
	CreatedAt         time.Time                  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time                  `gorm:"column:updated_at" json:"updated_at"`

	// Relationship
	CollateralHaircutTiers []CollateralHaircutTier `gorm:"foreignKey:CollateralID" json:"collateral_haircut_tiers,omitempty"`
}

// TableName specifies the table name for Collateral
func (Collateral) TableName() string {
	return "collaterals"
}

// SubAccountSLP represents the sub_account_slps table
// This table maps markets and/or collaterals to SLP subaccounts
type SubAccountSLP struct {
	ID           int       `gorm:"primaryKey;column:id" json:"id"`
	CollateralID *int64    `gorm:"column:collateral_id;index:idx_slp_collateral_subaccount,unique" json:"collateral_id"`
	MarketID     *int64    `gorm:"column:market_id;index:idx_slp_market_subaccount,unique" json:"market_id"`
	SubAccountID int64     `gorm:"column:sub_account_id;not null;index:idx_slp_market_subaccount,unique;index:idx_slp_collateral_subaccount,unique" json:"sub_account_id"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`

	// Relations
	Market     *Market     `gorm:"foreignKey:MarketID" json:"market,omitempty"`
	Collateral *Collateral `gorm:"foreignKey:CollateralID" json:"collateral,omitempty"`
	SubAccount *SubAccount `gorm:"foreignKey:SubAccountID" json:"sub_account,omitempty"`
}

// TableName specifies the table name for SubAccountSLP
func (SubAccountSLP) TableName() string {
	return "sub_account_slps"
}

// Validate ensures at least one of MarketID or CollateralID is set
func (s *SubAccountSLP) Validate() error {
	if s.MarketID == nil && s.CollateralID == nil {
		return errAtLeastOneOfMarketIdOrCollateralIdMustBeSet
	}
	return nil
}

// SubAccountCollateral represents the sub_account_collaterals table
type SubAccountCollateral struct {
	ID                      int                        `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID            int64                      `gorm:"column:sub_account_id;index:uidx_subaccount_collateral,unique,priority:1" json:"sub_account_id"`
	CollateralID            int64                      `gorm:"column:collateral_id;index:uidx_subaccount_collateral,unique,priority:2" json:"collateral_id"`
	Quantity                shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:quantity" json:"quantity"`
	CollateralValue         shopspring_decimal.Decimal `gorm:"column:collateral_value;type:decimal(20,8);default:0" json:"collateral_value"`
	AdjustedCollateralValue shopspring_decimal.Decimal `gorm:"column:adjusted_collateral_value;type:decimal(20,8);default:0" json:"adjusted_collateral_value"`
	HaircutRate             shopspring_decimal.Decimal `gorm:"column:haircut_rate;type:decimal(20,8);default:0" json:"haircut_rate"`
	HaircutAdjustment       shopspring_decimal.Decimal `gorm:"column:haircut_adjustment;type:decimal(20,8);default:0" json:"haircut_adjustment"`
	PendingWithdrawalAmount shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:pending_withdrawal_amount;default:0" json:"pending_withdrawal_amount"`
	WithdrawableAmount      shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:withdrawable_amount;default:0" json:"withdrawable_amount"`
	Price                   shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:price;default:0" json:"price"`
	CalculatedAt            time.Time                  `gorm:"column:calculated_at" json:"calculated_at"`

	// Relations
	SubAccount *SubAccount `gorm:"foreignKey:SubAccountID" json:"sub_account,omitempty"`
	Collateral *Collateral `gorm:"foreignKey:CollateralID" json:"collateral,omitempty"`
}

// TableName specifies the table name for SubAccountCollateral
func (SubAccountCollateral) TableName() string {
	return "sub_account_collaterals"
}

// SubAccountHistory represents the sub_account_histories table
type SubAccountHistory struct {
	ID            int                        `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID  int64                      `gorm:"column:sub_account_id" json:"sub_account_id"`
	Pnl           shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:pnl" json:"pnl"`
	RealizedPnl   shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:realized_pnl" json:"realized_pnl"`
	UnrealizedPnl shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:unrealized_pnl" json:"unrealized_pnl"`
	Balance       shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:balance" json:"balance"`
	CreatedTime   time.Time                  `gorm:"column:created_time" json:"created_time"`

	// Relations
	SubAccount *SubAccount `gorm:"foreignKey:SubAccountID" json:"sub_account,omitempty"`
}

// TableName specifies the table name for SubAccountHistory
func (SubAccountHistory) TableName() string {
	return "sub_account_histories"
}

// SubAccountDelegation represents the sub_account_delegations table for RBAC
type SubAccountDelegation struct {
	ID              int64       `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID    int64       `gorm:"column:sub_account_id;not null;index;uniqueIndex:idx_subaccount_delegate,unique" json:"sub_account_id"`
	DelegateAddress string      `gorm:"column:delegate_address;size:42;not null;index;uniqueIndex:idx_subaccount_delegate,unique" json:"delegate_address"`
	AddedBy         *string     `gorm:"column:added_by;size:42" json:"added_by"`
	Permissions     StringArray `gorm:"column:permissions;type:jsonb" json:"permissions"`
	ExpiresAt       *time.Time  `gorm:"column:expires_at;index" json:"expires_at"`
	CreatedAt       time.Time   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time   `gorm:"column:updated_at" json:"updated_at"`

	// Relations
	SubAccount *SubAccount `gorm:"foreignKey:SubAccountID" json:"sub_account,omitempty"`
}

// TableName specifies the table name for SubAccountDelegation
func (SubAccountDelegation) TableName() string {
	return "sub_account_delegations"
}

// Insurance status constants
const (
	// InsuranceStatusSubscribed indicates an active insurance subscription without active protection
	// This is the default state for accounts with insurance. Only one SUBSCRIBED row exists per account.
	InsuranceStatusSubscribed = 1

	// InsuranceStatusActiveProtection indicates wick insurance protection is currently active
	// Created when MMR breach occurs and account is within protection eligibility.
	// Only one ACTIVE_PROTECTION row exists per account at a time.
	InsuranceStatusActiveProtection = 2

	// InsuranceStatusProtectionCompleted indicates a protection period has ended
	// Historical record of past protection activations. Multiple completed rows can exist per account.
	InsuranceStatusProtectionCompleted = 3
)

// Insurance policy name constants
const (
	InsuranceDefaultPolicyName = "Wick Insurance Default"
)

// InsurancePolicyPackage represents the insurance_policy_packages table
type InsurancePolicyPackage struct {
	ID                        int64                      `gorm:"primaryKey;column:id" json:"id"`
	Name                      string                     `gorm:"column:name;not null" json:"name"`
	ProtectionDurationMinutes int                        `gorm:"column:protection_duration_minutes;not null" json:"protection_duration_minutes"`
	ExclusionPeriodRatio      shopspring_decimal.Decimal `gorm:"type:decimal(5,2);column:exclusion_period_ratio;not null" json:"exclusion_period_ratio"`
	PricePerUse               shopspring_decimal.Decimal `gorm:"type:decimal(20,8);column:price_per_use;not null;default:0" json:"price_per_use"`
	IsActive                  bool                       `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedAt                 time.Time                  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt                 time.Time                  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the table name for GORM
func (InsurancePolicyPackage) TableName() string {
	return "insurance_policy_packages"
}

// SubAccountInsurance represents the sub_account_insurances table
type SubAccountInsurance struct {
	ID                     int64      `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID           int64      `gorm:"column:sub_account_id;not null" json:"sub_account_id"`
	PolicyID               int64      `gorm:"column:policy_id;not null" json:"policy_id"`
	Status                 int        `gorm:"column:status;not null" json:"status"` // 1=SUBSCRIBED, 2=ACTIVE_PROTECTION, 3=PROTECTION_COMPLETED
	StartTime              time.Time  `gorm:"column:start_time;not null" json:"start_time"`
	ExpiredTime            *time.Time `gorm:"column:expired_time" json:"expired_time"`
	LastPositionIncreaseAt *time.Time `gorm:"column:last_position_increase_at" json:"last_position_increase_at"`
	CreatedAt              time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt              time.Time  `gorm:"column:updated_at" json:"updated_at"`

	SubAccount *SubAccount             `gorm:"-" json:"sub_account,omitempty"`
	Policy     *InsurancePolicyPackage `gorm:"foreignKey:PolicyID" json:"policy,omitempty"`
}

// TableName returns the table name for GORM
func (SubAccountInsurance) TableName() string {
	return "sub_account_insurances"
}
