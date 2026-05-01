package validation

import (
	"errors"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

var (
	errActionTypeMustBeTransferCollateral = errors.New("action type must be 'transferCollateral'")
)

type TransferCollateralActionPayload struct {
	Action string `mapstructure:"action"`
	To     string `mapstructure:"to"`
	Symbol Asset  `mapstructure:"symbol"` // TODO: SNX-6098: rename to `AssetName`
	Amount string `mapstructure:"amount"`
}

// Validated and parsed collateral transfer fields.
type ValidatedTransferCollateralAction struct {
	Amount shopspring_decimal.Decimal
	Symbol Asset // TODO: SNX-6098: rename to `AssetName`
	To     SubAccountId
}

// Decodes a raw map payload into a typed collateral transfer payload.
func DecodeTransferCollateralAction(params map[string]any) (*TransferCollateralActionPayload, error) {
	if params == nil {
		return nil, ErrActionPayloadRequired
	}

	var payload TransferCollateralActionPayload
	if err := mapstructure.Decode(params, &payload); err != nil {
		return nil, fmt.Errorf("invalid transferCollateral payload: %w", err)
	}

	return &payload, nil
}

// Validates and creates a collateral transfer action.
func NewValidatedTransferCollateralAction(payload *TransferCollateralActionPayload) (*ValidatedTransferCollateralAction, error) {
	if payload == nil {
		return nil, ErrActionPayloadRequired
	}

	// Validate action type
	if payload.Action != "transferCollateral" {
		return nil, errActionTypeMustBeTransferCollateral
	}

	// Parse to
	to, err := snx_lib_api_types.SubAccountIdToCoreSubaccountId(API_SubAccountId(payload.To))
	if err != nil {
		return nil, fmt.Errorf("to must be a valid positive integer: %w", err)
	}

	// TODO: follow this up with a remediation of the interconvertibility between Asset(Name) <-> Symbol
	symbol, err := snx_lib_core.AssetNameFromString(string(payload.Symbol))
	if err != nil {
		return nil, err
	}

	amount, err := shopspring_decimal.NewFromString(payload.Amount)
	if err != nil {
		return nil, ErrAmountMustBeValidDecimal
	}

	if !amount.IsPositive() {
		return nil, ErrAmountInvalid
	}

	return &ValidatedTransferCollateralAction{
		To:     SubAccountId(to),
		Symbol: Asset(string(symbol)), // TODO: SNX-6098: sort this out
		Amount: amount,
	}, nil
}
