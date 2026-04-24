// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package nats

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

type OffchainWithdrawalId = snx_lib_core.OffchainWithdrawalId
type OnchainWithdrawalId = snx_lib_core.OnchainWithdrawalId
type RequestId = snx_lib_core.RequestId
type SubAccountId = snx_lib_core.SubAccountId
type WalletAddress = snx_lib_core.WalletAddress
