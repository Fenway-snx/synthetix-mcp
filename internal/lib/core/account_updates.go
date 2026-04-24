package core

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// Represents a minimal delta update when prices change
type PriceTriggeredUpdate struct {
	SubAccountId SubAccountId `json:"sub_account_id"`
	Timestamp    time.Time    `json:"timestamp"`
	// Only the changed account-level values
	SubAccountState AccountState `json:"sub_account_state"`

	// Only the affected position
	UpdatedPosition *PositionState `json:"position,omitempty"`

	// Only the affected collaterals
	UpdatedCollaterals []CollateralState `json:"collaterals,omitempty"`
}

type AccountState struct {
	AdjustedAccountValue shopspring_decimal.Decimal `json:"adjusted_account_value"`
	AccountValue         shopspring_decimal.Decimal `json:"account_value"`
	AvailableMargin      shopspring_decimal.Decimal `json:"available_margin"`
	CumulativeIMR        shopspring_decimal.Decimal `json:"cumulative_imr"`
	CumulativeMMR        shopspring_decimal.Decimal `json:"cumulative_mmr"`
	CumulativeRMR        shopspring_decimal.Decimal `json:"cumulative_rmr"`
	RealizedPnl          shopspring_decimal.Decimal `json:"realized_pnl"`
	UnrealizedPnl        shopspring_decimal.Decimal `json:"unrealized_pnl"`
	Debt                 shopspring_decimal.Decimal `json:"debt"`
	Withdrawable         shopspring_decimal.Decimal `json:"withdrawable"` // withdrawable amount USDT denominated
}

type PositionState struct {
	ADLBucket  int64                      `json:"adl_bucket"`
	PositionID uint64                     `json:"position_id"`
	Symbol     string                     `json:"symbol"`
	UPNL       shopspring_decimal.Decimal `json:"upnl"`
	IMR        shopspring_decimal.Decimal `json:"imr"`
	MMR        shopspring_decimal.Decimal `json:"mmr"`
	MarkPrice  shopspring_decimal.Decimal `json:"mark_price"`
}

// Contains only the changed collateral data
type CollateralState struct {
	AdjustedCollateralValue shopspring_decimal.Decimal `json:"adjusted_collateral_value"`
	CalculatedAt            time.Time                  `json:"calculated_at"` // When the collateral price was last updated (zero for USDT)
	Collateral              string                     `json:"collateral"`
	CollateralValue         shopspring_decimal.Decimal `json:"collateral_value"`
	HaircutRate             shopspring_decimal.Decimal `json:"haircut_rate"`       // Matching tier's marginal haircut rate (0 = no haircut)
	HaircutAdjustment       shopspring_decimal.Decimal `json:"haircut_adjustment"` // Tier's bracket-continuity constant (0 for single-tier)
	PendingWithdraw         shopspring_decimal.Decimal `json:"pending_withdraw"`
	Price                   shopspring_decimal.Decimal `json:"price"` // Raw market price used for collateral value calculation (1 for USDT)
	Quantity                shopspring_decimal.Decimal `json:"quantity"`
	WithdrawableAmount      shopspring_decimal.Decimal `json:"withdrawable_amount"`
}
