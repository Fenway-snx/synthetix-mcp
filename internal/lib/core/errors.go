package core

import (
	"errors"

	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
)

// Error carrying a structured origin code.
type CodedError struct {
	msg  string
	code snx_lib_status_codes.ErrorCode
}

// Creates a new CodedError with the given message and error code.
func NewCodedError(msg string, code snx_lib_status_codes.ErrorCode) *CodedError {
	return &CodedError{msg: msg, code: code}
}

func (e *CodedError) Error() string                        { return e.msg }
func (e *CodedError) Code() snx_lib_status_codes.ErrorCode { return e.code }

// Extracts a structured code from an error chain, or returns a fallback.
func ErrorCodeFrom(err error, fallback ...snx_lib_status_codes.ErrorCode) snx_lib_status_codes.ErrorCode {
	if err == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	var coded *CodedError
	if errors.As(err, &coded) {
		return coded.Code()
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

var (
	// Position errors
	ErrPositionNotFound = NewCodedError("position not found", snx_lib_status_codes.ErrorCodePositionNotFound)

	// Order errors — coded sentinels carry their error code from birth
	Err_Order_BlockedDuringWickInsurance        = NewCodedError("orders blocked during wick insurance protection", snx_lib_status_codes.ErrorCodeWickInsuranceActive)
	Err_Order_ConditionOrderWrongSide           = NewCodedError("conditional order must have opposite side to position", snx_lib_status_codes.ErrorCodeInvalidOrderSide)
	Err_Order_FailedValidation                  = errors.New("order validation failed") // NOT coded — used as outer %w wrapper; inner error carries the real code
	Err_Order_InvalidReduceOnly_ExceedsPosition = NewCodedError("reduce-only orders would exceed position size", snx_lib_status_codes.ErrorCodeReduceOnlyWouldIncrease)
	Err_Order_MaxOrdersPerMarketExceeded        = NewCodedError("max open orders per market exceeded", snx_lib_status_codes.ErrorCodeMaxOrdersPerMarket)
	Err_Order_MaxTotalOrdersExceeded            = NewCodedError("max total open orders exceeded", snx_lib_status_codes.ErrorCodeMaxTotalOrders)
	Err_Order_InvalidReduceOnly_NoPosition      = NewCodedError("no open position found, cannot place reduce-only order", snx_lib_status_codes.ErrorCodeReduceOnlyNoPosition)
	Err_Order_InvalidReduceOnly_SameSide        = NewCodedError("same side position found, reduce-only order failed", snx_lib_status_codes.ErrorCodeReduceOnlySameSide)
	ErrConditionalOrdersError                   = NewCodedError("please provide at least a stop loss or take profit order", snx_lib_status_codes.ErrorCodeValidationError)
	ErrDifferentSymbols                         = errors.New("all orders must have the same symbol")
	ErrDuplicateClientOrderId                   = NewCodedError("duplicate clientOrderId", snx_lib_status_codes.ErrorCodeIdempotencyConflict)
	ErrInsufficientMargin                       = NewCodedError("insufficient margin", snx_lib_status_codes.ErrorCodeInsufficientMargin)
	ErrInvalidOrderSide                         = NewCodedError("invalid order side", snx_lib_status_codes.ErrorCodeInvalidOrderSide)
	errInvalidOrderType                         = errors.New("invalid order type")
	ErrInvalidTriggerPrice                      = NewCodedError("trigger price must be provided for conditional orders", snx_lib_status_codes.ErrorCodeInvalidTriggerPrice)
	ErrInvalidTriggerPriceType                  = errors.New("invalid trigger price type")
	ErrMarketClosed                             = NewCodedError("market is closed", snx_lib_status_codes.ErrorCodeMarketClosed)
	ErrMarketNotOpen                            = NewCodedError("market is not open", snx_lib_status_codes.ErrorCodeMarketClosed)
	ErrOrderNotFound                            = NewCodedError("order not found", snx_lib_status_codes.ErrorCodeOrderNotFound)
	ErrOrderRejected                            = NewCodedError("order rejected by matching engine", snx_lib_status_codes.ErrorCodeOrderRejectedByEngine)
	ErrQuantityBelowFilled                      = NewCodedError("quantity below filled amount", snx_lib_status_codes.ErrorCodeQuantityBelowFilled)
	ErrQuantityBelowMinimum                     = NewCodedError("quantity below minimum", snx_lib_status_codes.ErrorCodeQuantityTooSmall)

	// Account errors
	ErrInvalidSubAccountId         = NewCodedError("invalid sub account id", snx_lib_status_codes.ErrorCodeInvalidValue)
	ErrSubAccountInvalidName       = errors.New("invalid subaccount name")
	ErrSubAccountIsMaster          = errors.New("cannot modify master account")
	ErrSubAccountLimitExceeded     = errors.New("sub-account limit exceeded for current tier")
	ErrSubAccountNameAlreadyExists = errors.New("subaccount name already exists")
	ErrSubAccountNotFound          = errors.New("subaccount not found")

	// Market errors
	ErrInvalidSymbol     = errors.New("invalid symbol")
	ErrMarketCannotBeNil = errors.New("market cannot be nil")
	ErrMarketNotFound    = NewCodedError("market not found", snx_lib_status_codes.ErrorCodeMarketNotFound)

	// General errors
	ErrActorNotFound            = errors.New("actor was not found")
	ErrContextCancelled         = NewCodedError("context cancelled", snx_lib_status_codes.ErrorCodeOperationTimeout)
	ErrContextTimeout           = NewCodedError("context timeout", snx_lib_status_codes.ErrorCodeOperationTimeout)
	ErrInvalidActorType         = errors.New("invalid actor")
	ErrInvalidPrice             = NewCodedError("invalid price", snx_lib_status_codes.ErrorCodeInvalidValue)
	ErrInvalidQuantity          = NewCodedError("invalid quantity", snx_lib_status_codes.ErrorCodeInvalidValue)
	ErrInvalidRequestId         = errors.New("invalid request id")
	ErrTradingServiceDraining   = errors.New("trading is draining, not accepting new orders")
	ErrTradingServiceIsNotReady = errors.New("trading is not ready, still recovering")

	// Request validation errors
	ErrActionPayloadRequired     = errors.New("action payload is required")
	ErrAmountInvalid             = errors.New("amount must be a positive value")
	ErrDestinationInvalidAddress = errors.New("destination must be a valid Ethereum address")
	ErrDestinationRequired       = errors.New("destination address is required")
	ErrInvalidGrouping           = errors.New("invalid grouping")
	ErrInvalidModifyOrderPayload = errors.New("please provide at least one of the following: price, quantity, trigger price") // market orders ignore price
	ErrInvalidNumberOfOrders     = errors.New("invalid number of orders")
	ErrNoOrdersProvided          = errors.New("no orders provided")
	ErrOrderIdsMustBeNonempty    = errors.New("orderIds must be nonempty")
	ErrOrdersArrayEmpty          = errors.New("orders array cannot be empty")
	ErrRequestIdRequired         = errors.New("request id is required")
	ErrSubAccountIdRequired      = NewCodedError("sub account id is required", snx_lib_status_codes.ErrorCodeInvalidValue)
	ErrSymbolRequired            = errors.New("symbol is required")
	ErrSymbolsMustBeNonempty     = errors.New("symbols must be nonempty")
	ErrPriceMustBePositive       = errors.New("price must be a positive value")
	ErrPriceMustBeValidDecimal   = errors.New("price must be a valid decimal number")
	ErrAmountMustBeValidDecimal  = errors.New("amount must be a valid decimal number")
	ErrLeverageMustBePositive    = errors.New("leverage must be a positive integer")
	ErrOrderIdMustBePositive     = errors.New("orderId must be a positive integer")
)

// Carries sub-account limit context for enriched error responses.
type SubAccountLimitError struct {
	CurrentCount int64
	MaxAllowed   int64
	TierName     string
}

func (e *SubAccountLimitError) Error() string {
	return ErrSubAccountLimitExceeded.Error()
}

func (e *SubAccountLimitError) Is(target error) bool {
	return target == ErrSubAccountLimitExceeded
}
