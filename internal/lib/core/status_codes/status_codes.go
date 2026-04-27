// Package status_codes defines standardized error codes, error categories,
// and classification logic used across all API-facing services.
package status_codes

// ErrorCode represents a standardized error code for API responses.
// Using a type provides compile-time safety and better IDE support.
type ErrorCode string

// Error codes for different types of errors.
const (
	// Validation Errors (400)
	ErrorCodeValidationError      ErrorCode = "VALIDATION_ERROR"
	ErrorCodeMissingRequiredField ErrorCode = "MISSING_REQUIRED_FIELD"
	ErrorCodeInvalidFormat        ErrorCode = "INVALID_FORMAT"
	ErrorCodeInvalidValue         ErrorCode = "INVALID_VALUE"

	// Authentication Errors (401)
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden    ErrorCode = "FORBIDDEN"

	// Business Logic Errors (409/422)
	ErrorCodeMarketAlreadyExists ErrorCode = "MARKET_ALREADY_EXISTS"
	ErrorCodeAssetNotFound       ErrorCode = "ASSET_NOT_FOUND"
	ErrorCodeInvalidMarketConfig ErrorCode = "INVALID_MARKET_CONFIG"

	// Trading Errors (returned with HTTP 200, per-item rejection)
	ErrorCodeFOKNotFilled            ErrorCode = "FOK_NOT_FILLED"
	ErrorCodeIOCNotFilled            ErrorCode = "IOC_NOT_FILLED"
	ErrorCodeIdempotencyConflict     ErrorCode = "IDEMPOTENCY_CONFLICT"
	ErrorCodeInsufficientMargin      ErrorCode = "INSUFFICIENT_MARGIN"
	ErrorCodeInvalidOrderSide        ErrorCode = "INVALID_ORDER_SIDE"
	ErrorCodeInvalidTriggerPrice     ErrorCode = "INVALID_TRIGGER_PRICE"
	ErrorCodeMarketClosed            ErrorCode = "MARKET_CLOSED"
	ErrorCodeMarketNotFound          ErrorCode = "MARKET_NOT_FOUND"
	ErrorCodeMaxOrdersPerMarket      ErrorCode = "MAX_ORDERS_PER_MARKET"
	ErrorCodeMaxTotalOrders          ErrorCode = "MAX_TOTAL_ORDERS"
	ErrorCodeMaxSubAccountsExceeded  ErrorCode = "MAX_SUB_ACCOUNTS_EXCEEDED"
	ErrorCodeNoLiquidity             ErrorCode = "NO_LIQUIDITY"
	ErrorCodeOICapExceeded           ErrorCode = "OI_CAP_EXCEEDED"
	ErrorCodeOperationCancelFailed   ErrorCode = "CANCEL_FAILED"
	ErrorCodeOperationTimeout        ErrorCode = "OPERATION_TIMEOUT"
	ErrorCodeOrderNotFound           ErrorCode = "ORDER_NOT_FOUND"
	ErrorCodeOrderRejectedByEngine   ErrorCode = "ORDER_REJECTED_BY_ENGINE"
	ErrorCodePositionNotFound        ErrorCode = "POSITION_NOT_FOUND"
	ErrorCodePostOnlyWouldTrade      ErrorCode = "POST_ONLY_WOULD_TRADE"
	ErrorCodePriceOutOfBounds        ErrorCode = "PRICE_OUT_OF_BOUNDS"
	ErrorCodeQuantityBelowFilled     ErrorCode = "QUANTITY_BELOW_FILLED"
	ErrorCodeQuantityTooSmall        ErrorCode = "QUANTITY_TOO_SMALL"
	ErrorCodeReduceOnlyNoPosition    ErrorCode = "REDUCE_ONLY_NO_POSITION"
	ErrorCodeReduceOnlySameSide      ErrorCode = "REDUCE_ONLY_SAME_SIDE"
	ErrorCodeReduceOnlyWouldIncrease ErrorCode = "REDUCE_ONLY_WOULD_INCREASE"
	ErrorCodeSelfTradePrevented      ErrorCode = "SELF_TRADE_PREVENTED"
	ErrorCodeWickInsuranceActive     ErrorCode = "WICK_INSURANCE_ACTIVE"

	// Rate Limiting Errors (429)
	ErrorCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	// System Errors (500)
	ErrorCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrorCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrorCodeCacheError         ErrorCode = "CACHE_ERROR"
	ErrorCodeMethodNotAllowed   ErrorCode = "METHOD_NOT_ALLOWED"
	ErrorCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrorCodeRequestTimeout     ErrorCode = "REQUEST_TIMEOUT"
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeUnknownError       ErrorCode = "UNKNOWN_ERROR"
)

