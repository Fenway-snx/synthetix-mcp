package events

import (
	"encoding/json"
	"fmt"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// Identifies whether an SLP mapping targets a market or a collateral.
// The zero value (SLPMappingType_Unknown) is intentionally invalid.
type SLPMappingType int64

const (
	SLPMappingType_Unknown SLPMappingType = iota
	SLPMappingType_Collateral
	SLPMappingType_Market
)

const (
	slpMappingTypeStringCollateral = "collateral"
	slpMappingTypeStringMarket     = "market"
	slpMappingTypeStringUnknown    = "unknown"
)

var (
	slpMappingTypeToString = map[SLPMappingType]string{
		SLPMappingType_Collateral: slpMappingTypeStringCollateral,
		SLPMappingType_Market:     slpMappingTypeStringMarket,
		SLPMappingType_Unknown:    slpMappingTypeStringUnknown,
	}

	slpMappingTypeFromString = map[string]SLPMappingType{
		slpMappingTypeStringCollateral: SLPMappingType_Collateral,
		slpMappingTypeStringMarket:     SLPMappingType_Market,
		slpMappingTypeStringUnknown:    SLPMappingType_Unknown,
	}
)

func (s SLPMappingType) String() string {
	if str, ok := slpMappingTypeToString[s]; ok {
		return str
	}
	return fmt.Sprintf("SLPMappingType(%d)", s)
}

func (s SLPMappingType) Valid() bool {
	return s == SLPMappingType_Market || s == SLPMappingType_Collateral
}

func (s SLPMappingType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SLPMappingType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	if v, ok := slpMappingTypeFromString[str]; ok {
		*s = v
		return nil
	}

	return fmt.Errorf("unknown SLPMappingType: %q", str)
}

// Requests reassignment of the SLP sub account for a market or collateral.
// Sent from the subaccount service to the trading service via NATS
// request-reply.
type AssignLiquidatorRequest struct {
	MappingType  SLPMappingType            `json:"mapping_type"`
	RequestID    string                    `json:"request_id"`
	SubAccountId snx_lib_core.SubAccountId `json:"sub_account_id"`
	Symbol       string                    `json:"symbol"`
}

// Carries the outcome of an SLP reassignment request.
type AssignLiquidatorResponse struct {
	Error        string                    `json:"error,omitempty"`
	MappingType  SLPMappingType            `json:"mapping_type"`
	RequestID    string                    `json:"request_id"`
	SubAccountId snx_lib_core.SubAccountId `json:"sub_account_id"`
	Success      bool                      `json:"success"`
	Symbol       string                    `json:"symbol"`
}
