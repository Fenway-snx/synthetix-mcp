package core

import (
	"time"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// NATSCandleMessage represents the candle message published to NATS
// This is the shared NATS message type used by both marketdata and websocket services
type NATSCandleMessage struct {
	ClosePrice  string                       `json:"close_price"`
	CloseTime   time.Time                    `json:"close_time"`
	HighPrice   string                       `json:"high_price"`
	LowPrice    string                       `json:"low_price"`
	OpenPrice   string                       `json:"open_price"`
	OpenTime    time.Time                    `json:"open_time"`
	QuoteVolume string                       `json:"quote_volume"`
	Symbol      string                       `json:"symbol"`
	Timeframe   snx_lib_utils_time.Timeframe `json:"timeframe"`
	TradeCount  int32                        `json:"trade_count"`
	Volume      string                       `json:"volume"`
}
