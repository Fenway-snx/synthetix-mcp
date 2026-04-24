package core

import (
	"fmt"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/transfer"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// EventType represents the type of matching engine event
type EventType string

const (
	EventTypeOrderAccepted EventType = "ORDER_ACCEPTED"
	EventTypeOrderCanceled EventType = "ORDER_CANCELED"
	EventTypeTradeExecuted EventType = "TRADE_EXECUTED"
)

// LiquidationReason indicates why a liquidation was triggered.
type LiquidationReason string

const (
	LiquidationReasonMMRBreach LiquidationReason = "MMR_BREACH"
	LiquidationReasonRMRBreach LiquidationReason = "RMR_BREACH"
)

// LiquidationStageLabel is the terminal stage of a liquidation for
// audit records.
type LiquidationStageLabel string

const (
	LiquidationStageLabelSLPTakeover LiquidationStageLabel = "SLP_TAKEOVER"
	LiquidationStageLabelADL         LiquidationStageLabel = "ADL"
	LiquidationStageLabelPartialMMR  LiquidationStageLabel = "PARTIAL_MMR"
)

// LiquidationEvent is published when a liquidation completes and is
// persisted to the liquidations table by the subaccount service.
type LiquidationEvent struct {
	ID                         int64                      `json:"id"`
	SubAccountID               SubAccountId               `json:"sub_account_id"`
	Reason                     LiquidationReason          `json:"reason"`
	Stage                      LiquidationStageLabel      `json:"stage"`
	TotalPositions             int64                      `json:"total_positions"`
	TotalCollateralSymbols     int64                      `json:"total_collateral_symbols"`
	PreLiquidationMargin       shopspring_decimal.Decimal `json:"pre_liquidation_margin"`
	PreLiquidationAccountValue shopspring_decimal.Decimal `json:"pre_liquidation_account_value"`
	BadDebt                    shopspring_decimal.Decimal `json:"bad_debt"`
	CreatedAt                  time.Time                  `json:"created_at"`
}

// DepositEvent represents a deposit event from NATS
type DepositEvent struct {
	Collateral            string                     `json:"collateral"`
	LogIndex              uint                       `json:"log_index"`
	Quantity              shopspring_decimal.Decimal `json:"quantity"`
	ReceiverWalletAddress string                     `json:"receiver_wallet_address"`
	SubAccountId          SubAccountId               `json:"subaccount_id"`
	TxHash                string                     `json:"tx_hash"`
}

// This event is sent from the sub account service to the trading service
// to create a sub account either from a deposit or via the APIs
type SubAccountCreatedEvent struct {
	Collateral         string                     `json:"collateral"`
	MakerFeeRate       FeeRateAmount              `json:"maker_fee_rate"`
	MasterID           int64                      `json:"master_id"`
	MaxBorrowCapacity  shopspring_decimal.Decimal `json:"max_borrow_capacity"`
	MaxOrdersPerMarket int64                      `json:"max_orders_per_market"`
	MaxTotalOrders     int64                      `json:"max_total_orders"`
	Name               string                     `json:"name"`
	Quantity           shopspring_decimal.Decimal `json:"quantity"`
	RequestId          string                     `json:"request_id"`
	SubAccountId       SubAccountId               `json:"subaccount_id"`
	TakerFeeRate       FeeRateAmount              `json:"taker_fee_rate"`
}

// DepositProcessedEvent represents an event when a deposit is processed
type DepositProcessedEvent struct {
	SubAccountId          SubAccountId               `json:"subaccount_id"`
	ReceiverWalletAddress string                     `json:"receiver_wallet_address"`
	Collateral            string                     `json:"collateral"`
	Quantity              shopspring_decimal.Decimal `json:"quantity"`
	Timestamp             time.Time                  `json:"timestamp"`
	TxHash                string                     `json:"tx_hash,omitempty"`
	TransactionID         int                        `json:"transaction_id"`
}

// WithdrawalFinalizedEvent represents a terminal withdrawal outcome after
// the off-chain balance change has been fully applied.
type WithdrawalFinalizedEvent struct {
	Amount               shopspring_decimal.Decimal `json:"amount"`
	Asset                AssetName                  `json:"asset"`
	DestinationAddress   WalletAddress              `json:"destination_address,omitempty"`
	FailureReason        string                     `json:"failure_reason,omitempty"`
	OffchainWithdrawalId OffchainWithdrawalId       `json:"offchain_withdrawal_id"`
	OnchainWithdrawalId  OnchainWithdrawalId        `json:"onchain_withdrawal_id,omitempty"`
	Status               WithdrawalStatus           `json:"status"`
	SubAccountId         SubAccountId               `json:"sub_account_id"`
	Timestamp            time.Time                  `json:"timestamp"`
	TxHash               string                     `json:"tx_hash,omitempty"`
}

// This is the initial withdrawal request from trading service to subaccount service
// after all validations
type WithdrawalRequestEvent struct {
	Amount               shopspring_decimal.Decimal `json:"amount"`
	Asset                AssetName                  `json:"asset"`
	Destination          WalletAddress              `json:"destination"`
	Fee                  shopspring_decimal.Decimal `json:"fee"` // Withdrawal fee in native asset (preserved for response)
	OffchainWithdrawalId OffchainWithdrawalId       `json:"offchain_withdrawal_id"`
	RequestId            RequestId                  `json:"request_id"`
	RequestedAt          time.Time                  `json:"requested_at"`
	SubAccountId         SubAccountId               `json:"sub_account_id"`
	WalletAddress        WalletAddress              `json:"wallet_address"`
}

// WithdrawCompletedEvent represents an event when a withdrawal is completed on-chain
// Sent from relayer to subaccount service
type WithdrawalCompletedEvent struct {
	CompletedAt         time.Time           `json:"completed_at"`
	OnchainWithdrawalId OnchainWithdrawalId `json:"onchain_withdrawal_id"`
	Status              WithdrawalStatus    `json:"status"`
	TxHash              string              `json:"tx_hash"` // Transaction hash of the disburse transaction
}

// WithdrawDisputedEvent represents an event when a withdrawal is disputed
// Sent from relayer to subaccount service
type WithdrawalDisputedEvent struct {
	DisputedAt          time.Time           `json:"disputed_at"`
	OnchainWithdrawalId OnchainWithdrawalId `json:"onchain_withdrawal_id"`
	ReasonCode          int64               `json:"reason_code"` // TODO: Make this a strong type and sort out codes with @pp
}

// WithdrawProcessedEvent represents a request to trading service to deduct collateral from actor
// Sent from subaccount service to trading service
type WithdrawalProcessedEvent struct {
	Asset                AssetName                  `json:"asset"`
	Error                string                     `json:"error"` // Empty = success, non-empty = failure reason
	Fee                  shopspring_decimal.Decimal `json:"fee"`   // Withdrawal fee (echoed from request)
	OffchainWithdrawalId OffchainWithdrawalId       `json:"offchain_withdrawal_id"`
	ProcessedAt          time.Time                  `json:"processed_at"`
	Quantity             shopspring_decimal.Decimal `json:"quantity"`
	SubAccountId         SubAccountId               `json:"sub_account_id"`
}

// WithdrawActorUpdatedEvent represents confirmation that actor collateral was updated
// Sent from trading service back to subaccount service as response
type WithdrawalActorUpdatedEvent struct {
	Error                string               `json:"error,omitempty"`
	OffchainWithdrawalId OffchainWithdrawalId `json:"offchain_withdrawal_id"`
}

// WithdrawSubmissionFailedEvent represents an event when relayer fails to submit withdrawal on-chain
// Sent from relayer to subaccount service so it can unlock pending funds
type WithdrawalSubmissionFailedEvent struct {
	Error                string               `json:"error"` // Error message describing the failure
	FailedAt             time.Time            `json:"failed_at"`
	OffchainWithdrawalId OffchainWithdrawalId `json:"offchain_withdrawal_id"`
}

// The following event is sent from the sub account service to trading via nats req/reply
type UserInitiatedTransferRequest struct {
	Asset      AssetName                  `json:"asset"`
	From       SubAccountId               `json:"from"`
	Quantity   shopspring_decimal.Decimal `json:"quantity"`
	RequestId  RequestId                  `json:"request_id"`
	To         SubAccountId               `json:"to"`
	TransferId transfer.Id                `json:"transfer_id"`
}

// Represents a transfer response from trading to the sub account service
// If Error is empty, the transfer succeeded
type UserInitiatedTransferResponse struct {
	TransferId    transfer.Id                    `json:"transfer_id"`
	RequestId     RequestId                      `json:"request_id"`
	FromBalance   shopspring_decimal.Decimal     `json:"from_balance"`
	ToBalance     shopspring_decimal.Decimal     `json:"to_balance"`
	Status        transfer.Status                `json:"status"`
	Error         string                         `json:"error,omitempty"`
	ErrorCode     snx_lib_status_codes.ErrorCode `json:"error_code,omitempty"`
	TransferredAt time.Time                      `json:"transferred_at"`
}

// Published by trading service after a transfer is processed.
// The sub account service listens for this event and records it in the transfer_history table.
type TransferHistoryEvent struct {
	TransferId    transfer.Id                `json:"transfer_id"`
	RequestId     RequestId                  `json:"request_id"`
	From          SubAccountId               `json:"from"`
	To            SubAccountId               `json:"to"`
	Asset         AssetName                  `json:"asset"`
	Quantity      shopspring_decimal.Decimal `json:"quantity"`
	TransferType  transfer.Type              `json:"transfer_type"`
	ExchangeType  *CollateralExchangeType    `json:"exchange_type,omitempty"`
	Status        transfer.Status            `json:"status"`
	Error         string                     `json:"error,omitempty"`
	TransferredAt time.Time                  `json:"transferred_at"`
}

// OrderAcceptedEvent represents an order that was accepted into the order book
type OrderAcceptedEvent struct {
	SequenceNumber SnapshotSequence `json:"sequence_number"`
	EventType      EventType        `json:"event_type"`
	Timestamp      time.Time        `json:"timestamp"`
	Symbol         string           `json:"symbol"`
	OrderId        OrderId          `json:"order_id"`
	SubAccountId   SubAccountId     `json:"account_id"` // Account that placed the order
	Side           string           `json:"side"`
	Price          uint64           `json:"price"`    // Price in smallest units (8 decimals)
	Quantity       uint64           `json:"quantity"` // Quantity in smallest units (8 decimals)
}

// OrderCanceledEvent represents an order that was canceled
type OrderCanceledEvent struct {
	SequenceNumber   SnapshotSequence `json:"sequence_number"`
	EventType        EventType        `json:"event_type"`
	Timestamp        time.Time        `json:"timestamp"`
	Symbol           string           `json:"symbol"`
	OrderId          OrderId          `json:"order_id"`
	SubAccountId     SubAccountId     `json:"account_id"` // Account that owned the canceled order
	Side             string           `json:"side"`
	Price            uint64           `json:"price"`
	CanceledQuantity uint64           `json:"canceled_quantity"`
}

// TradeExecutedEvent represents a trade that was executed
type TradeExecutedEvent struct {
	SequenceNumber    SnapshotSequence `json:"sequence_number"`
	EventType         EventType        `json:"event_type"`
	Symbol            string           `json:"symbol"`
	TakerOrderID      VenueOrderId     `json:"taker_order_id"`
	TakerSubAccountId SubAccountId     `json:"taker_account_id"` // Account that initiated the trade (taker)
	MakerOrderID      VenueOrderId     `json:"maker_order_id"`
	MakerSubAccountId SubAccountId     `json:"maker_account_id"` // Account that provided liquidity (maker)
	Price             uint64           `json:"price"`
	Quantity          uint64           `json:"quantity"`
	TradedAt          time.Time        `json:"filled_at"`
}

// T.B.C.
type FundingHistory struct {
	ID           uint64                     `json:"id"`
	SubAccountId SubAccountId               `json:"sub_account_id"`
	Symbol       string                     `json:"symbol"`
	PositionSize shopspring_decimal.Decimal `json:"position_size"`
	FundingRate  shopspring_decimal.Decimal `json:"funding_rate"`
	Payment      shopspring_decimal.Decimal `json:"payment"`
	MarkPrice    shopspring_decimal.Decimal `json:"mark_price"`
	FundedAt     time.Time                  `json:"funded_at"`
}

// LiquidationNotification represents a notification when a position is liquidated
type LiquidationNotification struct {
	ID           uint64                     `json:"id"`
	SubAccountId SubAccountId               `json:"sub_account_id"`
	TradeID      uint64                     `json:"trade_id"`
	Symbol       string                     `json:"symbol"`
	Direction    Direction                  `json:"direction"`
	Price        shopspring_decimal.Decimal `json:"price"`
	Quantity     shopspring_decimal.Decimal `json:"quantity"`
	Leverage     uint32                     `json:"leverage"` // Position leverage at time of liquidation (1-100)
	LiquidatedAt time.Time                  `json:"created_at"`
}

// LiquidationFeeSettlement represents the settlement of accumulated liquidation clearance fees
// Published when a liquidation session ends (account recovers from liquidation state)
type LiquidationFeeSettlement struct {
	SubAccountId   SubAccountId               `json:"sub_account_id"`
	AccumulatedFee shopspring_decimal.Decimal `json:"accumulated_fee"` // Total fee accumulated during liquidation session
	FeeCharged     shopspring_decimal.Decimal `json:"fee_charged"`     // Actual fee charged (may be capped at equity)
	FeeWrittenOff  shopspring_decimal.Decimal `json:"fee_written_off"` // Fee that couldn't be charged (accumulated - charged)
	SettledAt      time.Time                  `json:"settled_at"`
}

// Records a snapshot for persistence and downstream APIs. Timestamp fields
// may alias the same *time.Time: you may reassign a field (including nil)
// to refer to a different instant; do not mutate the pointed-to time
// through a field (e.g. *order.CreatedAt).
type OrderHistory struct {
	ID                     int64                      `json:"id"`
	OrderId                OrderId                    `json:"order_id"`
	Symbol                 string                     `json:"symbol"`
	Side                   OrderSide                  `json:"side"`
	Type                   OrderType                  `json:"type"`
	Direction              Direction                  `json:"direction"`
	TimeInForce            v4grpc.TimeInForce         `json:"time_in_force"`
	Price                  shopspring_decimal.Decimal `json:"price"`
	TriggerPrice           shopspring_decimal.Decimal `json:"trigger_price"`
	TriggerPriceType       TriggerPriceType           `json:"trigger_price_type"`
	Quantity               shopspring_decimal.Decimal `json:"quantity"`
	FilledQuantity         shopspring_decimal.Decimal `json:"filled_quantity"`
	FilledPrice            shopspring_decimal.Decimal `json:"filled_price"`
	Status                 OrderState                 `json:"status"`
	TPOrderId              *OrderId                   `json:"tp_order_id"`
	SLOrderId              *OrderId                   `json:"sl_order_id"`
	SubAccountId           SubAccountId               `json:"sub_account_id"`
	PriceExponent          int64                      `json:"price_exponent"`
	TriggeredByLiquidation bool                       `json:"triggered_by_liquidation"`
	ClosePosition          bool                       `json:"close_position"`
	ReduceOnly             bool                       `json:"reduce_only"`
	PostOnly               bool                       `json:"post_only"`
	CancelReason           OrderCancelReason          `json:"cancel_reason,omitempty"`
	Source                 string                     `json:"source,omitempty"`
	Meseq                  uint64                     `json:"meseq,omitempty"` // Matching-engine sequence; 0 = not from matching
	Met                    int64                      `json:"met,omitempty"`   // Matching-engine event time (Unix µs); 0 = not from matching
	CancelledAt            *time.Time                 `json:"cancelled_at"`
	CreatedAt              *time.Time                 `json:"created_at"`
	ExpiresAt              *time.Time                 `json:"expires_at"`
	ModifiedAt             *time.Time                 `json:"modified_at"`
	PlacedAt               *time.Time                 `json:"placed_at"`
	RejectedAt             *time.Time                 `json:"rejected_at"`
	RejectionReason        string                     `json:"rejection_reason,omitempty"`
	TradedAt               *time.Time                 `json:"traded_at"`
	UpdatedAt              time.Time                  `json:"updated_at"`
}
type TradeHistory struct {
	ID                          uint64                     `json:"id"`
	Symbol                      string                     `json:"symbol"`
	Price                       shopspring_decimal.Decimal `json:"price"`
	Quantity                    shopspring_decimal.Decimal `json:"quantity"`
	OrderId                     OrderId                    `json:"order_id"`
	Direction                   Direction                  `json:"direction"`
	SubAccountId                SubAccountId               `json:"sub_account_id"`
	OrderType                   OrderType                  `json:"order_type"`
	TradedAt                    time.Time                  `json:"traded_at"`
	PriceExponent               int64                      `json:"price_exponent"`
	IsTaker                     bool                       `json:"is_taker"`
	Fee                         shopspring_decimal.Decimal `json:"fee"`
	FeeRate                     shopspring_decimal.Decimal `json:"fee_rate"`
	ClosedPNL                   shopspring_decimal.Decimal `json:"closed_pnl"`
	RealizedPNL                 shopspring_decimal.Decimal `json:"realized_pnl"`
	MarkPrice                   shopspring_decimal.Decimal `json:"mark_price"`
	EntryPrice                  shopspring_decimal.Decimal `json:"entry_price"`
	Position                    *Position                  `json:"position,omitempty"`
	TriggeredByLiquidation      bool                       `json:"triggered_by_liquidation"`
	LiquidationClearanceFee     shopspring_decimal.Decimal `json:"liquidation_clearance_fee"`
	LiquidationClearanceFeeRate shopspring_decimal.Decimal `json:"liquidation_clearance_fee_rate"`
	PostOnly                    bool                       `json:"post_only"`
	ReduceOnly                  bool                       `json:"reduce_only"`
	Source                      string                     `json:"source,omitempty"`
	Meseq                       uint64                     `json:"meseq,omitempty"` // Matching-engine sequence; 0 = not from matching
	Met                         int64                      `json:"met,omitempty"`   // Matching-engine event time (Unix µs); 0 = not from matching
	TradeID                     int64                      `json:"trade_id,omitempty"`
	LiquidationReason           LiquidationReason          `json:"liquidation_reason,omitempty"`
	LiquidationID               int64                      `json:"liquidation_id,omitempty"`
}

type PositionEvent struct {
	ID                       uint64                     `json:"id"`
	Symbol                   string                     `json:"symbol"`
	Side                     PositionSide               `json:"side"` // Position side (LONG/SHORT position state)
	TradeSide                PositionSide               `json:"-"`    // Internal use only - trade side (BUY/SELL action for coordinator)
	EntryPrice               shopspring_decimal.Decimal `json:"entry_price"`
	Quantity                 shopspring_decimal.Decimal `json:"quantity"`
	UPNL                     shopspring_decimal.Decimal `json:"upnl"`
	UsedMargin               shopspring_decimal.Decimal `json:"used_margin"`
	MaintenanceMargin        shopspring_decimal.Decimal `json:"maintenance_margin"`
	LiquidationPrice         shopspring_decimal.Decimal `json:"liquidation_price"`
	NetPositionFundingPnl    shopspring_decimal.Decimal `json:"net_position_funding_pnl"`
	AccumulatedRealizedPnl   shopspring_decimal.Decimal `json:"accumulated_realized_pnl"`
	AccumulatedFees          shopspring_decimal.Decimal `json:"accumulated_fees"`
	AccumulatedCloseValue    shopspring_decimal.Decimal `json:"accumulated_close_value"`
	AccumulatedCloseQuantity shopspring_decimal.Decimal `json:"accumulated_close_quantity"`
	ClosePrice               shopspring_decimal.Decimal `json:"close_price"`
	CloseReason              CloseReason                `json:"close_reason,omitempty"`
	Action                   PositionEventAction        `json:"action"`
	ADLBucket                int64                      `json:"adl_bucket"`
	Leverage                 *uint32                    `json:"leverage,omitempty"`
	SubAccountId             SubAccountId               `json:"sub_account_id"`
	TakeProfitOrderIds       []OrderId                  `json:"take_profit_orders"`
	StopLossOrderIds         []OrderId                  `json:"stop_loss_orders"`
	TradeID                  int64                      `json:"trade_id,omitempty"`
	Meseq                    uint64                     `json:"meseq,omitempty"` // Matching-engine sequence; 0 = not from matching
	Met                      int64                      `json:"met,omitempty"`   // Matching-engine event time (Unix µs); 0 = not from matching
	ClosedAt                 *time.Time                 `json:"closed_at"`
	CreatedAt                *time.Time                 `json:"created_at"`
	ModifiedAt               *time.Time                 `json:"modified_at"`
	UpdatedAt                time.Time                  `json:"updated_at"`
}

type PositionUpdate struct {
	TradeID   int64           `json:"trade_id"`
	Symbol    string          `json:"symbol"`
	Positions []PositionEvent `json:"positions"`
	Timestamp time.Time       `json:"timestamp"`
}

type PositionTPSLEvent struct {
	PositionID         uint64       `json:"position_id"`
	SubAccountId       SubAccountId `json:"sub_account_id"`
	Symbol             string       `json:"symbol"`
	TakeProfitOrderIds []OrderId    `json:"take_profit_orders"`
	StopLossOrderIds   []OrderId    `json:"stop_loss_orders"`
	Timestamp          time.Time    `json:"timestamp"`
}

// TODO: Too late for the Trading Comp but I would love the make 0 == Unknown before go live
type PositionEventAction int32

const (
	PositionEventActionOpen PositionEventAction = iota
	PositionEventActionClose
	PositionEventActionUpdate
)

func (a PositionEventAction) Int32() int32 {
	return int32(a)
}

// Helper function to convert position action enum to string for display.
func PositionActionToString(action PositionEventAction) string {
	switch action {
	case PositionEventActionOpen:
		return "Open"
	case PositionEventActionClose:
		return "Close"
	case PositionEventActionUpdate:
		return "Update"
	default:
		return fmt.Sprintf("<unknown action: int(v)=%d>", action)
	}
}

type OpenOrderEvent struct {
	OrderId            OrderId                    `json:"order_id"`
	Symbol             string                     `json:"symbol"`
	Side               OrderSide                  `json:"side"`
	Type               OrderType                  `json:"type"`
	OriginalType       OrderType                  `json:"original_type"`
	Direction          Direction                  `json:"direction"`
	TimeInForce        v4grpc.TimeInForce         `json:"time_in_force"`
	Price              shopspring_decimal.Decimal `json:"price"`
	PriceExponent      int64                      `json:"price_exponent"`
	Quantity           shopspring_decimal.Decimal `json:"quantity"`
	QuantityExponent   int64                      `json:"quantity_exponent"`
	RemainingQuantity  shopspring_decimal.Decimal `json:"remaining_quantity"`
	SubAccountId       SubAccountId               `json:"sub_account_id"`
	ReduceOnly         bool                       `json:"reduce_only"`
	PostOnly           bool                       `json:"post_only"`
	TakeProfitOrderId  *OrderId                   `json:"take_profit_order_id"`
	StopLossOrderId    *OrderId                   `json:"stop_loss_order_id"`
	PositionID         uint64                     `json:"position_id"`
	Action             OpenOrderEventAction       `json:"action"`
	TriggerPrice       shopspring_decimal.Decimal `json:"trigger_price"`
	TriggerPriceType   TriggerPriceType           `json:"trigger_price_type"`
	IsActive           bool                       `json:"is_active"`
	ClosePosition      bool                       `json:"close_position"`
	CancelReason       OrderCancelReason          `json:"cancel_reason,omitempty"`
	Meseq              uint64                     `json:"meseq,omitempty"` // Matching-engine sequence; 0 = not from matching
	Met                int64                      `json:"met,omitempty"`   // Matching-engine event time (Unix µs); 0 = not from matching
	CreatedAt          *time.Time                 `json:"created_at"`
	ExpiresAt          *time.Time                 `json:"expires_at"`
	ModifiedAt         *time.Time                 `json:"modified_at"`
	PlacedAt           *time.Time                 `json:"placed_at"`
	TradedAt           *time.Time                 `json:"traded_at"`
	TWAPExecutionState *TWAPExecutionState        `json:"twap_execution_state,omitempty"`
	UpdatedAt          time.Time                  `json:"updated_at"`
}

type OpenOrderEventAction int32

const (
	OpenOrderEventAction_Cancel OpenOrderEventAction = iota
	OpenOrderEventAction_Create
	OpenOrderEventAction_Fill
	OpenOrderEventAction_PartialFill
	OpenOrderEventAction_Update
	OpenOrderEventAction_CreatePartialFill
)

type PriceUpdateMessage struct {
	PublishTime     time.Time                   `json:"publishTime"`
	Price           shopspring_decimal.Decimal  `json:"price"`
	FundingRate     *shopspring_decimal.Decimal `json:"fundingRate,omitempty"`
	NextFundingTime *int64                      `json:"nextFundingTime,omitempty"` // unix ms of next UTC hour boundary
}

type MarketPriceMessage struct {
	Symbol          string                     `json:"symbol"`
	Timestamp       time.Time                  `json:"timestamp"`
	BestBidPrice    shopspring_decimal.Decimal `json:"best_bid_price"`
	BestAskPrice    shopspring_decimal.Decimal `json:"best_ask_price"`
	LastTradedPrice shopspring_decimal.Decimal `json:"last_traded_price"`
	LastTradedTime  time.Time                  `json:"last_traded_time"`
}

// FundingRateMessage represents a funding rate update message for NATS
type FundingRateMessage struct {
	// TODO: change this type to time.Time (or another strong time type)
	PublishTime  int64                      `json:"publish_time"`
	FundingRate  shopspring_decimal.Decimal `json:"funding_rate"`
	Symbol       string                     `json:"symbol"`
	IndexPrice   shopspring_decimal.Decimal `json:"index_price"`
	IsSettlement bool                       `json:"is_settlement"` // true = hourly settlement with user charges, false = sample for display only
}

// FundingRateBalanceUpdate represents a single account's funding rate balance update
type FundingRateBalanceUpdate struct {
	SubAccountId   SubAccountId               `json:"sub_account_id"`
	FundingPayment shopspring_decimal.Decimal `json:"funding_payment"` // Positive = receive funding, Negative = pay funding
	PositionSize   shopspring_decimal.Decimal `json:"position_size"`   // Position size at funding time (signed: positive = long, negative = short)
	PositionID     uint64                     `json:"position_id"`
	Symbol         string                     `json:"symbol"`
}

// FundingRateBalanceUpdateMessage represents batched funding rate balance updates for a symbol
type FundingRateBalanceUpdateMessage struct {
	Symbol      string                     `json:"symbol"`
	FundingRate shopspring_decimal.Decimal `json:"funding_rate"`
	IndexPrice  shopspring_decimal.Decimal `json:"index_price"`
	// TODO: change this type to time.Time (or another strong time type)
	PublishTime int64                      `json:"publish_time"`
	Updates     []FundingRateBalanceUpdate `json:"updates"`
}

// SubaccountCreatedResponse represents the response when a subaccount is created
// Contains leverages and initial collateral information
type SubaccountCreatedResponse struct {
	AdjustedAccountValue shopspring_decimal.Decimal `json:"adjusted_account_value"`
	AccountValue         shopspring_decimal.Decimal `json:"account_value"`
	AvailableMargin      shopspring_decimal.Decimal `json:"available_margin"`
	Collaterals          []CollateralState          `json:"collaterals,omitempty"`
	Error                *string                    `json:"error,omitempty"`
	Leverages            map[string]uint32          `json:"leverages"`
	RequestID            string                     `json:"request_id"`
	SubAccountId         SubAccountId               `json:"sub_account_id"`
}

// UpdateLeverageRequest represents a request to update leverage for a specific market
type UpdateLeverageRequest struct {
	Leverage     uint32       `json:"leverage"`
	RequestID    string       `json:"request_id"`
	SubAccountId SubAccountId `json:"sub_account_id"`
	Symbol       string       `json:"symbol"`
}

// UpdateLeverageResponse represents the response from a leverage update request
type UpdateLeverageResponse struct {
	Error        string       `json:"error,omitempty"`
	NewLeverage  uint32       `json:"new_leverage"`
	OldLeverage  uint32       `json:"old_leverage"`
	RequestID    string       `json:"request_id"`
	SubAccountId SubAccountId `json:"sub_account_id"`
	Success      bool         `json:"success"`
	Symbol       string       `json:"symbol"`
}

type FeeInfo struct {
	MakerFeeRate       shopspring_decimal.Decimal `json:"maker_fee_rate"`
	MaxBorrowCapacity  shopspring_decimal.Decimal `json:"max_borrow_capacity"`
	MaxOrdersPerMarket int64                      `json:"max_orders_per_market"`
	MaxTotalOrders     int64                      `json:"max_total_orders"`
	TakerFeeRate       shopspring_decimal.Decimal `json:"taker_fee_rate"`
}

type UpdateSubAccountFeeRates struct {
	FeeRates  map[SubAccountId]FeeInfo `json:"data"`
	UpdatedAt time.Time                `json:"updated_at"`
}
