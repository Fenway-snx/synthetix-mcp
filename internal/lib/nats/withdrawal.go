package nats

import (
	"time"
)

// TODO: we always pass one event with one amount and one token address.
// We need to update this event so we only pass on struct with the existing fields
// plus the amount, receiver, and token.
// and here token is meant to be token address
// WithdrawalRequestEvent represents a withdrawal request published to NATS
type WithdrawalRequestEvent struct {
	Entries              []WithdrawalRequestEventEntry `json:"entries"`
	OffchainWithdrawalId OffchainWithdrawalId          `json:"offchain_withdrawal_id"`
	RequestedAt          time.Time                     `json:"requested_at"`
	SubAccountId         SubAccountId                  `json:"sub_account_id"`
	UserWalletAddress    WalletAddress                 `json:"user_wallet_address"`
}

// WithdrawalRequestEventEntry represents a single withdrawal entry
type WithdrawalRequestEventEntry struct {
	Amounts               []string      `json:"amounts"`
	AssetName             string        `json:"asset_name,omitempty"` // Asset symbol (e.g., "USDC") for WebSocket notifications
	ReceiverWalletAddress WalletAddress `json:"receiver_wallet_address"`
	Tokens                []string      `json:"tokens"`
}

// WithdrawalExpiredStatusRequest is sent by subaccount service to relayer
// to check if withdrawals have expired on-chain (supports batch queries)
type WithdrawalExpiredStatusRequest struct {
	OnchainWithdrawalIds []OnchainWithdrawalId `json:"onchain_withdrawal_ids"` // On-chain withdrawal IDs to check
	RequestId            RequestId             `json:"request_id"`             // Correlation ID
}

// WithdrawalExpiredStatusResult represents the status of a single withdrawal
type WithdrawalExpiredStatusResult struct {
	Error               string              `json:"error"`      // Error message if query failed for this specific withdrawal
	Exists              bool                `json:"exists"`     // False if withdrawal not found on-chain
	IsExpired           bool                `json:"is_expired"` // True if withdrawal is expired/denied/cancelled on-chain
	OnchainWithdrawalId OnchainWithdrawalId `json:"onchain_withdrawal_id"`
	Status              uint8               `json:"status"`        // Contract status enum (for logging/debugging)
	StatusString        string              `json:"status_string"` // Human-readable status ("disbursed", "expired", etc.)
}

// WithdrawalExpiredStatusResponse returns status for multiple withdrawals
type WithdrawalExpiredStatusResponse struct {
	Error     string                          `json:"error,omitempty"` // Overall error message
	RequestId RequestId                       `json:"request_id"`
	Results   []WithdrawalExpiredStatusResult `json:"results"` // Status for each withdrawal ID
}

// WithdrawalStatusNotification is published by the relayer for WebSocket notifications.
// This is fire-and-forget via regular NATS (not JetStream) for real-time UI updates.
type WithdrawalStatusNotification struct {
	Amount               string               `json:"amount"`
	Asset                string               `json:"asset"`
	DestinationAddress   string               `json:"destination_address,omitempty"`
	FailureReason        string               `json:"failure_reason,omitempty"`
	OffchainWithdrawalId OffchainWithdrawalId `json:"offchain_withdrawal_id"`
	OnchainWithdrawalId  OnchainWithdrawalId  `json:"onchain_withdrawal_id,omitempty"`
	Status               string               `json:"status"`
	SubAccountId         SubAccountId         `json:"sub_account_id"`
	Timestamp            time.Time            `json:"timestamp"`
	TxHash               string               `json:"tx_hash,omitempty"`
}
