package status_codes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ErrorCodeStringValues(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		// Validation
		{ErrorCodeInvalidFormat, "INVALID_FORMAT"},
		{ErrorCodeInvalidValue, "INVALID_VALUE"},
		{ErrorCodeMissingRequiredField, "MISSING_REQUIRED_FIELD"},
		{ErrorCodeValidationError, "VALIDATION_ERROR"},
		// Auth
		{ErrorCodeForbidden, "FORBIDDEN"},
		{ErrorCodeUnauthorized, "UNAUTHORIZED"},
		// Business Logic
		{ErrorCodeAssetNotFound, "ASSET_NOT_FOUND"},
		{ErrorCodeInvalidMarketConfig, "INVALID_MARKET_CONFIG"},
		{ErrorCodeMarketAlreadyExists, "MARKET_ALREADY_EXISTS"},
		// Trading
		{ErrorCodeFOKNotFilled, "FOK_NOT_FILLED"},
		{ErrorCodeIOCNotFilled, "IOC_NOT_FILLED"},
		{ErrorCodeIdempotencyConflict, "IDEMPOTENCY_CONFLICT"},
		{ErrorCodeInsufficientMargin, "INSUFFICIENT_MARGIN"},
		{ErrorCodeInvalidOrderSide, "INVALID_ORDER_SIDE"},
		{ErrorCodeInvalidTriggerPrice, "INVALID_TRIGGER_PRICE"},
		{ErrorCodeMarketClosed, "MARKET_CLOSED"},
		{ErrorCodeMarketNotFound, "MARKET_NOT_FOUND"},
		{ErrorCodeMaxOrdersPerMarket, "MAX_ORDERS_PER_MARKET"},
		{ErrorCodeMaxSubAccountsExceeded, "MAX_SUB_ACCOUNTS_EXCEEDED"},
		{ErrorCodeMaxTotalOrders, "MAX_TOTAL_ORDERS"},
		{ErrorCodeNoLiquidity, "NO_LIQUIDITY"},
		{ErrorCodeOICapExceeded, "OI_CAP_EXCEEDED"},
		{ErrorCodeOperationCancelFailed, "CANCEL_FAILED"},
		{ErrorCodeOperationTimeout, "OPERATION_TIMEOUT"},
		{ErrorCodeOrderNotFound, "ORDER_NOT_FOUND"},
		{ErrorCodeOrderRejectedByEngine, "ORDER_REJECTED_BY_ENGINE"},
		{ErrorCodePositionNotFound, "POSITION_NOT_FOUND"},
		{ErrorCodePostOnlyWouldTrade, "POST_ONLY_WOULD_TRADE"},
		{ErrorCodePriceOutOfBounds, "PRICE_OUT_OF_BOUNDS"},
		{ErrorCodeQuantityBelowFilled, "QUANTITY_BELOW_FILLED"},
		{ErrorCodeQuantityTooSmall, "QUANTITY_TOO_SMALL"},
		{ErrorCodeReduceOnlyNoPosition, "REDUCE_ONLY_NO_POSITION"},
		{ErrorCodeReduceOnlySameSide, "REDUCE_ONLY_SAME_SIDE"},
		{ErrorCodeReduceOnlyWouldIncrease, "REDUCE_ONLY_WOULD_INCREASE"},
		{ErrorCodeSelfTradePrevented, "SELF_TRADE_PREVENTED"},
		{ErrorCodeWickInsuranceActive, "WICK_INSURANCE_ACTIVE"},
		// Rate Limiting
		{ErrorCodeRateLimitExceeded, "RATE_LIMIT_EXCEEDED"},
		// System
		{ErrorCodeCacheError, "CACHE_ERROR"},
		{ErrorCodeDatabaseError, "DATABASE_ERROR"},
		{ErrorCodeInternalError, "INTERNAL_ERROR"},
		{ErrorCodeMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{ErrorCodeNotFound, "NOT_FOUND"},
		{ErrorCodeRequestTimeout, "REQUEST_TIMEOUT"},
		{ErrorCodeServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{ErrorCodeUnknownError, "UNKNOWN_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.code))
		})
	}
}

