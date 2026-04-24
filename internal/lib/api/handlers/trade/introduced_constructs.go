// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package trade

import (
	snx_lib_api_constants "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/constants"
	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_net_http "github.com/Fenway-snx/synthetix-mcp/internal/lib/net/http"
)

type TradeContext = snx_lib_api_handlers_types.TradeContext
type HandlerParams = snx_lib_api_handlers_types.HandlerParams

type OrderDataResponse = snx_lib_api_json.OrderDataResponse
type OrderStatusResponse = snx_lib_api_json.OrderStatusResponse

type Asset = snx_lib_api_types.Asset
type ClientOrderId = snx_lib_api_types.ClientOrderId
type ClientRequestId = snx_lib_api_types.ClientRequestId
type OrderId = snx_lib_api_types.OrderId
type OrderType = snx_lib_api_types.OrderType
type Price = snx_lib_api_types.Price
type Quantity = snx_lib_api_types.Quantity
type Side = snx_lib_api_types.Side
type SubAccountId = snx_lib_api_types.SubAccountId
type Symbol = snx_lib_api_types.Symbol
type TimeInForce = snx_lib_api_types.TimeInForce
type Timestamp = snx_lib_api_types.Timestamp
type TradeId = snx_lib_api_types.TradeId
type TriggerPriceType = snx_lib_api_types.TriggerPriceType
type TxHash = snx_lib_api_types.TxHash
type VenueOrderId = snx_lib_api_types.VenueOrderId
type WalletAddress = snx_lib_api_types.WalletAddress

type PlaceOrdersActionPayload = snx_lib_api_validation.PlaceOrdersActionPayload
type ValidatedCancelAllOrdersAction = snx_lib_api_validation.ValidatedCancelAllOrdersAction
type ValidatedCancelOrdersByCloidAction = snx_lib_api_validation.ValidatedCancelOrdersByCloidAction
type ValidatedCancelOrdersAction = snx_lib_api_validation.ValidatedCancelOrdersAction
type ValidatedModifyOrderAction = snx_lib_api_validation.ValidatedModifyOrderAction
type ValidatedModifyOrderByCloidAction = snx_lib_api_validation.ValidatedModifyOrderByCloidAction
type ValidatedModifyOrderBatchAction = snx_lib_api_validation.ValidatedModifyOrderBatchAction
type ValidatedModifyOrderBatchItem = snx_lib_api_validation.ValidatedModifyOrderBatchItem
type ValidatedPlaceOrdersAction = snx_lib_api_validation.ValidatedPlaceOrdersAction
type ValidatedClearSnaxpotPreferenceAction = snx_lib_api_validation.ValidatedClearSnaxpotPreferenceAction
type ValidatedSaveSnaxpotTicketsAction = snx_lib_api_validation.ValidatedSaveSnaxpotTicketsAction
type ValidatedScheduleCancelAction = snx_lib_api_validation.ValidatedScheduleCancelAction
type ValidatedSetSnaxpotPreferenceAction = snx_lib_api_validation.ValidatedSetSnaxpotPreferenceAction
type ValidatedUpdateLeverageAction = snx_lib_api_validation.ValidatedUpdateLeverageAction
type ValidatedTransferCollateralAction = snx_lib_api_validation.ValidatedTransferCollateralAction
type ValidatedVoluntaryAutoExchangeAction = snx_lib_api_validation.ValidatedVoluntaryAutoExchangeAction
type ValidatedWithdrawCollateralAction = snx_lib_api_validation.ValidatedWithdrawCollateralAction

type HTTPStatusCode = snx_lib_net_http.HTTPStatusCode

const (
	API_WKS_amount         = snx_lib_api_constants.API_WKS_amount
	API_WKS_avgPrice       = snx_lib_api_constants.API_WKS_avgPrice
	API_WKS_clientOrderId  = snx_lib_api_constants.API_WKS_clientOrderId
	API_WKS_cumQty         = snx_lib_api_constants.API_WKS_cumQty
	API_WKS_destination    = snx_lib_api_constants.API_WKS_destination
	API_WKS_limit          = snx_lib_api_constants.API_WKS_limit
	API_WKS_offset         = snx_lib_api_constants.API_WKS_offset
	API_WKS_orderId        = snx_lib_api_constants.API_WKS_orderId
	API_WKS_price          = snx_lib_api_constants.API_WKS_price
	API_WKS_quantity       = snx_lib_api_constants.API_WKS_quantity
	API_WKS_requestID      = snx_lib_api_constants.API_WKS_requestID
	API_WKS_status         = snx_lib_api_constants.API_WKS_status
	API_WKS_symbol         = snx_lib_api_constants.API_WKS_symbol
	API_WKS_timestamp      = snx_lib_api_constants.API_WKS_timestamp
	API_WKS_time           = snx_lib_api_constants.API_WKS_time
	API_WKS_timeoutSeconds = snx_lib_api_constants.API_WKS_timeoutSeconds
	API_WKS_trades         = snx_lib_api_constants.API_WKS_trades
	API_WKS_triggerPrice   = snx_lib_api_constants.API_WKS_triggerPrice
	API_WKS_user           = snx_lib_api_constants.API_WKS_user
)

const (
	GroupingValues_na            = snx_lib_api_constants.GroupingValues_na
	GroupingValues_normalTpsl    = snx_lib_api_constants.GroupingValues_normalTpsl
	GroupingValues_positionsTpsl = snx_lib_api_constants.GroupingValues_positionsTpsl
	GroupingValues_twap          = snx_lib_api_constants.GroupingValues_twap
)

const (
	API_WKS_triggerSl = snx_lib_api_constants.API_WKS_triggerSl
	API_WKS_triggerTp = snx_lib_api_constants.API_WKS_triggerTp
)

const (
	ClientOrderId_Empty   = snx_lib_api_types.ClientOrderId_Empty
	Price_None            = snx_lib_api_types.Price_None
	Quantity_None         = snx_lib_api_types.Quantity_None
	Symbol_None           = snx_lib_api_types.Symbol_None
	TriggerPriceType_mark = snx_lib_api_types.TriggerPriceType_mark
	VenueOrderId_None     = snx_lib_api_types.VenueOrderId_None
)

const (
	HTTPStatusCode_200_OK                    = snx_lib_net_http.HTTPStatusCode_200_OK
	HTTPStatusCode_400_BadRequest            = snx_lib_net_http.HTTPStatusCode_400_BadRequest
	HTTPStatusCode_401_Unauthorized          = snx_lib_net_http.HTTPStatusCode_401_Unauthorized
	HTTPStatusCode_403_Forbidden             = snx_lib_net_http.HTTPStatusCode_403_Forbidden
	HTTPStatusCode_404_NotFound              = snx_lib_net_http.HTTPStatusCode_404_NotFound
	HTTPStatusCode_429_StatusTooManyRequests = snx_lib_net_http.HTTPStatusCode_429_StatusTooManyRequests
	HTTPStatusCode_500_InternalServerError   = snx_lib_net_http.HTTPStatusCode_500_InternalServerError
	HTTPStatusCode_501_StatusNotImplemented  = snx_lib_net_http.HTTPStatusCode_501_StatusNotImplemented
)

var (
	ErrSymbolNameEmpty  = snx_lib_api_constants.ErrSymbolNameEmpty
	ErrSymbolsNameEmpty = snx_lib_api_constants.ErrSymbolsNameEmpty
)

var (
	ErrorCodeInternalError   = snx_lib_api_json.ErrorCodeInternalError
	ErrorCodeValidationError = snx_lib_api_json.ErrorCodeValidationError
)
