package guardrails

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	PresetReadOnly = "read_only"
	PresetStandard = "standard"
)

type Config struct {
	AllowedOrderTypes   []string `json:"allowedOrderTypes,omitempty"`
	AllowedSymbols      []string `json:"allowedSymbols,omitempty"`
	MaxOrderQuantity    string   `json:"maxOrderQuantity,omitempty"`
	MaxPositionQuantity string   `json:"maxPositionQuantity,omitempty"`
	Preset              string   `json:"preset,omitempty"`
}

type Resolved struct {
	AllowedOrderTypes      []string
	AllowedSymbols         []string
	EffectivePreset        string
	MaxOrderQuantity       decimal.Decimal
	MaxPositionQuantity    decimal.Decimal
	RequestedPreset        string
	hasMaxOrderQuantity    bool
	hasMaxPositionQuantity bool
	writeEnabled           bool
}

func Resolve(cfg *Config) (*Resolved, error) {
	requestedPreset := normalizePreset("")
	if cfg != nil {
		requestedPreset = normalizePreset(cfg.Preset)
	}

	switch requestedPreset {
	case PresetStandard:
		return resolveStandard(cfg)
	case PresetReadOnly:
		fallthrough
	default:
		return &Resolved{
			EffectivePreset: PresetReadOnly,
			RequestedPreset: requestedPreset,
		}, nil
	}
}

func (r *Resolved) HasMaxOrderQuantity() bool {
	return r != nil && r.hasMaxOrderQuantity
}

func (r *Resolved) HasMaxPositionQuantity() bool {
	return r != nil && r.hasMaxPositionQuantity
}

func (r *Resolved) IsOrderTypeAllowed(orderType string) bool {
	if r == nil || len(r.AllowedOrderTypes) == 0 {
		return false
	}
	return slices.Contains(r.AllowedOrderTypes, normalizeOrderType(orderType))
}

func (r *Resolved) IsReadOnly() bool {
	return r == nil || !r.writeEnabled
}

func (r *Resolved) IsSymbolAllowed(symbol string) bool {
	if r == nil || len(r.AllowedSymbols) == 0 {
		return false
	}
	return slices.Contains(r.AllowedSymbols, normalizeSymbol(symbol))
}

func (r *Resolved) WriteEnabled() bool {
	return r != nil && r.writeEnabled
}

func normalizeOrderType(orderType string) string {
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case "STOP_LIMIT":
		return "STOP"
	case "TAKE_PROFIT_LIMIT":
		return "TAKE_PROFIT"
	default:
		return strings.ToUpper(strings.TrimSpace(orderType))
	}
}

func normalizePreset(preset string) string {
	preset = strings.TrimSpace(strings.ToLower(preset))
	if preset == "" {
		return PresetReadOnly
	}
	return preset
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

func normalizeStrings(values []string, normalize func(string) string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalize(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	return out
}

func parsePositiveDecimal(raw string, fieldName string) (decimal.Decimal, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, false, nil
	}

	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, false, fmt.Errorf("%s must be a valid decimal", fieldName)
	}
	if !value.GreaterThan(decimal.Zero) {
		return decimal.Zero, false, fmt.Errorf("%s must be greater than zero", fieldName)
	}

	return value, true, nil
}

func resolveStandard(cfg *Config) (*Resolved, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	allowedSymbols := normalizeStrings(cfg.AllowedSymbols, normalizeSymbol)
	if len(allowedSymbols) == 0 {
		return nil, fmt.Errorf("standard preset requires at least one allowed symbol")
	}

	allowedOrderTypes := normalizeStrings(cfg.AllowedOrderTypes, normalizeOrderType)
	if len(allowedOrderTypes) == 0 {
		allowedOrderTypes = []string{"LIMIT", "MARKET"}
	}
	for _, orderType := range allowedOrderTypes {
		switch orderType {
		case "LIMIT", "MARKET", "STOP", "STOP_MARKET", "TAKE_PROFIT", "TAKE_PROFIT_MARKET":
		default:
			return nil, fmt.Errorf("unsupported allowed order type %q", orderType)
		}
	}

	maxOrderQuantity, hasMaxOrderQuantity, err := parsePositiveDecimal(cfg.MaxOrderQuantity, "maxOrderQuantity")
	if err != nil {
		return nil, err
	}
	if !hasMaxOrderQuantity {
		return nil, fmt.Errorf("standard preset requires maxOrderQuantity")
	}

	maxPositionQuantity, hasMaxPositionQuantity, err := parsePositiveDecimal(cfg.MaxPositionQuantity, "maxPositionQuantity")
	if err != nil {
		return nil, err
	}
	if !hasMaxPositionQuantity {
		return nil, fmt.Errorf("standard preset requires maxPositionQuantity")
	}

	return &Resolved{
		AllowedOrderTypes:      allowedOrderTypes,
		AllowedSymbols:         allowedSymbols,
		EffectivePreset:        PresetStandard,
		MaxOrderQuantity:       maxOrderQuantity,
		MaxPositionQuantity:    maxPositionQuantity,
		RequestedPreset:        normalizePreset(cfg.Preset),
		hasMaxOrderQuantity:    true,
		hasMaxPositionQuantity: true,
		writeEnabled:           true,
	}, nil
}