func Test_ErrorCategoryStringValues(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		want     string
	}{
		{ErrorCategoryRequest, "REQUEST"},
		{ErrorCategoryAuth, "AUTH"},
		{ErrorCategoryRateLimit, "RATE_LIMIT"},
		{ErrorCategoryTrading, "TRADING"},
		{ErrorCategorySystem, "SYSTEM"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.category))
		})
	}
}

func Test_CategorizeError(t *testing.T) {
	tests := []struct {
		code          ErrorCode
		wantCategory  ErrorCategory
		wantRetryable bool
	}{
		// Request validation errors - not retryable
		{ErrorCodeValidationError, ErrorCategoryRequest, false},
		{ErrorCodeMissingRequiredField, ErrorCategoryRequest, false},
		{ErrorCodeInvalidFormat, ErrorCategoryRequest, false},
		{ErrorCodeInvalidValue, ErrorCategoryRequest, false},

		// Auth errors - not retryable
		{ErrorCodeUnauthorized, ErrorCategoryAuth, false},
		{ErrorCodeForbidden, ErrorCategoryAuth, false},

		// Rate limiting - retryable after backoff
		{ErrorCodeRateLimitExceeded, ErrorCategoryRateLimit, true},

		// Trading errors - not retryable (business logic rejections)
		{ErrorCodeInsufficientMargin, ErrorCategoryTrading, false},
		{ErrorCodePostOnlyWouldTrade, ErrorCategoryTrading, false},
		{ErrorCodeReduceOnlyWouldIncrease, ErrorCategoryTrading, false},
		{ErrorCodeReduceOnlyNoPosition, ErrorCategoryTrading, false},
		{ErrorCodeReduceOnlySameSide, ErrorCategoryTrading, false},
		{ErrorCodeOrderNotFound, ErrorCategoryTrading, false},
		{ErrorCodeMarketClosed, ErrorCategoryTrading, false},
		{ErrorCodePriceOutOfBounds, ErrorCategoryTrading, false},
		{ErrorCodeQuantityTooSmall, ErrorCategoryTrading, false},
		{ErrorCodeQuantityBelowFilled, ErrorCategoryTrading, false},
		{ErrorCodeMaxOrdersPerMarket, ErrorCategoryTrading, false},
		{ErrorCodeMaxTotalOrders, ErrorCategoryTrading, false},
		{ErrorCodeMaxSubAccountsExceeded, ErrorCategoryTrading, false},
		{ErrorCodeFOKNotFilled, ErrorCategoryTrading, false},
		{ErrorCodeIOCNotFilled, ErrorCategoryTrading, false},
		{ErrorCodeIdempotencyConflict, ErrorCategoryTrading, false},
		{ErrorCodeInvalidTriggerPrice, ErrorCategoryTrading, false},
		{ErrorCodeInvalidOrderSide, ErrorCategoryTrading, false},
		{ErrorCodeMarketNotFound, ErrorCategoryTrading, false},
		{ErrorCodeNoLiquidity, ErrorCategoryTrading, false},
		{ErrorCodeOICapExceeded, ErrorCategoryTrading, false},
		{ErrorCodeOperationCancelFailed, ErrorCategoryTrading, false},
		{ErrorCodePositionNotFound, ErrorCategoryTrading, false},
		{ErrorCodeOrderRejectedByEngine, ErrorCategoryTrading, false},
		{ErrorCodeSelfTradePrevented, ErrorCategoryTrading, false},

		// Wick insurance - retryable (temporary protection)
		{ErrorCodeWickInsuranceActive, ErrorCategoryTrading, true},

		// Timeout - retryable
		{ErrorCodeOperationTimeout, ErrorCategorySystem, true},

		// System errors - retryable
		{ErrorCodeInternalError, ErrorCategorySystem, true},
		{ErrorCodeDatabaseError, ErrorCategorySystem, true},
		{ErrorCodeCacheError, ErrorCategorySystem, true},
		{ErrorCodeRequestTimeout, ErrorCategorySystem, true},
		{ErrorCodeServiceUnavailable, ErrorCategorySystem, true},

		// System errors - not retryable
		{ErrorCodeNotFound, ErrorCategorySystem, false},
		{ErrorCodeMethodNotAllowed, ErrorCategorySystem, false},
		{ErrorCodeMarketAlreadyExists, ErrorCategorySystem, false},
		{ErrorCodeAssetNotFound, ErrorCategorySystem, false},
		{ErrorCodeInvalidMarketConfig, ErrorCategorySystem, false},
		{ErrorCodeUnknownError, ErrorCategorySystem, false},

		// Unknown code defaults to SYSTEM, not retryable
		{ErrorCode("UNKNOWN_CODE"), ErrorCategorySystem, false},
		{ErrorCode(""), ErrorCategorySystem, false},
	}

	for _, tt := range tests {
		name := string(tt.code)
		if name == "" {
			name = "empty_code"
		}
		t.Run(name, func(t *testing.T) {
			category, retryable := CategorizeError(tt.code)
			assert.Equal(t, tt.wantCategory, category, "category mismatch for code %s", tt.code)
			assert.Equal(t, tt.wantRetryable, retryable, "retryable mismatch for code %s", tt.code)
		})
	}
}

