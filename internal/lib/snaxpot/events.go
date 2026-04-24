package snaxpot

import "time"

// SnaxpotPurchaseEvent is the JSON payload that the relayer publishes when it
// observes an on-chain sUSD ticket purchase that is eligible for Snaxpot
// credits.
type SnaxpotPurchaseEvent struct {
	AmountSUSD    string    `json:"amount_susd"`
	PurchasedAt   time.Time `json:"purchased_at"`
	LogIndex      uint      `json:"log_index"`
	WalletAddress string    `json:"wallet_address"`
	TxHash        string    `json:"tx_hash"`
}
