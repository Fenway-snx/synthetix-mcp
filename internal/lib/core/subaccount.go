package core

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// Re-export SubAccountType constants for convenience
const (
	AccountTypeNormal = postgrestypes.AccountTypeNormal
	AccountTypeSLP    = postgrestypes.AccountTypeSLP
)

// DefaultStateFlushInterval is the default interval for flushing buffered account states to DB
const DefaultStateFlushInterval = 200 * time.Millisecond

type SubAccount struct {
	Id   SubAccountId                 `json:"id"`
	Type postgrestypes.SubAccountType `json:"type"` // Account type: 0=normal, 1=slp

	RealizedPnl   shopspring_decimal.Decimal `json:"realized_pnl"` // realized PnL in USDT
	Collaterals   map[string]Collateral      `json:"collaterals"`
	DeadManSwitch *DeadManSwitchState        `json:"dead_man_switch,omitempty"`

	cloidIndex map[ClientOrderId]cloidEntry
	leverage   map[string]uint32       // symbol -> leverage (1-100)
	OpenOrders map[VenueOrderId]*Order `json:"open_orders"` // orderID -> order
	Positions  map[string]Position     `json:"positions"`   // symbol -> position

	ConditionalOrders map[string]ConditionalOrders `json:"pending_orders"` // symbol

	Status v4grpc.SubaccountStatus `json:"status"` // Account status: 1=active, 2=frozen
	// LTV tracking
	CurrentLTV     shopspring_decimal.Decimal `json:"current_ltv"`     // Current loan-to-value ratio
	BorrowCapacity shopspring_decimal.Decimal `json:"borrow_capacity"` // Maximum USDT that can be borrowed
	LastLTVCheck   time.Time                  `json:"last_ltv_check"`  // Last time LTV was checked

	// Wick Insurance - all insurance state grouped together
	Insurance *InsuranceState `json:"insurance"`

	FeeRates *FeeRate

	// TWAPPendingRecoveries holds JSON-encoded TWAPExecutionState snapshots from the
	// subaccount sync path (rebuilt from open_order_twap and sent as BasicOrder.twap_execution_state).
	// The trading actor drains this once during TWAP recovery after startup sync.
	TWAPPendingRecoveries []string `json:"-"`
}
type NewSubAccountParams struct {
	Collaterals           map[string]Collateral
	ConditionalOrders     map[string]ConditionalOrders
	DeadManSwitch         *DeadManSwitchState
	FeeRates              *FeeRate
	Leverage              map[string]uint32
	OpenOrders            map[VenueOrderId]*Order
	Positions             map[string]Position
	SubAccountId          SubAccountId
	TWAPPendingRecoveries []string
	Type                  postgrestypes.SubAccountType // Account type: 0=normal, 1=slp
}

type DeadManSwitchState struct {
	Generation     uint64     `json:"generation"`
	IsActive       bool       `json:"is_active"`
	TimeoutSeconds int64      `json:"timeout_seconds"`
	TriggerTime    *time.Time `json:"trigger_time,omitempty"`
}

// NewSubAccount creates a new SubAccount with properly initialized fields
// Note: Balance, AdjustedBalance, AvailableMargin, and cumulative margins are
// computed properties (via Actor methods)
func NewSubAccount(
	params NewSubAccountParams,
) *SubAccount {
	// Initialize maps if nil
	if params.Positions == nil {
		params.Positions = make(map[string]Position)
	}
	if params.OpenOrders == nil {
		params.OpenOrders = make(map[VenueOrderId]*Order)
	}
	if params.Collaterals == nil {
		params.Collaterals = make(map[string]Collateral)
	}
	if params.Leverage == nil {
		params.Leverage = make(map[string]uint32)
	}
	if params.ConditionalOrders == nil {
		params.ConditionalOrders = make(map[string]ConditionalOrders)
	}
	if params.FeeRates == nil {
		params.FeeRates = NewFeeRate(shopspring_decimal.Zero, shopspring_decimal.Zero)
	}

	// Default to normal account type if not specified (0 is the default for int32)
	// No need to check, as AccountTypeNormal is 0 which is the zero value
	subAccount := &SubAccount{
		Id:                    params.SubAccountId,
		Type:                  params.Type,
		Positions:             params.Positions,
		OpenOrders:            params.OpenOrders,
		Collaterals:           params.Collaterals,
		DeadManSwitch:         params.DeadManSwitch,
		cloidIndex:            make(map[ClientOrderId]cloidEntry),
		leverage:              params.Leverage,
		ConditionalOrders:     params.ConditionalOrders,
		RealizedPnl:           shopspring_decimal.Zero,
		Status:                v4grpc.SubaccountStatus_ACCOUNT_STATUS_ACTIVE,
		CurrentLTV:            shopspring_decimal.Zero,
		BorrowCapacity:        shopspring_decimal.Zero,
		LastLTVCheck:          time.Time{},
		FeeRates:              params.FeeRates,
		TWAPPendingRecoveries: params.TWAPPendingRecoveries,
	}

	subAccount.RebuildCloidIndex()

	return subAccount
}