func Test_CategorizeError_AllCodesHandled(t *testing.T) {
	allCodes := []ErrorCode{
		ErrorCodeValidationError, ErrorCodeMissingRequiredField,
		ErrorCodeInvalidFormat, ErrorCodeInvalidValue,
		ErrorCodeUnauthorized, ErrorCodeForbidden,
		ErrorCodeMarketAlreadyExists, ErrorCodeAssetNotFound, ErrorCodeInvalidMarketConfig,
		ErrorCodeInsufficientMargin, ErrorCodePostOnlyWouldTrade,
		ErrorCodeReduceOnlyWouldIncrease, ErrorCodeReduceOnlyNoPosition,
		ErrorCodeReduceOnlySameSide, ErrorCodeOrderNotFound,
		ErrorCodeMarketClosed, ErrorCodePriceOutOfBounds,
		ErrorCodeQuantityTooSmall, ErrorCodeQuantityBelowFilled,
		ErrorCodeMaxOrdersPerMarket,
		ErrorCodeMaxSubAccountsExceeded, ErrorCodeMaxTotalOrders,
		ErrorCodeIdempotencyConflict,
		ErrorCodeWickInsuranceActive, ErrorCodeInvalidTriggerPrice,
		ErrorCodeInvalidOrderSide, ErrorCodeMarketNotFound,
		ErrorCodeNoLiquidity, ErrorCodeOICapExceeded,
		ErrorCodeOperationCancelFailed, ErrorCodePositionNotFound,
		ErrorCodeOrderRejectedByEngine, ErrorCodeSelfTradePrevented,
		ErrorCodeFOKNotFilled, ErrorCodeIOCNotFilled,
		ErrorCodeOperationTimeout,
		ErrorCodeRateLimitExceeded,
		ErrorCodeInternalError, ErrorCodeDatabaseError, ErrorCodeCacheError, ErrorCodeRequestTimeout,
		ErrorCodeServiceUnavailable,
		ErrorCodeNotFound, ErrorCodeMethodNotAllowed, ErrorCodeUnknownError,
	}

	validCategories := map[ErrorCategory]bool{
		ErrorCategoryRequest:   true,
		ErrorCategoryAuth:      true,
		ErrorCategoryRateLimit: true,
		ErrorCategoryTrading:   true,
		ErrorCategorySystem:    true,
	}

	for _, code := range allCodes {
		category, _ := CategorizeError(code)
		assert.True(t, validCategories[category],
			"code %s returned invalid category %s", code, category,
		)
	}
}

func Test_IsKnownErrorCode(t *testing.T) {
	knownCodes := []ErrorCode{
		ErrorCodeValidationError, ErrorCodeMissingRequiredField,
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
		ErrorCodeServiceUnavailable, ErrorCodeUnknownError,
	}

	for _, code := range knownCodes {
		t.Run(string(code), func(t *testing.T) {
			assert.True(t, IsKnownErrorCode(code),
				"expected code %s to be recognized", code,
			)
		})
	}

	unknownCodes := []ErrorCode{
		ErrorCode(""),
		ErrorCode("UNKNOWN_CODE"),
		ErrorCode("internal.db.connection_lost"),
		ErrorCode("subaccount-service-internal"),
		ErrorCode("max_sub_accounts_exceeded"),
	}

	for _, code := range unknownCodes {
		name := string(code)
		if name == "" {
			name = "empty"
		}
		t.Run("unknown_"+name, func(t *testing.T) {
			assert.False(t, IsKnownErrorCode(code),
				"expected code %q to be unrecognized", code,
			)
		})
	}
}
