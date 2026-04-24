// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package auth

import (
	snx_lib_api_constants "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/constants"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

type Nonce = snx_lib_api_types.Nonce
type RequestAction = snx_lib_api_types.RequestAction
type SubAccountId = snx_lib_api_types.SubAccountId

const (
	API_WKS_action           = snx_lib_api_constants.API_WKS_action
	API_WKS_amount           = snx_lib_api_constants.API_WKS_amount
	API_WKS_destination      = snx_lib_api_constants.API_WKS_destination
	API_WKS_domain           = snx_lib_api_constants.API_WKS_domain
	API_WKS_expiresAfter     = snx_lib_api_constants.API_WKS_expiresAfter
	API_WKS_grouping         = snx_lib_api_constants.API_WKS_grouping
	API_WKS_leverage         = snx_lib_api_constants.API_WKS_leverage
	API_WKS_message          = snx_lib_api_constants.API_WKS_message
	API_WKS_name             = snx_lib_api_constants.API_WKS_name
	API_WKS_nonce            = snx_lib_api_constants.API_WKS_nonce
	API_WKS_orderId          = snx_lib_api_constants.API_WKS_orderId
	API_WKS_orderIds         = snx_lib_api_constants.API_WKS_orderIds
	API_WKS_orders           = snx_lib_api_constants.API_WKS_orders
	API_WKS_price            = snx_lib_api_constants.API_WKS_price
	API_WKS_primaryType      = snx_lib_api_constants.API_WKS_primaryType
	API_WKS_quantity         = snx_lib_api_constants.API_WKS_quantity
	API_WKS_sourceAsset      = snx_lib_api_constants.API_WKS_sourceAsset
	API_WKS_subAccountId     = snx_lib_api_constants.API_WKS_subAccountId
	API_WKS_symbol           = snx_lib_api_constants.API_WKS_symbol
	API_WKS_symbols          = snx_lib_api_constants.API_WKS_symbols
	API_WKS_targetUSDTAmount = snx_lib_api_constants.API_WKS_targetUSDTAmount
	API_WKS_time             = snx_lib_api_constants.API_WKS_time
	API_WKS_timeoutSeconds   = snx_lib_api_constants.API_WKS_timeoutSeconds
	API_WKS_triggerPrice     = snx_lib_api_constants.API_WKS_triggerPrice
	API_WKS_types            = snx_lib_api_constants.API_WKS_types
)

const (
	Price_None    = snx_lib_api_types.Price_None
	Quantity_None = snx_lib_api_types.Quantity_None
)

const (
	DelegationPermissionSession = snx_lib_core.DelegationPermissionSession
	DelegationPermissionTrading = snx_lib_core.DelegationPermissionTrading
)
