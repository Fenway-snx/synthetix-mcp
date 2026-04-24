package core

import (
	"encoding/json"
	"fmt"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// SLPTransferType represents the type of SLP transfer
type SLPTransferType string

const (
	SLPTransferTypeForceAutoExchange     SLPTransferType = "force_auto_exchange"
	SLPTransferTypeLiquidation           SLPTransferType = "liquidation"
	SLPTransferTypeVoluntaryAutoExchange SLPTransferType = "voluntary_auto_exchange"
)

// SLPTransferReason represents the reason for SLP transfer
type SLPTransferReason string

const (
	// Force auto-exchange reasons
	SLPTransferReasonLLTVBreach    SLPTransferReason = "lltv_breach"
	SLPTransferReasonLTVBreach     SLPTransferReason = "ltv_breach"
	SLPTransferReasonOverleveraged SLPTransferReason = "overleveraged"

	// Liquidation reasons
	SLPTransferReasonPositionLiquidated SLPTransferReason = "position_liquidated"

	// Voluntary auto-exchange reasons
	SLPTransferReasonUserRequested SLPTransferReason = "user_requested"
)

// CollateralExchangeType distinguishes auto (forced) from voluntary exchanges.
type CollateralExchangeType int64

const (
	CollateralExchangeType_Unknown CollateralExchangeType = iota
	CollateralExchangeType_Auto
	CollateralExchangeType_Voluntary
)

func (t CollateralExchangeType) String() string {
	switch t {
	case CollateralExchangeType_Auto:
		return "auto"
	case CollateralExchangeType_Voluntary:
		return "voluntary"
	default:
		return "unknown"
	}
}

func (t CollateralExchangeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *CollateralExchangeType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "auto":
		*t = CollateralExchangeType_Auto
	case "voluntary":
		*t = CollateralExchangeType_Voluntary
	case "unknown":
		*t = CollateralExchangeType_Unknown
	default:
		return fmt.Errorf("unrecognized CollateralExchangeType: %q", s)
	}
	return nil
}

type SlpCollateralTakeoverEvent struct {
	Asset         AssetName                  `json:"asset"`
	From          SubAccountId               `json:"from"`
	Quantity      shopspring_decimal.Decimal `json:"Quantity"`
	To            SubAccountId               `json:"to"`
	TransferredAt time.Time                  `json:"transferred_at"`
}

// VoluntaryAutoExchangeRequest represents a request for voluntary auto-exchange
type VoluntaryAutoExchangeRequest struct {
	TimestampMs      int64                      `json:"timestamp_ms"`
	TimestampUs      int64                      `json:"timestamp_us"`
	SubAccountId     SubAccountId               `json:"sub_account_id"`
	TargetUSDTAmount shopspring_decimal.Decimal `json:"target_usdt_amount"`
	SourceAsset      string                     `json:"source_asset"`
	RequestID        string                     `json:"request_id"`
}

// VoluntaryAutoExchangeResponse represents the response after processing voluntary auto-exchange
type VoluntaryAutoExchangeResponse struct {
	TimestampMs       int64                      `json:"timestamp_ms"`
	TimestampUs       int64                      `json:"timestamp_us"`
	SubAccountId      SubAccountId               `json:"sub_account_id"`
	SourceAsset       string                     `json:"source_asset"`
	SourceAmountTaken shopspring_decimal.Decimal `json:"source_amount_taken"`
	TargetAsset       string                     `json:"target_asset"`
	TargetAmount      shopspring_decimal.Decimal `json:"target_amount"`
	IndexPrice        shopspring_decimal.Decimal `json:"index_price"`
	EffectiveHaircut  shopspring_decimal.Decimal `json:"effective_haircut"`
	Collateral        []CollateralBalance        `json:"collateral"`
	RequestID         string                     `json:"request_id"`
	Success           bool                       `json:"success"`
	Error             string                     `json:"error,omitempty"`
}

// CollateralBalance represents a collateral balance in the response
type CollateralBalance struct {
	Symbol   string                     `json:"symbol"`
	Quantity shopspring_decimal.Decimal `json:"quantity"`
}

// SLPTakeoverBroadcastEvent represents a comprehensive SLP takeover event
// containing all position and balance transfers for WebSocket broadcast
// and JetStream audit persistence.
type SLPTakeoverBroadcastEvent struct {
	TakeoverID    uint64                `json:"takeover_id"`
	LiquidationID int64                 `json:"liquidation_id,omitempty"`
	SubAccountId  SubAccountId          `json:"sub_account_id"` // liquidatee
	Reason        LiquidationReason     `json:"reason,omitempty"`
	Transfers     []SLPTakeoverTransfer `json:"transfers"`
	TakenOverAt   time.Time             `json:"taken_over_at"`

	PreTakeoverMargin       shopspring_decimal.Decimal `json:"pre_takeover_margin,omitempty"`
	PreTakeoverAccountValue shopspring_decimal.Decimal `json:"pre_takeover_account_value,omitempty"`
	BadDebt                 shopspring_decimal.Decimal `json:"bad_debt,omitempty"`
}

// SLPTakeoverTransfer represents transfers to a single SLP destination
type SLPTakeoverTransfer struct {
	DestinationSubaccount SubAccountId         `json:"destination_subaccount"`
	Positions             SLPTakeoverPositions `json:"positions"`
	Balances              SLPTakeoverBalances  `json:"balances"`
}

// SLPTakeoverTransferMap maps SLP subaccount IDs to their takeover transfers
type SLPTakeoverTransferMap map[SubAccountId]*SLPTakeoverTransfer

// SLPTakeoverPosition represents a position transferred during takeover
type SLPTakeoverPosition struct {
	Symbol            string `json:"symbol"`
	Side              string `json:"side"`
	Quantity          string `json:"quantity"`
	EntryPrice        string `json:"entry_price"`
	MarkPrice         string `json:"mark_price,omitempty"`
	UnrealizedPnl     string `json:"unrealized_pnl"`
	UsedMargin        string `json:"used_margin"`
	MaintenanceMargin string `json:"maintenance_margin"`
	Status            string `json:"status"`
	TradeID           int64  `json:"trade_id,omitempty"`
}

// SLPTakeoverPositions is a slice of SLPTakeoverPosition
type SLPTakeoverPositions []SLPTakeoverPosition

// SLPTakeoverBalance represents a balance transferred during takeover
type SLPTakeoverBalance struct {
	Symbol   string `json:"symbol"`
	Quantity string `json:"quantity"`
}

// SLPTakeoverBalances is a slice of SLPTakeoverBalance
type SLPTakeoverBalances []SLPTakeoverBalance
