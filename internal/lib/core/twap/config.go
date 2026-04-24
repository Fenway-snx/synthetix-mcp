package twap

import shopspring_decimal "github.com/shopspring/decimal"

// Config holds validated TWAP order parameters. Runtime execution progress lives in
// lib/core.TWAPExecutionState (trading) and open_order_twap (subaccount persistence).
type Config struct {
	ChunkIntervalMs int64                      `json:"chunk_interval_ms"`
	ChunkQuantity   shopspring_decimal.Decimal `json:"chunk_quantity"`
	// ChunksFilled is runtime progress for TWAP parent projections (e.g. after recovery).
	ChunksFilled int `json:"chunks_filled,omitempty"`
	// ChunksSubmitted is how many slice windows have had a child order published.
	ChunksSubmitted int                        `json:"chunks_submitted,omitempty"`
	ChunksTotal     int                        `json:"chunks_total"`
	PriceLimit      shopspring_decimal.Decimal `json:"price_limit,omitempty"`
	TotalQuantity   shopspring_decimal.Decimal `json:"total_quantity"`
}
