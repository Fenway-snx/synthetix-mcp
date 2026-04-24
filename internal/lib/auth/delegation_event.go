package auth

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Wire-shape for delegation revocation events. Lives in the NATS-free
// part of lib/auth so importers that only need the struct (e.g. for
// JSON unmarshalling) don't pull in lib/db/nats.
type DelegationRevokedEvent struct {
	DelegateAddress snx_lib_core.WalletAddress `json:"delegateAddress"`
	RevokedAt       int64                      `json:"revokedAt"`
	SubAccountId    snx_lib_core.SubAccountId  `json:"subAccountId"`
}

// Receives deserialised delegation-revoked events.
type DelegationRevokedHandler func(event DelegationRevokedEvent)
