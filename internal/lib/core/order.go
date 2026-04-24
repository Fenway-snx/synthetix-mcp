package core

import (
	"fmt"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// =========================================================================
// Constants
// =========================================================================

// =========================================================================
// Types
// =========================================================================

// Holds runtime state for resting or historical orders. Timestamp fields
// may alias the same *time.Time: you may reassign a field (including nil)
// to refer to a different instant; do not mutate the pointed-to time
// through a field (e.g. *order.CreatedAt).
type Order struct {
	OrderId           OrderId
	Symbol            string // Symbol name (e.g., "BTC-USD", "ETH/USDC")
	Side              OrderSide
	Type              OrderType
	OriginalType      OrderType // Set once at creation; never mutated after conditional-to-regular conversion
	Direction         Direction
	TimeInForce       v4grpc.TimeInForce // ALO, IOC, or GTC
	Quantity          shopspring_decimal.Decimal
	RemainingQuantity shopspring_decimal.Decimal
	Price             shopspring_decimal.Decimal // Limit price (for LIMIT orders)
	TriggerPrice      shopspring_decimal.Decimal // Trigger price (for conditional/pending orders)
	TriggerPriceType  TriggerPriceType
	ReduceOnly        bool
	State             OrderState
	PositionID        *uint64 // Position ID (for conditional orders)
	TakeProfit        *OrderId
	StopLoss          *OrderId
	IsActive          bool // This is only for conditional/pending orders
	TriggerSide       *ConditionalTriggerSide
	PostOnly          bool
	ClosePosition     bool
	Source            string
	CancelledAt       *time.Time
	CreatedAt         *time.Time
	ExpiresAt         *time.Time
	ModifiedAt        *time.Time
	PlacedAt          *time.Time
	RejectedAt        *time.Time
	TradedAt          *time.Time
	TriggeredAt       *time.Time  // TODO: will handle/save this one later on
	TWAPConfig        *TWAPConfig `json:"twap_config,omitempty"`
	CancelReason      OrderCancelReason
}

type OrderSide int32

const (
	OrderSideShort OrderSide = iota
	OrderSideLong
)

// Distinguishes opening vs closing legs relative to long and short
// exposure. Numeric values are stable for storage and on-the-wire enums.
type Direction int32

const (
	Direction_Short      Direction = iota // Opening or adding to short exposure (sell-side entry in typical books).
	Direction_Long                        // Opening or adding to long exposure (buy-side entry in typical books).
	Direction_CloseShort                  // Closing or reducing an existing short (buy-to-cover in typical books).
	Direction_CloseLong                   // Closing or reducing an existing long (sell-to-flatten in typical books).
)

// CancelReason or CancellationReason?
type OrderCancelReason string

const (
	OrderCancelReason_ADL         OrderCancelReason = "ADL"
	OrderCancelReason_Expired     OrderCancelReason = "EXPIRED"
	OrderCancelReason_MMRBreach   OrderCancelReason = "MMR_BREACH"
	OrderCancelReason_RMRBreach   OrderCancelReason = "RMR_BREACH"
	OrderCancelReason_SLPTakeover OrderCancelReason = "SLP_TAKEOVER"
	OrderCancelReason_Zero        OrderCancelReason = ""
)

// IsClosePositionDirection checks if the direction represents closing a position
func IsClosePositionDirection(direction *Direction) bool {
	return direction != nil && (*direction == Direction_CloseShort || *direction == Direction_CloseLong)
}

type TriggerPriceType int32

const (
	TriggerPriceTypeMarkPrice TriggerPriceType = iota
	TriggerPriceTypeLastPrice
)

// Converts a trigger-price-type wire string to the core enum.
// Returns an error for unrecognised values.
func CoreTriggerPriceTypeFromString(v string) (TriggerPriceType, error) {
	switch v {
	case "last":
		return TriggerPriceTypeLastPrice, nil
	case "mark":
		return TriggerPriceTypeMarkPrice, nil
	default:
		// panic(fmt.Sprintf("VIOLATION: unexpected trigger price type string '%s'", v))
		return -1, fmt.Errorf("%w: '%s'", ErrInvalidTriggerPriceType, v)
	}
}

type OrderState int32

const (
	OrderStateStarted OrderState = iota
	OrderStatePlaced
	OrderStatePartiallyFilled
	OrderStateFilled
	OrderStateRejected
	OrderStateModify
	OrderStateModified
	OrderStateCancel
	OrderStateCancelled
)

func (os OrderState) String() string {
	switch os {
	case OrderStateStarted:
		return "OrderStateStarted"
	case OrderStatePlaced:
		return "OrderStatePlaced"
	case OrderStatePartiallyFilled:
		return "OrderStatePartiallyFilled"
	case OrderStateFilled:
		return "OrderStateFilled"
	case OrderStateRejected:
		return "OrderStateRejected"
	case OrderStateModify:
		return "OrderStateModify"
	case OrderStateModified:
		return "OrderStateModified"
	case OrderStateCancel:
		return "OrderStateCancel"
	case OrderStateCancelled:
		return "OrderStateCancelled"
	default:
		return fmt.Sprintf("OrderState(%d)", int32(os))
	}
}

// represents the execution type of an order
type OrderType int32

const (
	OrderTypeLimit OrderType = iota
	OrderTypeMarket
	OrderTypeStopMarket
	OrderTypeTakeProfitMarket
	OrderTypeStopLimit
	OrderTypeTakeProfitLimit
	OrderTypeTWAP
)

// returns true if the order type uses a limit price
func (ot OrderType) HasLimitPrice() bool {
	switch ot {
	case
		OrderTypeLimit,
		OrderTypeStopLimit,
		OrderTypeTakeProfitLimit:

		return true
	}

	return false
}

// returns true if the order type uses a trigger price
func (ot OrderType) HasTriggerPrice() bool {

	switch ot {
	case
		OrderTypeStopLimit,
		OrderTypeStopMarket,
		OrderTypeTakeProfitLimit,
		OrderTypeTakeProfitMarket:

		return true
	}

	return false
}

// =========================================================================
// API functions
// =========================================================================

// TODO: reorder the following sections at an appropriate time

// ===========================
// Utility functions
// ===========================

// ===========================
// `Direction`
// ===========================

// Converts a Direction to a canonical wire string when the value is one of
// the defined variants (short, long, closeShort, closeLong). For any other
// value, returns a diagnostic string of the form UNKNOWN-Direction<v=N> and
// sets recognised to false.
func DirectionToString(
	v Direction,
) (r string, recognised bool) {

	switch v {
	case Direction_Short:

		return "short", true
	case Direction_Long:

		return "long", true
	case Direction_CloseShort:

		return "closeShort", true
	case Direction_CloseLong:

		return "closeLong", true
	default:

		return fmt.Sprintf("UNKNOWN-Direction<v=%v>", v), false
	}
}

// ===========================
// misc.
// ===========================

func ParseGRCPOrderType(orderT v4grpc.OrderType) (OrderType, error) {
	switch orderT {
	case v4grpc.OrderType_LIMIT:
		return OrderTypeLimit, nil
	case v4grpc.OrderType_MARKET:
		return OrderTypeMarket, nil
	case v4grpc.OrderType_STOP:
		return OrderTypeStopLimit, nil
	case v4grpc.OrderType_STOP_MARKET:
		return OrderTypeStopMarket, nil
	case v4grpc.OrderType_TAKE_PROFIT:
		return OrderTypeTakeProfitLimit, nil
	case v4grpc.OrderType_TAKE_PROFIT_MARKET:
		return OrderTypeTakeProfitMarket, nil
	case v4grpc.OrderType_TWAP:
		return OrderTypeTWAP, nil
	default:
		return -1, errInvalidOrderType
	}
}

func ParseGRPCOrderSide(orderS v4grpc.Side) (OrderSide, error) {
	switch orderS {
	case v4grpc.Side_BUY:
		return OrderSideLong, nil
	case v4grpc.Side_SELL:
		return OrderSideShort, nil
	default:
		return -1, ErrInvalidOrderSide
	}
}

func OrderTypeToGRPC(orderT OrderType) (v4grpc.OrderType, error) {
	switch orderT {
	case OrderTypeLimit:
		return v4grpc.OrderType_LIMIT, nil
	case OrderTypeMarket:
		return v4grpc.OrderType_MARKET, nil
	case OrderTypeStopLimit:
		return v4grpc.OrderType_STOP, nil
	case OrderTypeStopMarket:
		return v4grpc.OrderType_STOP_MARKET, nil
	case OrderTypeTakeProfitLimit:
		return v4grpc.OrderType_TAKE_PROFIT, nil
	case OrderTypeTakeProfitMarket:
		return v4grpc.OrderType_TAKE_PROFIT_MARKET, nil
	case OrderTypeTWAP:
		return v4grpc.OrderType_TWAP, nil
	default:
		return -1, errInvalidOrderType
	}
}

func OrderSideToGRPC(orderS OrderSide) (v4grpc.Side, error) {
	switch orderS {
	case OrderSideLong:
		return v4grpc.Side_BUY, nil
	case OrderSideShort:
		return v4grpc.Side_SELL, nil
	default:
		return -1, ErrInvalidOrderSide
	}
}

func IsMarketOrder(orderType OrderType) bool {
	switch orderType {
	case
		OrderTypeMarket,
		OrderTypeStopMarket,
		OrderTypeTakeProfitMarket:
		return true
	case
		OrderTypeLimit,
		OrderTypeStopLimit,
		OrderTypeTakeProfitLimit:
		return false
	default:
		return false
	}
}

// IsConditionalOrder checks if an order type is a conditional order (TP/SL)
func IsConditionalOrder(orderType OrderType) bool {
	switch orderType {
	case
		OrderTypeStopLimit,
		OrderTypeStopMarket,
		OrderTypeTakeProfitLimit,
		OrderTypeTakeProfitMarket:
		return true
	}
	return false
}

func GetStopLossOrTakeProfitDirection(side OrderSide) Direction {
	if side == OrderSideShort {
		return Direction_CloseShort
	}
	return Direction_CloseLong
}

func IsTakeProfit(orderType OrderType) bool {
	switch orderType {
	case
		OrderTypeTakeProfitLimit,
		OrderTypeTakeProfitMarket:
		return true
	default:
		return false
	}
}

func IsStopLoss(orderType OrderType) bool {
	switch orderType {
	case
		OrderTypeStopLimit,
		OrderTypeStopMarket:
		return true
	default:
		return false
	}
}

func IsRestingOrder(orderType OrderType, timeInForce v4grpc.TimeInForce) bool {
	if IsMarketOrder(orderType) {
		return false
	}

	switch timeInForce {
	case
		v4grpc.TimeInForce_FOK,
		v4grpc.TimeInForce_IOC:
		return false
	default:
		return true
	}
}