// RecalculateUPNL recalculates total UPNL from all positions
// Use this when positions are added/removed or during full account state updates
func (s *SubAccount) CalculateUPNL() shopspring_decimal.Decimal {
	upnl := shopspring_decimal.Zero
	for _, position := range s.Positions {
		upnl = upnl.Add(position.UPNL)
	}
	return upnl
}

// IsSLPAccount returns true if the subaccount is of SLP account type.
func (s *SubAccount) IsSLPAccount() bool {
	return s.Type == postgrestypes.AccountTypeSLP
}

// IsFrozen returns true if the subaccount status is FROZEN (for backward compatibility)
func (s *SubAccount) IsFrozen() bool {
	return s.Status == v4grpc.SubaccountStatus_ACCOUNT_STATUS_FROZEN
}

func (s *SubAccount) GetLeverage(symbol string) uint32 {
	leverage, exists := s.leverage[symbol]
	if !exists {
		// default to 1x if not found
		return 1
	}
	return leverage
}

func (s *SubAccount) SetLeverage(symbol string, leverage uint32) {
	s.leverage[symbol] = leverage
}

type Collateral struct {
	Symbol          string                     `json:"symbol"`
	Quantity        shopspring_decimal.Decimal `json:"quantity"`
	Withdrawable    shopspring_decimal.Decimal `json:"withdrawable"`
	PendingWithdraw shopspring_decimal.Decimal `json:"pending_withdraw"`
}

// OperationType represents the type of trading operation
type OperationType string

const (
	OperationCreateOrder OperationType = "CREATE_ORDER"
	OperationCancelOrder OperationType = "CANCEL_ORDER"
	OperationModifyOrder OperationType = "MODIFY_ORDER"
)

// DelegationPermission represents a permission type for delegations - strongly typed for type safety
type DelegationPermission string

// Delegation Permission constants - strongly typed (alphabetically ordered)
const (
	DelegationPermissionAdmin    DelegationPermission = "admin"
	DelegationPermissionDelegate DelegationPermission = "delegate"
	DelegationPermissionSession  DelegationPermission = "session"
	DelegationPermissionTrading  DelegationPermission = "trading"
)

// String returns the string representation of the permission
func (p DelegationPermission) String() string {
	return string(p)
}

// Returns true for session-level permissions ("session" or legacy "trading")
func (p DelegationPermission) IsSessionLevel() bool {
	return p == DelegationPermissionSession || p == DelegationPermissionTrading
}

// Returns true for delegate-level permission ("delegate")
func (p DelegationPermission) IsDelegateLevel() bool {
	return p == DelegationPermissionDelegate
}

// PermissionSatisfiedBy checks whether a stored permission level satisfies a requested permission.
// Hierarchy: delegate > session/trading (equivalent).
// - "delegate" satisfies requests for "delegate", "session", or "trading"
// - "session" satisfies requests for "session" or "trading" (but NOT "delegate")
// - "trading" satisfies requests for "session" or "trading" (but NOT "delegate")
func PermissionSatisfiedBy(requested, stored DelegationPermission) bool {
	if stored.IsDelegateLevel() {
		// Delegate satisfies everything
		return requested.IsDelegateLevel() || requested.IsSessionLevel()
	}
	if stored.IsSessionLevel() {
		// Session/trading only satisfies session-level requests
		return requested.IsSessionLevel()
	}
	return false
}

// IsValidDelegationPermission returns true for valid delegation permission values.
// Valid values: "delegate", "session", "trading". Admin is not a valid delegation permission.
func IsValidDelegationPermission(p DelegationPermission) bool {
	return p == DelegationPermissionDelegate || p == DelegationPermissionSession || p == DelegationPermissionTrading
}

// InsuranceState holds all insurance-related state for an account
type InsuranceState struct {
	Subscription     *InsuranceSubscription // Active subscription (from SUBSCRIBED row)
	ActiveProtection *InsuranceProtection   // Current protection instance (from ACTIVE_PROTECTION row)
}

// InsuranceSubscription represents an active insurance subscription
type InsuranceSubscription struct {
	PolicyID                  int64
	ProtectionDurationMinutes int
	ExclusionPeriodRatio      shopspring_decimal.Decimal
	LastPositionIncreaseAt    *time.Time
}

// InsuranceProtection represents an active protection instance
type InsuranceProtection struct {
	StartTime   time.Time
	ExpiredTime time.Time
}
