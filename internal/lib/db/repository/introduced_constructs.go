// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package repository

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

type OffchainWithdrawalId = snx_lib_core.OffchainWithdrawalId
type OnchainWithdrawalId = snx_lib_core.OnchainWithdrawalId
type PriceType = snx_lib_core.PriceType
type SubAccountId = snx_lib_core.SubAccountId
type VenueOrderId = snx_lib_core.VenueOrderId
type WithdrawalHistory = snx_lib_core.WithdrawalHistory
type WithdrawalHistory_View = snx_lib_core.WithdrawalHistory_View
type WithdrawalStatus = snx_lib_core.WithdrawalStatus

const (
	OrderTypeLimit = snx_lib_core.OrderTypeLimit
)
