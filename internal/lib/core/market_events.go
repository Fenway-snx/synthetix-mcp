package core

import (
	"encoding/json"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// Represents a market-related event from the market service, consumed by:
// - Pricing Service;
// - Trading Service;
type MarketEvent struct {
	EventType MarketEventType       `json:"event_type"`
	EventID   string                `json:"event_id"`
	Timestamp string                `json:"timestamp"`
	Version   string                `json:"version"`
	Market    *postgrestypes.Market `json:"data"` // TODO: `json:"market_configuration"`
	Metadata  EventMetadata         `json:"metadata"`
}

// EventMetadata contains metadata about the event
type EventMetadata struct {
	Source    string `json:"source"`
	RequestID string `json:"request_id,omitempty"`
}

type MarketEventType string

// Event types
const (
	MarketEventType_Created MarketEventType = "created"
	MarketEventType_Updated MarketEventType = "updated"
	MarketEventType_Deleted MarketEventType = "deleted"
)

// ParseMarketEvent parses a market event from JSON bytes
func ParseMarketEvent(data []byte) (*MarketEvent, error) {
	var event MarketEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// Extracts market data from an event.
func GetMarketFromEvent(e *MarketEvent) (*postgrestypes.Market, error) {

	return e.Market, nil
}
