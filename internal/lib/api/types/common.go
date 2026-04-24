// API-specific strong types.
//
// The following types are defined in separate files:
// - `Timestamp`;

package types

import (
	"errors"
	"fmt"
	"strconv"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// =========================================================================
// Constants
// =========================================================================

const (
	// NOTE: these values explicitly specified as an _untyped_ integer

	PaymentMaximumValidValue    = 0x8000_0000_0000_0000 - 2
	SubAccountMaximumValidValue = 0x8000_0000_0000_0000 - 2
	TradeMaximumValidValue      = 0x8000_0000_0000_0000 - 2
)

var (
	errDirectionUnrecognised = errors.New("direction unrecognised")

	errOrderTypeUnrecognised = errors.New("order type unrecognised")

	errPaymentIdTooLarge = errors.New("payment id too large")

	errPositionSideUnrecognised = errors.New("position side unrecognised")

	errSideUnrecognised = errors.New("side unrecognised")

	errSubAccountIdCannotBeNegative = errors.New("subaccount id cannot be negative")
	errSubAccountIdCannotBeZero     = errors.New("subaccount id cannot be zero")
	errSubAccountIdTooLarge         = errors.New("subaccount id too large")

	errTimeInForceUnrecognised = errors.New("time-in-force unrecognised")

	errTradeIdTooLarge = errors.New("trade id too large")

	errTriggerPriceTypeUnrecognised = errors.New("trigger-price type unrecognised")

	errVIOLATIONUnexpectedType = errors.New("VIOLATION: unexpected type")
)

// =========================================================================
// Types
// =========================================================================

type Direction string

const (
	Direction_short      Direction = "short"
	Direction_long       Direction = "long"
	Direction_closeShort Direction = "closeShort"
	Direction_closeLong  Direction = "closeLong"
)

type DelegationEventType string

const (
	DelegationEventType_delegationAdded   DelegationEventType = "delegationAdded"
	DelegationEventType_delegationRevoked DelegationEventType = "delegationRevoked"
)

type Fee string

type FeeRate string

type DepositEventType string

type FundingEventType string

type FundingRate string

const (
	DepositEventType_depositCredited DepositEventType = "depositCredited"
	DepositEventType_depositReceived DepositEventType = "depositReceived"
	FundingEventType_funding         FundingEventType = "funding"
)

type InterestRate = shopspring_decimal.Decimal

type LiquidationEventType string

const (
	LiquidationEventType_liquidation LiquidationEventType = "liquidation"
)

type MarginSummaryEventType string

const (
	MarginSummaryEventType_marginUpdate MarginSummaryEventType = "marginUpdate"
)

type OrderEventType string

const (
	OrderEventType_orderCancelled       OrderEventType = "orderCancelled"
	OrderEventType_orderFilled          OrderEventType = "orderFilled"
	OrderEventType_orderModified        OrderEventType = "orderModified"
	OrderEventType_orderPlaced          OrderEventType = "orderPlaced"
	OrderEventType_orderPartiallyFilled OrderEventType = "orderPartiallyFilled"
	OrderEventType_orderRejected        OrderEventType = "orderRejected"
)

type OrderType string

const (
	OrderType_limit            OrderType = "limit"
	OrderType_market           OrderType = "market"
	OrderType_stopMarket       OrderType = "stopMarket"
	OrderType_takeProfitMarket OrderType = "takeProfitMarket"
	OrderType_stopLimit        OrderType = "stopLimit"
	OrderType_takeProfitLimit  OrderType = "takeProfitLimit"
)

type PNL string

type PositionSide string

const (
	PositionSide_short PositionSide = "short"
	PositionSide_long  PositionSide = "long"
)

type Payment string

type PaymentId string

type Side string

const (
	Side_buy  Side = "buy"
	Side_sell Side = "sell"
)

type Status string

type SubAccountId string

const (
	SubAccountId_Empty SubAccountId = ""
	SubAccountId_Zero  SubAccountId = "0"
)

type CollateralExchangeEventType string

const (
	CollateralExchangeEventType_collateralExchange CollateralExchangeEventType = "collateralExchange"
)

type TakeoverEventType string

const (
	TakeoverEventType_subAccountTakeover TakeoverEventType = "subAccountTakeover"
)

// The time in force for an order.
type TimeInForce string

const (
	TimeInForce_FOK TimeInForce = "FOK" // Fill Or Kill
	TimeInForce_GTC TimeInForce = "GTC" // Good 'Til Canceled
	TimeInForce_GTD TimeInForce = "GTD" // Good Til Date
	TimeInForce_IOC TimeInForce = "IOC" // Immediate Or Cancel
)

type TradeEventType string

const (
	TradeEventType_trade TradeEventType = "trade"
)

type TradeId string

type TriggerPriceType string

const (
	TriggerPriceType_last TriggerPriceType = "last"
	TriggerPriceType_mark TriggerPriceType = "mark"
)

type Volume = shopspring_decimal.Decimal

type WickInsuranceEventType string

const (
	WickInsuranceEventType_positionIncreased   WickInsuranceEventType = "wickInsurancePositionIncreased"
	WickInsuranceEventType_protectionActivated WickInsuranceEventType = "wickInsuranceProtectionActivated"
	WickInsuranceEventType_protectionCompleted WickInsuranceEventType = "wickInsuranceProtectionCompleted"
)

type WithdrawalEventType string

const (
	WithdrawalEventType_withdrawal WithdrawalEventType = "withdrawal"
)

type WithdrawalStatusEventType string

const (
	WithdrawalStatusEventType_withdrawalStatus WithdrawalStatusEventType = "withdrawalStatus"
)

// =========================================================================
// Utility functions
// =========================================================================

func itoa(
	v int64,
) string {
	return strconv.FormatInt(v, 10)
}

func uitoa(
	v uint64,
) string {
	return strconv.FormatUint(v, 10)
}

func stringFromDecimalUnvalidated(
	v shopspring_decimal.Decimal,
) string {
	return v.String()
}

func stringFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) string {
	if v == shopspring_decimal.Zero {
		return ""
	}

	return stringFromDecimalUnvalidated(v)
}

