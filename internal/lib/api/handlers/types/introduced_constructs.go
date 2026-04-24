// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package types

import (
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_net_http "github.com/Fenway-snx/synthetix-mcp/internal/lib/net/http"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

type Asset = snx_lib_api_types.Asset
type ClientRequestId = snx_lib_api_types.ClientRequestId
type RequestAction = snx_lib_api_types.RequestAction
type Symbol = snx_lib_api_types.Symbol
type TradeId = snx_lib_api_types.TradeId
type WalletAddress = snx_lib_api_types.WalletAddress

type RequestId = snx_lib_request.RequestId

type HTTPStatusCode = snx_lib_net_http.HTTPStatusCode
