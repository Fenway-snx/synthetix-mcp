package twap

import shopspring_decimal "github.com/shopspring/decimal"

// Validated TWAP order parameters.
type Config struct {
	ChunkIntervalMs int64                      `json:"chunk_interval_ms"`
	ChunkQuantity   shopspring_decimal.Decimal `json:"chunk_quantity"`
	// Runtime progress for TWAP parent projections.
	ChunksFilled int `json:"chunks_filled,omitempty"`
	// Count of slice windows with a published child order.
	ChunksSubmitted int                        `json:"chunks_submitted,omitempty"`
	ChunksTotal     int                        `json:"chunks_total"`
	PriceLimit      shopspring_decimal.Decimal `json:"price_limit,omitempty"`
	TotalQuantity   shopspring_decimal.Decimal `json:"total_quantity"`
}
