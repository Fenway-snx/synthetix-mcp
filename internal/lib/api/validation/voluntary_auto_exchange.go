package validation

import (
	"errors"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

var (
	errActionTypeMustBeVoluntaryAutoExchange = errors.New("action type must be 'voluntaryAutoExchange'")
	errSourceAssetRequired                   = errors.New("sourceAsset is required")
	errSourceAssetCannotBeUSDT               = errors.New("sourceAsset cannot be USDT")
	errTargetUSDTAmountRequired              = errors.New("targetUSDTAmount is required")
	errTargetUSDTAmountInvalid               = errors.New("targetUSDTAmount must be 'all' or a valid positive decimal")
)

type VoluntaryAutoExchangeActionPayload struct {
	Action           string `mapstructure:"action"`
	SourceAsset      string `mapstructure:"sourceAsset"`
	TargetUSDTAmount string `mapstructure:"targetUSDTAmount"`
}

// ValidatedVoluntaryAutoExchangeAction contains validated voluntary auto-exchange fields.
type ValidatedVoluntaryAutoExchangeAction struct {
	Payload          *VoluntaryAutoExchangeActionPayload
	SourceAsset      string
	TargetUSDTAmount string // "all" or valid positive decimal string
}

// DecodeVoluntaryAutoExchangeAction decodes a raw map payload into a typed voluntary auto-exchange payload.
func DecodeVoluntaryAutoExchangeAction(params map[string]any) (*VoluntaryAutoExchangeActionPayload, error) {
	if params == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload VoluntaryAutoExchangeActionPayload
	if err := mapstructure.Decode(params, &payload); err != nil {
		return nil, fmt.Errorf("invalid voluntaryAutoExchange payload: %w", err)
	}

	return &payload, nil
}

// NewValidatedVoluntaryAutoExchangeAction validates and creates a validated voluntary auto-exchange action.
func NewValidatedVoluntaryAutoExchangeAction(payload *VoluntaryAutoExchangeActionPayload) (*ValidatedVoluntaryAutoExchangeAction, error) {
	if payload == nil {
		return nil, ErrActionPayloadRequired
	}

	if payload.Action != "voluntaryAutoExchange" {
		return nil, errActionTypeMustBeVoluntaryAutoExchange
	}

	if payload.SourceAsset == "" {
		return nil, errSourceAssetRequired
	}

	if payload.SourceAsset == snx_lib_core.NominatedCollateral {
		return nil, errSourceAssetCannotBeUSDT
	}

	if payload.TargetUSDTAmount == "" {
		return nil, errTargetUSDTAmountRequired
	}

	// Accept "all" or a valid positive decimal
	if payload.TargetUSDTAmount != "all" {
		amount, err := shopspring_decimal.NewFromString(payload.TargetUSDTAmount)
		if err != nil {
			return nil, errTargetUSDTAmountInvalid
		}
		if !amount.IsPositive() {
			return nil, errTargetUSDTAmountInvalid
		}
	}

	return &ValidatedVoluntaryAutoExchangeAction{
		Payload:          payload,
		SourceAsset:      payload.SourceAsset,
		TargetUSDTAmount: payload.TargetUSDTAmount,
	}, nil
}
