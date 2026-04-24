package events

import (
	"time"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Is the NATS event published by marketdata and consumed
// by the websocket service. It carries a point-in-time orderbook snapshot for a
// single symbol.
type OrderbookDepthMessage struct {
	Symbol         string                        `json:"symbol"`
	SequenceNumber snx_lib_core.SnapshotSequence `json:"sequenceNumber"`
	Met            time.Time                     `json:"met"`
	Bids           []OrderbookDepthPriceLevel    `json:"bids"`
	Asks           []OrderbookDepthPriceLevel    `json:"asks"`
}

// OrderbookDepthPriceLevel represents a single price level in the orderbook.
type OrderbookDepthPriceLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}
