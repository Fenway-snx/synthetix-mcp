// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package utils

import (
	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
)

type InfoContext = snx_lib_api_handlers_types.InfoContext
type MaintenanceMarginTier = snx_lib_api_handlers_types.MaintenanceMarginTier
type MarketResponse = snx_lib_api_handlers_types.MarketResponse

type Price = snx_lib_api_types.Price
type Quantity = snx_lib_api_types.Quantity
type Symbol = snx_lib_api_types.Symbol
type Timestamp = snx_lib_api_types.Timestamp
type TradeId = snx_lib_api_types.TradeId
type Volume = snx_lib_api_types.Volume

const (
	Timestamp_Invalid = snx_lib_api_types.Timestamp_Invalid
	Timestamp_Never   = snx_lib_api_types.Timestamp_Never
	Timestamp_Zero    = snx_lib_api_types.Timestamp_Zero
)
