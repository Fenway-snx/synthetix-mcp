// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package info

import (
	snx_lib_api_constants "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/constants"
	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_net_http "github.com/Fenway-snx/synthetix-mcp/internal/lib/net/http"
)

type CollateralConfigResponse = snx_lib_api_handlers_types.CollateralConfigResponse
type CollateralConfigTierResponse = snx_lib_api_handlers_types.CollateralConfigTierResponse
type ContextCommon = snx_lib_api_handlers_types.ContextCommon
type HandlerParams = snx_lib_api_handlers_types.HandlerParams
type InfoContext = snx_lib_api_handlers_types.InfoContext
type MarketResponse = snx_lib_api_handlers_types.MarketResponse

type Asset = snx_lib_api_types.Asset
type ClientRequestId = snx_lib_api_types.ClientRequestId
type FundingRate = snx_lib_api_types.FundingRate
type InterestRate = snx_lib_api_types.InterestRate
type Price = snx_lib_api_types.Price
type Quantity = snx_lib_api_types.Quantity
type SubAccountId = snx_lib_api_types.SubAccountId
type Symbol = snx_lib_api_types.Symbol
type Timestamp = snx_lib_api_types.Timestamp
type TradeId = snx_lib_api_types.TradeId
type Volume = snx_lib_api_types.Volume
type WalletAddress = snx_lib_api_types.WalletAddress

type HTTPStatusCode = snx_lib_net_http.HTTPStatusCode

const (
	API_WKS_limit  = snx_lib_api_constants.API_WKS_limit
	API_WKS_symbol = snx_lib_api_constants.API_WKS_symbol
)

const (
	ErrorCodeValidationError = snx_lib_api_json.ErrorCodeValidationError
)

const (
	Price_None    = snx_lib_api_types.Price_None
	Quantity_None = snx_lib_api_types.Quantity_None
)

const (
	HTTPStatusCode_200_OK                  = snx_lib_net_http.HTTPStatusCode_200_OK
	HTTPStatusCode_400_BadRequest          = snx_lib_net_http.HTTPStatusCode_400_BadRequest
	HTTPStatusCode_404_NotFound            = snx_lib_net_http.HTTPStatusCode_404_NotFound
	HTTPStatusCode_500_InternalServerError = snx_lib_net_http.HTTPStatusCode_500_InternalServerError
)