// ===========================
// `Direction`
// ===========================

func _DirectionFromCoreDirection(
	v snx_lib_core.Direction,
) (Direction, bool) {

	s, known := snx_lib_core.DirectionToString(v)

	return Direction(s), known
}

func DirectionFromCoreDirection(
	v snx_lib_core.Direction,
) (Direction, error) {

	if r, recognised := _DirectionFromCoreDirection(v); recognised {
		return r, nil
	} else {
		return r, errDirectionUnrecognised
	}
}

func DirectionFromCoreDirectionUnvalidated(
	v snx_lib_core.Direction,
) Direction {
	r, _ := _DirectionFromCoreDirection(v)

	return r
}

// ===========================
// `FundingRate`
// ===========================

func FundingRateFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) FundingRate {
	if v == shopspring_decimal.Zero {
		return FundingRate("")
	}

	return FundingRate(v.String())
}

// ===========================
// `Fee`
// ===========================

// Obtains, from a decimal value, a fee string equivalent, or a blank in the
// case that the value is zero.
func FeeFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) Fee {
	return Fee(stringFromDecimalOrBlankWhenZero(v))
}

// ===========================
// `FeeRate`
// ===========================

// Obtains, from a decimal value, a fee-rate string equivalent, or a blank
// in the case that the value is zero.
func FeeRateFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) FeeRate {
	return FeeRate(stringFromDecimalOrBlankWhenZero(v))
}

// ===========================
// `OrderType`
// ===========================

