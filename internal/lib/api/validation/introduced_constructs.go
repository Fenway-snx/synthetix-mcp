// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package validation

import (
	snx_lib_api_constants "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/constants"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

type GroupingValues = snx_lib_api_constants.GroupingValues

type API_SubAccountId = snx_lib_api_types.SubAccountId
type Asset = snx_lib_api_types.Asset
type ClientOrderId = snx_lib_api_types.ClientOrderId
type Price = snx_lib_api_types.Price
type Quantity = snx_lib_api_types.Quantity
type RequestAction = snx_lib_api_types.RequestAction
type Symbol = snx_lib_api_types.Symbol
type Timestamp = snx_lib_api_types.Timestamp
type TradeId = snx_lib_api_types.TradeId
type VenueOrderId = snx_lib_api_types.VenueOrderId
type WalletAddress = snx_lib_api_types.WalletAddress

type SubAccountId = snx_lib_core.SubAccountId

const (
	API_WKS_clientOrderId = snx_lib_api_constants.API_WKS_clientOrderId
	API_WKS_orderId       = snx_lib_api_constants.API_WKS_orderId
	API_WKS_side          = snx_lib_api_constants.API_WKS_side
	API_WKS_triggerSl     = snx_lib_api_constants.API_WKS_triggerSl
	API_WKS_triggerTp     = snx_lib_api_constants.API_WKS_triggerTp
)

const (
	GroupingValues_na            = snx_lib_api_constants.GroupingValues_na
	GroupingValues_normalTpsl    = snx_lib_api_constants.GroupingValues_normalTpsl
	GroupingValues_positionsTpsl = snx_lib_api_constants.GroupingValues_positionsTpsl
	GroupingValues_twap          = snx_lib_api_constants.GroupingValues_twap
)

const (
	AssetName_None      = snx_lib_api_types.AssetName_None
	ClientOrderId_Empty = snx_lib_api_types.ClientOrderId_Empty
	Price_None          = snx_lib_api_types.Price_None
	Quantity_None       = snx_lib_api_types.Quantity_None
	Symbol_None         = snx_lib_api_types.Symbol_None
	VenueOrderId_None   = snx_lib_api_types.VenueOrderId_None
)

var (
	ErrActionPayloadRequired     = snx_lib_core.ErrActionPayloadRequired
	ErrAmountInvalid             = snx_lib_core.ErrAmountInvalid
	ErrAmountMustBeValidDecimal  = snx_lib_core.ErrAmountMustBeValidDecimal
	ErrDestinationInvalidAddress = snx_lib_core.ErrDestinationInvalidAddress
	ErrDestinationRequired       = snx_lib_core.ErrDestinationRequired
	ErrInvalidGrouping           = snx_lib_core.ErrInvalidGrouping
	ErrInvalidModifyOrderPayload = snx_lib_core.ErrInvalidModifyOrderPayload
	ErrLeverageMustBePositive    = snx_lib_core.ErrLeverageMustBePositive
	ErrOrderIdMustBePositive     = snx_lib_core.ErrOrderIdMustBePositive
	ErrOrderIdsMustBeNonempty    = snx_lib_core.ErrOrderIdsMustBeNonempty
	ErrOrdersArrayEmpty          = snx_lib_core.ErrOrdersArrayEmpty
	ErrPriceMustBePositive       = snx_lib_core.ErrPriceMustBePositive
	ErrPriceMustBeValidDecimal   = snx_lib_core.ErrPriceMustBeValidDecimal
	ErrSymbolRequired            = snx_lib_core.ErrSymbolRequired
	ErrSymbolsMustBeNonempty     = snx_lib_core.ErrSymbolsMustBeNonempty
)
