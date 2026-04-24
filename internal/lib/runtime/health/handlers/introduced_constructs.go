// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package handlers

import (
	snx_lib_net_http "github.com/Fenway-snx/synthetix-mcp/internal/lib/net/http"
)

const (
	HTTPStatusCode_200_OK                  = snx_lib_net_http.HTTPStatusCode_200_OK
	HTTPStatusCode_400_BadRequest          = snx_lib_net_http.HTTPStatusCode_400_BadRequest
	HTTPStatusCode_500_InternalServerError = snx_lib_net_http.HTTPStatusCode_500_InternalServerError
)
