package auth

import (
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Typed event published when a delegation is added or activated, shared between
// event publishers and WebSocket consumers.
type DelegationAddedEvent struct {
	AddedAt         snx_lib_api_types.Timestamp  `json:"addedAt"` // unix milliseconds for API/WebSocket client state sync
	AddedBy         *snx_lib_core.WalletAddress  `json:"addedBy,omitempty"`
	DelegateAddress snx_lib_core.WalletAddress   `json:"delegateAddress"`
	ExpiresAt       *snx_lib_api_types.Timestamp `json:"expiresAt,omitempty"` // unix milliseconds
	Permissions     []string                     `json:"permissions"`
	SubAccountId    snx_lib_core.SubAccountId    `json:"subAccountId"`
}