// Reports whether code is a member of the curated set of
// error codes defined in this package. It is intended for use at the
// external API boundary to gate forwarding of server-supplied reason
// strings (for example, `errdetails.ErrorInfo.Reason`) into client-facing
// `ErrorCode` values, so that upstream services cannot inject arbitrary
// codes into public responses.
func IsKnownErrorCode(code ErrorCode) bool {
	switch code {
	case ErrorCodeValidationError, ErrorCodeMissingRequiredField,
		ErrorCodeInvalidFormat, ErrorCodeInvalidValue,
		ErrorCodeUnauthorized, ErrorCodeForbidden,
		ErrorCodeMarketAlreadyExists, ErrorCodeAssetNotFound,
		ErrorCodeInvalidMarketConfig,
		ErrorCodeFOKNotFilled, ErrorCodeIOCNotFilled,
		ErrorCodeIdempotencyConflict, ErrorCodeInsufficientMargin,
		ErrorCodeInvalidOrderSide, ErrorCodeInvalidTriggerPrice,
		ErrorCodeMarketClosed, ErrorCodeMarketNotFound,
		ErrorCodeMaxOrdersPerMarket, ErrorCodeMaxTotalOrders,
		ErrorCodeMaxSubAccountsExceeded, ErrorCodeNoLiquidity,
		ErrorCodeOICapExceeded, ErrorCodeOperationCancelFailed,
		ErrorCodeOperationTimeout, ErrorCodeOrderNotFound,
		ErrorCodeOrderRejectedByEngine, ErrorCodePositionNotFound,
		ErrorCodePostOnlyWouldTrade, ErrorCodePriceOutOfBounds,
		ErrorCodeQuantityBelowFilled, ErrorCodeQuantityTooSmall,
		ErrorCodeReduceOnlyNoPosition, ErrorCodeReduceOnlySameSide,
		ErrorCodeReduceOnlyWouldIncrease, ErrorCodeSelfTradePrevented,
		ErrorCodeWickInsuranceActive,
		ErrorCodeRateLimitExceeded,
		ErrorCodeInternalError, ErrorCodeDatabaseError,
		ErrorCodeCacheError, ErrorCodeMethodNotAllowed,
		ErrorCodeNotFound, ErrorCodeRequestTimeout,
		ErrorCodeServiceUnavailable, ErrorCodeUnknownError:
		return true
	default:
		return false
	}
}

// ErrorCategory classifies the type of error for client handling.
type ErrorCategory string

const (
	ErrorCategoryRequest   ErrorCategory = "REQUEST"    // Validation, format errors
	ErrorCategoryAuth      ErrorCategory = "AUTH"       // Authentication/authorization
	ErrorCategoryRateLimit ErrorCategory = "RATE_LIMIT" // Rate limiting
	ErrorCategoryTrading   ErrorCategory = "TRADING"    // Business logic rejections
	ErrorCategorySystem    ErrorCategory = "SYSTEM"     // Internal errors
)

// CategorizeError determines the category and retryability based on error code.
func CategorizeError(code ErrorCode) (ErrorCategory, bool) {
	switch code {
	// Request validation errors - not retryable
	case ErrorCodeValidationError, ErrorCodeMissingRequiredField,
		ErrorCodeInvalidFormat, ErrorCodeInvalidValue:
		return ErrorCategoryRequest, false

	// Auth errors - not retryable
	case ErrorCodeUnauthorized, ErrorCodeForbidden:
		return ErrorCategoryAuth, false

	// Rate limiting - retryable after backoff
	case ErrorCodeRateLimitExceeded:
		return ErrorCategoryRateLimit, true

	// Trading errors - not retryable (business logic rejections)
	case ErrorCodeFOKNotFilled, ErrorCodeIOCNotFilled,
		ErrorCodeIdempotencyConflict, ErrorCodeInsufficientMargin,
		ErrorCodeInvalidOrderSide, ErrorCodeInvalidTriggerPrice,
		ErrorCodeMarketClosed, ErrorCodeMarketNotFound,
		ErrorCodeMaxOrdersPerMarket,
		ErrorCodeMaxSubAccountsExceeded, ErrorCodeMaxTotalOrders, ErrorCodeNoLiquidity,
		ErrorCodeOICapExceeded, ErrorCodeOperationCancelFailed,
		ErrorCodeOrderNotFound,
		ErrorCodeOrderRejectedByEngine, ErrorCodePositionNotFound,
		ErrorCodePostOnlyWouldTrade, ErrorCodePriceOutOfBounds,
		ErrorCodeQuantityBelowFilled, ErrorCodeQuantityTooSmall,
		ErrorCodeReduceOnlyNoPosition, ErrorCodeReduceOnlySameSide,
		ErrorCodeReduceOnlyWouldIncrease, ErrorCodeSelfTradePrevented:
		return ErrorCategoryTrading, false

	// Wick insurance - retryable after protection period ends
	case ErrorCodeWickInsuranceActive:
		return ErrorCategoryTrading, true

	// Timeout - retryable
	case ErrorCodeOperationTimeout:
		return ErrorCategorySystem, true

	// System errors - some retryable
	case ErrorCodeInternalError, ErrorCodeDatabaseError, ErrorCodeCacheError, ErrorCodeRequestTimeout,
		ErrorCodeServiceUnavailable:
		return ErrorCategorySystem, true

	case ErrorCodeNotFound, ErrorCodeMethodNotAllowed,
		ErrorCodeMarketAlreadyExists, ErrorCodeAssetNotFound,
		ErrorCodeInvalidMarketConfig, ErrorCodeUnknownError:
		return ErrorCategorySystem, false

	default:
		return ErrorCategorySystem, false
	}
}
