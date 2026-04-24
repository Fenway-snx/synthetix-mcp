package core

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// ADLRankings is the payload element for NATS ADL ranking events.
// Each entry represents one position's ranking within a market+side bucket.
// The Trading service deserializes these from ADLRankingsUpdatedEvent messages
// published by the Subaccount service.
type ADLRankings struct {
	Bucket       int                        `json:"bucket"`
	PositionId   uint64                     `json:"position_id"`
	Quantity     shopspring_decimal.Decimal `json:"quantity"` // position quantity at ranking time (may be stale)
	Side         PositionSide               `json:"side"`
	SubAccountId SubAccountId               `json:"sub_account_id"`
	Symbol       string                     `json:"symbol"`
}

// ADLRankingsUpdatedEvent is published to NATS per market when rankings are recalculated.
// Each message contains separate long and short rankings and is published to a subject
// like "adl.rankings.updated.BTC-USDT", keeping payload small per market.
// The Trading service subscribes to these events for ADL execution decisions.
type ADLRankingsUpdatedEvent struct {
	LongRankings  []ADLRankings `json:"long_rankings"`
	ShortRankings []ADLRankings `json:"short_rankings"`
	Symbol        string        `json:"symbol"`
	Timestamp     time.Time     `json:"timestamp"`
}
