// Introduces related package symbols to reduce local qualification noise.

// NOTE: this package should be moved to **lib/api/json**

package json

import (
	snx_lib_api_constants "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/constants"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
)

type ClientOrderId = snx_lib_api_types.ClientOrderId
type ClientRequestId = snx_lib_api_types.ClientRequestId
type Nonce = snx_lib_api_types.Nonce
type OrderId = snx_lib_api_types.OrderId
type Price = snx_lib_api_types.Price
type Quantity = snx_lib_api_types.Quantity
type SubAccountId = snx_lib_api_types.SubAccountId
type Symbol = snx_lib_api_types.Symbol
type VenueOrderId = snx_lib_api_types.VenueOrderId
type WalletAddress = snx_lib_api_types.WalletAddress

type ErrorCategory = snx_lib_status_codes.ErrorCategory
type ErrorCode = snx_lib_status_codes.ErrorCode

const (
	API_WKS_triggerSl = snx_lib_api_constants.API_WKS_triggerSl
	API_WKS_triggerTp = snx_lib_api_constants.API_WKS_triggerTp
)

const (
	Price_None    = snx_lib_api_types.Price_None
	Quantity_None = snx_lib_api_types.Quantity_None
)

const (
	ErrorCategoryAuth      = snx_lib_status_codes.ErrorCategoryAuth
	ErrorCategoryRateLimit = snx_lib_status_codes.ErrorCategoryRateLimit
	ErrorCategoryRequest   = snx_lib_status_codes.ErrorCategoryRequest
	ErrorCategorySystem    = snx_lib_status_codes.ErrorCategorySystem
	ErrorCategoryTrading   = snx_lib_status_codes.ErrorCategoryTrading

	ErrorCodeAssetNotFound           = snx_lib_status_codes.ErrorCodeAssetNotFound
	ErrorCodeCacheError              = snx_lib_status_codes.ErrorCodeCacheError
	ErrorCodeDatabaseError           = snx_lib_status_codes.ErrorCodeDatabaseError
	ErrorCodeFOKNotFilled            = snx_lib_status_codes.ErrorCodeFOKNotFilled
	ErrorCodeForbidden               = snx_lib_status_codes.ErrorCodeForbidden
	ErrorCodeIOCNotFilled            = snx_lib_status_codes.ErrorCodeIOCNotFilled
	ErrorCodeIdempotencyConflict     = snx_lib_status_codes.ErrorCodeIdempotencyConflict
	ErrorCodeInsufficientMargin      = snx_lib_status_codes.ErrorCodeInsufficientMargin
	ErrorCodeInternalError           = snx_lib_status_codes.ErrorCodeInternalError
	ErrorCodeInvalidFormat           = snx_lib_status_codes.ErrorCodeInvalidFormat
	ErrorCodeInvalidMarketConfig     = snx_lib_status_codes.ErrorCodeInvalidMarketConfig
	ErrorCodeInvalidOrderSide        = snx_lib_status_codes.ErrorCodeInvalidOrderSide
	ErrorCodeInvalidTriggerPrice     = snx_lib_status_codes.ErrorCodeInvalidTriggerPrice
	ErrorCodeInvalidValue            = snx_lib_status_codes.ErrorCodeInvalidValue
	ErrorCodeMarketAlreadyExists     = snx_lib_status_codes.ErrorCodeMarketAlreadyExists
	ErrorCodeMarketClosed            = snx_lib_status_codes.ErrorCodeMarketClosed
	ErrorCodeMarketNotFound          = snx_lib_status_codes.ErrorCodeMarketNotFound
	ErrorCodeMaxOrdersPerMarket      = snx_lib_status_codes.ErrorCodeMaxOrdersPerMarket
	ErrorCodeMaxSubAccountsExceeded  = snx_lib_status_codes.ErrorCodeMaxSubAccountsExceeded
	ErrorCodeMaxTotalOrders          = snx_lib_status_codes.ErrorCodeMaxTotalOrders
	ErrorCodeMethodNotAllowed        = snx_lib_status_codes.ErrorCodeMethodNotAllowed
	ErrorCodeMissingRequiredField    = snx_lib_status_codes.ErrorCodeMissingRequiredField
	ErrorCodeNoLiquidity             = snx_lib_status_codes.ErrorCodeNoLiquidity
	ErrorCodeNotFound                = snx_lib_status_codes.ErrorCodeNotFound
	ErrorCodeOICapExceeded           = snx_lib_status_codes.ErrorCodeOICapExceeded
	ErrorCodeOperationCancelFailed   = snx_lib_status_codes.ErrorCodeOperationCancelFailed
	ErrorCodeOperationTimeout        = snx_lib_status_codes.ErrorCodeOperationTimeout
	ErrorCodeOrderNotFound           = snx_lib_status_codes.ErrorCodeOrderNotFound
	ErrorCodeOrderRejectedByEngine   = snx_lib_status_codes.ErrorCodeOrderRejectedByEngine
	ErrorCodePositionNotFound        = snx_lib_status_codes.ErrorCodePositionNotFound
	ErrorCodePostOnlyWouldTrade      = snx_lib_status_codes.ErrorCodePostOnlyWouldTrade
	ErrorCodePriceOutOfBounds        = snx_lib_status_codes.ErrorCodePriceOutOfBounds
	ErrorCodeQuantityBelowFilled     = snx_lib_status_codes.ErrorCodeQuantityBelowFilled
	ErrorCodeQuantityTooSmall        = snx_lib_status_codes.ErrorCodeQuantityTooSmall
	ErrorCodeRateLimitExceeded       = snx_lib_status_codes.ErrorCodeRateLimitExceeded
	ErrorCodeReduceOnlyNoPosition    = snx_lib_status_codes.ErrorCodeReduceOnlyNoPosition
	ErrorCodeReduceOnlySameSide      = snx_lib_status_codes.ErrorCodeReduceOnlySameSide
	ErrorCodeReduceOnlyWouldIncrease = snx_lib_status_codes.ErrorCodeReduceOnlyWouldIncrease
	ErrorCodeRequestTimeout          = snx_lib_status_codes.ErrorCodeRequestTimeout
	ErrorCodeSelfTradePrevented      = snx_lib_status_codes.ErrorCodeSelfTradePrevented
	ErrorCodeServiceUnavailable      = snx_lib_status_codes.ErrorCodeServiceUnavailable
	ErrorCodeUnauthorized            = snx_lib_status_codes.ErrorCodeUnauthorized
	ErrorCodeUnknownError            = snx_lib_status_codes.ErrorCodeUnknownError
	ErrorCodeValidationError         = snx_lib_status_codes.ErrorCodeValidationError
	ErrorCodeWickInsuranceActive     = snx_lib_status_codes.ErrorCodeWickInsuranceActive
)

// Re-export CategorizeError so callers within this package and external
// consumers that already import lib/api/json continue to work.
var CategorizeError = snx_lib_status_codes.CategorizeError

// Re-export IsKnownErrorCode so API-boundary code can gate forwarding of
// server-supplied reason strings into client-facing error codes without
// taking a direct dependency on lib/core/status_codes.
var IsKnownErrorCode = snx_lib_status_codes.IsKnownErrorCode
