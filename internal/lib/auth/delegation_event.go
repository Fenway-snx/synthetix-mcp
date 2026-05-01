package auth

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Wire-shape for delegation revocation events.
type DelegationRevokedEvent struct {
	DelegateAddress snx_lib_core.WalletAddress `json:"delegateAddress"`
	RevokedAt       int64                      `json:"revokedAt"`
	SubAccountId    snx_lib_core.SubAccountId  `json:"subAccountId"`
}

// Receives deserialised delegation-revoked events.
type DelegationRevokedHandler func(event DelegationRevokedEvent)