func _OrderTypeFromCoreOrderType(
	v snx_lib_core.OrderType,
) (OrderType, bool) {
	switch v {
	case snx_lib_core.OrderTypeLimit:
		return OrderType_limit, true
	case snx_lib_core.OrderTypeMarket:
		return OrderType_market, true
	case snx_lib_core.OrderTypeStopMarket:
		return OrderType_stopMarket, true
	case snx_lib_core.OrderTypeTakeProfitMarket:
		return OrderType_takeProfitMarket, true
	case snx_lib_core.OrderTypeStopLimit:
		return OrderType_stopLimit, true
	case snx_lib_core.OrderTypeTakeProfitLimit:
		return OrderType_takeProfitLimit, true
	default:
		return OrderType(fmt.Sprintf("UNKNOWN-OrderType<v=%v>", v)), false
	}
}

func OrderTypeFromCoreOrderType(
	v snx_lib_core.OrderType,
) (OrderType, error) {

	if r, recognised := _OrderTypeFromCoreOrderType(v); recognised {
		return r, nil
	} else {
		return r, errOrderTypeUnrecognised
	}
}

func OrderTypeFromCoreOrderTypeUnvalidated(
	v snx_lib_core.OrderType,
) OrderType {
	r, _ := _OrderTypeFromCoreOrderType(v)

	return r
}

// ===========================
// `Payment`
// ===========================

func PaymentFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) Payment {
	if v == shopspring_decimal.Zero {
		return Payment("")
	}

	return Payment(v.String())
}

// ===========================
// `PaymentId`
// ===========================

func PaymentIdFromUint(
	v uint64,
) (PaymentId, error) {
	switch {
	// case v == 0:

	// 	return "", errPaymentIdCannotBeZero
	case v > PaymentMaximumValidValue:

		return "", errPaymentIdTooLarge
	default:

		return PaymentIdFromUintUnvalidated(v), nil
	}
}

func PaymentIdFromUintRaw(
	v uint64,
) string {
	return uitoa(v)
}

func PaymentIdFromUintUnvalidated(
	v uint64,
) PaymentId {
	return PaymentId(PaymentIdFromUintRaw(v))
}

// ===========================
// `PositionSide`
// ===========================

func _PositionSideFromCorePositionSide(
	v snx_lib_core.PositionSide,
) (PositionSide, bool) {
	switch v {
	case snx_lib_core.PositionSideShort:
		return PositionSide_short, true
	case snx_lib_core.PositionSideLong:
		return PositionSide_long, true
	default:
		return PositionSide(fmt.Sprintf("UNKNOWN-PositionSide<v=%v>", v)),
			false
	}
}

func PositionSideFromCorePositionSide(
	v snx_lib_core.PositionSide,
) (PositionSide, error) {
	if side, recognised := _PositionSideFromCorePositionSide(v); recognised {
		return side, nil
	} else {
		return side, errPositionSideUnrecognised
	}
}

func PositionSideFromCorePositionSideUnvalidated(
	v snx_lib_core.PositionSide,
) PositionSide {
	side, _ := _PositionSideFromCorePositionSide(v)

	return side
}

// ===========================
// `PNL`
// ===========================

func PNLFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) PNL {
	return PNL(stringFromDecimalOrBlankWhenZero(v))
}

// ===========================
// `Side`
// ===========================

func _SideFromCoreDirection(
	v snx_lib_core.Direction,
) (Side, bool) {
	switch v {
	case snx_lib_core.Direction_Short:
		return Side_sell, true
	case snx_lib_core.Direction_Long:
		return Side_buy, true
	case snx_lib_core.Direction_CloseShort:
		return Side_buy, true
	case snx_lib_core.Direction_CloseLong:
		return Side_sell, true
	default:
		return Side(fmt.Sprintf("UNKNOWN-Side<v=%v>", v)),
			false
	}
}

func SideFromCoreDirection(
	v snx_lib_core.Direction,
) (Side, error) {
	if side, recognised := _SideFromCoreDirection(v); recognised {
		return side, nil
	} else {
		return side, errSideUnrecognised
	}
}

