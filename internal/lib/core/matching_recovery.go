package core

import (
	"time"
)

// Service ID constants
const (
	MatchingServiceID   = "matching-service"
	MarketdataServiceID = "marketdata-service"
)

// MatchingRecoveryState represents the state of matching service recovery
type MatchingRecoveryState int8

// Recovery state constants (matching the enum values from matching service)
const (
	MatchingRecoveryStateNotStarted MatchingRecoveryState = iota // 0
	MatchingRecoveryStateInProgress                              // 1
	MatchingRecoveryStateCompleted                               // 2
	MatchingRecoveryStateFailed                                  // 3
)

// MatchingRecoveryRequest represents a request for orderbook recovery from matching service
type MatchingRecoveryRequest struct {
	RequestID    string           `json:"request_id"`
	Symbol       string           `json:"symbol"`
	FromSequence SnapshotSequence `json:"from_sequence"`
	RequesterID  string           `json:"requester_id"`
	Timestamp    time.Time        `json:"timestamp"`
}

// MatchingRecoveryResponse represents the response to a recovery request from matching service
type MatchingRecoveryResponse struct {
	RequestID      string                     `json:"request_id"`
	Symbol         string                     `json:"symbol"`
	SequenceNumber SnapshotSequence           `json:"sequence_number"`
	Snapshot       *MatchingOrderbookSnapshot `json:"snapshot"`
	Status         string                     `json:"status"` // "success", "error", "not_ready"
	Error          string                     `json:"error,omitempty"`
	Timestamp      time.Time                  `json:"timestamp"`
}

// MatchingOrderbookSnapshot represents a point-in-time orderbook state from matching service
type MatchingOrderbookSnapshot struct {
	Bids            []MatchingPriceLevel `json:"bids"`
	Asks            []MatchingPriceLevel `json:"asks"`
	LastTradedPrice uint64               `json:"last_traded_price,omitempty"` // Last trade price (0 if no trades yet)
	LastTradedTime  time.Time            `json:"last_traded_time,omitempty"`  // Last trade timestamp
}

// MatchingPriceLevel represents a price level in the orderbook snapshot
type MatchingPriceLevel struct {
	Price    uint64                `json:"price"`
	Quantity uint64                `json:"quantity"`
	Orders   []MatchingOrderDetail `json:"orders"`
}

// MatchingOrderDetail represents order information at a price level in the snapshot
type MatchingOrderDetail struct {
	VenueOrderId VenueOrderId `json:"order_id"`
	SubAccountId SubAccountId `json:"sub_account_id"`
	Quantity     uint64       `json:"quantity"`
	Timestamp    time.Time    `json:"timestamp"`
}

// MatchingRecoveryStatus represents the overall recovery status of matching service
type MatchingRecoveryStatus struct {
	ServiceID     string                `json:"service_id"`
	RecoveryState MatchingRecoveryState `json:"recovery_state"` // 0=not_started, 1=in_progress, 2=completed, 3=failed
	StartTime     time.Time             `json:"start_time"`
	EndTime       time.Time             `json:"end_time,omitempty"`
	Duration      time.Duration         `json:"duration,omitempty"`
}
