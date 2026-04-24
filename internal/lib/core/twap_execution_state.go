package core

import (
	shopspring_decimal "github.com/shopspring/decimal"
)

// TWAPExecutionState is a portable snapshot of TWAP parent execution. The subaccount
// service stores it in open_order_twap; the NATS OpenOrderEvent carries it as a native
// JSON object. Trading embeds this type in its runtime TWAP model.
type TWAPExecutionState struct {
	// Identity
	OrderId      OrderId      `json:"order_id"`
	SubAccountID SubAccountId `json:"sub_account_id"`

	// Configuration (immutable after creation)
	ChunkIntervalMs int64                      `json:"chunk_interval_ms"`
	ChunkQuantity   shopspring_decimal.Decimal `json:"chunk_quantity"`
	ChunksTotal     int                        `json:"chunks_total"`
	Direction       Direction                  `json:"direction"`
	PriceLimit      shopspring_decimal.Decimal `json:"price_limit"`
	ReduceOnly      bool                       `json:"reduce_only"`
	Side            OrderSide                  `json:"side"`
	Source          string                     `json:"source"`
	StartedAtMs     int64                      `json:"started_at_ms"`
	Symbol          string                     `json:"symbol"`
	TotalQuantity   shopspring_decimal.Decimal `json:"total_quantity"`

	// Progress (mutable, authoritative for recovery)
	ChunksFilled      int                        `json:"chunks_filled"`
	ChunksSubmitted   int                        `json:"chunks_submitted"`
	FilledNotional    shopspring_decimal.Decimal `json:"filled_notional"`
	NextChunkAtMs     int64                      `json:"next_chunk_at_ms"`
	QuantityFilled    shopspring_decimal.Decimal `json:"quantity_filled"`
	QuantitySubmitted shopspring_decimal.Decimal `json:"quantity_submitted"`
	State             OrderState                 `json:"state"`
	TotalFees         shopspring_decimal.Decimal `json:"total_fees"`
}

// AveragePrice returns the volume-weighted average fill price. Returns zero
// when no quantity has been filled.
func (xs *TWAPExecutionState) AveragePrice(
	priceExponent int32,
) shopspring_decimal.Decimal {
	if xs.QuantityFilled.IsZero() {
		return shopspring_decimal.Zero
	}
	return xs.FilledNotional.Div(xs.QuantityFilled).RoundBank(priceExponent)
}

// TWAPExecutionStateHasPayload reports whether w carries a non-nil state payload.
func TWAPExecutionStateHasPayload(w *TWAPExecutionState) bool {
	return w != nil
}