func SideFromCoreDirectionUnvalidated(
	v snx_lib_core.Direction,
) Side {
	side, _ := _SideFromCoreDirection(v)

	return side
}

// ===========================
// `Side`
// ===========================

func _SideFromCoreSide(
	v snx_lib_core.OrderSide,
) (Side, bool) {
	switch v {
	case snx_lib_core.OrderSideLong:
		return Side_buy, true
	case snx_lib_core.OrderSideShort:
		return Side_sell, true
	default:
		return Side(fmt.Sprintf("UNKNOWN-SIDE<v=%v>", v)), false
	}
}

func SideFromCoreSide(
	v snx_lib_core.OrderSide,
) (Side, error) {
	if r, recognised := _SideFromCoreSide(v); recognised {
		return r, nil
	} else {
		return r, errSideUnrecognised
	}
}

func SideFromCoreSideUnvalidated(
	v snx_lib_core.OrderSide,
) Side {
	r, _ := _SideFromCoreSide(v)

	return r
}

// ===========================
// `SubAccountId`
// ===========================

func SubAccountIdFromInt(
	v int64,
) (SubAccountId, error) {
	switch {
	case v < 0:

		return "", errSubAccountIdCannotBeNegative
	case v == 0:

		return "", errSubAccountIdCannotBeZero
	case v > SubAccountMaximumValidValue:

		return "", errSubAccountIdTooLarge
	default:

		return SubAccountIdFromIntUnvalidated(v), nil
	}
}

func SubAccountIdFromIntRaw(
	v int64,
) string {
	return itoa(v)
}

func SubAccountIdFromIntUnvalidated(
	v int64,
) SubAccountId {
	return SubAccountId(SubAccountIdFromIntRaw(v))
}

func SubAccountIdToCoreSubaccountId(
	v SubAccountId,
) (r snx_lib_core.SubAccountId, err error) {

	var i int64
	i, err = strconv.ParseInt(string(v), 10, 64)
	if err == nil {
		r = snx_lib_core.SubAccountId(i)
	}

	return
}

// ===========================
// `TimeInForce`
// ===========================

// Converts an internal representation of time-in-force to API type value.
func TimeInForceFromGRPC(
	v v4grpc.TimeInForce,
) (TimeInForce, error) {
	switch v {
	case v4grpc.TimeInForce_FOK:
		return TimeInForce_FOK, nil
	case v4grpc.TimeInForce_GTC:
		return TimeInForce_GTC, nil
	case v4grpc.TimeInForce_GTD:
		return TimeInForce_GTD, nil
	case v4grpc.TimeInForce_IOC:
		return TimeInForce_IOC, nil
	default:
		return "", errTimeInForceUnrecognised
	}
}

// ===========================
// `TradeId`
// ===========================

func TradeIdFromUint(
	v uint64,
) (TradeId, error) {
	switch {
	// case v == 0:

	// 	return "", errTradeIdCannotBeZero
	case v > TradeMaximumValidValue:

		return "", errTradeIdTooLarge
	default:

		return TradeIdFromUintUnvalidated(v), nil
	}
}

func TradeIdFromUintRaw(
	v uint64,
) string {
	return uitoa(v)
}

func TradeIdFromUintUnvalidated(
	v uint64,
) TradeId {
	return TradeId(TradeIdFromUintRaw(v))
}

// ===========================
// `TriggerPriceType`
// ===========================

// Validates a raw trigger-price-type string and returns the API strong type.
// Exact match only; empty defaults to "mark". Rejects unrecognised values.
func APITriggerPriceTypeFromString(
	v string,
) (TriggerPriceType, error) {

	switch v {
	case "", "mark":
		return TriggerPriceType_mark, nil
	case "last":
		return TriggerPriceType_last, nil
	default:
		return "", errTriggerPriceTypeUnrecognised
	}
}
