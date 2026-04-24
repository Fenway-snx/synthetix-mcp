// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package whitelist

import (
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
)

type WalletAddress = snx_lib_api_types.WalletAddress
