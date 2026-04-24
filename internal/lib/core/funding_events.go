package core

import (
	"encoding/json"

	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
)

// FundingSettingsEvent represents a funding settings-related event
type FundingSettingsEvent struct {
	EventType string         `json:"event_type"`
	EventID   string         `json:"event_id"`
	Timestamp string         `json:"timestamp"`
	Version   string         `json:"version"`
	Data      map[string]any `json:"data"`
	Metadata  EventMetadata  `json:"metadata"`
}

// Event types for funding settings
const (
	EventTypeFundingSettingsUpdated = "updated"
	EventTypeFundingSettingsCreated = "created"
)

// ParseFundingSettingsEvent parses a funding settings event from JSON bytes
func ParseFundingSettingsEvent(data []byte) (*FundingSettingsEvent, error) {
	var event FundingSettingsEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// GetFundingSettingsFromEvent extracts funding settings data from an event
func GetFundingSettingsFromEvent(e *FundingSettingsEvent) (*postgrestypes.FundingSettings, error) {
	if e.Data == nil {
		return nil, nil
	}

	// Extract funding settings data from the event
	settingsData, ok := e.Data["settings"]
	if !ok {
		return nil, nil
	}

	// Convert to JSON and back to ensure proper parsing
	settingsBytes, err := json.Marshal(settingsData)
	if err != nil {
		return nil, err
	}

	var settings postgrestypes.FundingSettings
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}
