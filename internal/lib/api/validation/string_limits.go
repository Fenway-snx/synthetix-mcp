package validation

import (
	"fmt"
	"strings"

	snx_lib_api_validation_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation/utils"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

const (
	MaxDecimalStringLength = 128
	MaxEnumFieldLength     = 32
	MaxEthAddressLength    = 42
	MaxOrdersPerBatch      = 20
	MaxRequestBodyBytes    = 20 * 1024
	MaxSymbolLength        = 20
)

// Static label or deferred label builder for validation errors.
type deferredValidationLabel interface {
	string | func() string
}

func deferredValidationLabelString[T deferredValidationLabel](context T) string {
	switch fn := any(context).(type) {
	case string:
		return fn
	case func() string:
		return fn()
	default:
		panic("deferredValidationLabelString: unexpected type")
	}
}

// Rejects overlong strings before deeper parsing.
// Deferred labels avoid allocation on the valid path.
func ValidateStringMaxLength[V ~string, C deferredValidationLabel](value V, maxLength int, context C) error {
	if maxLength > 0 && len(value) > maxLength {
		name := deferredValidationLabelString(context)

		return fmt.Errorf("%s exceeds maximum length of %d characters", name, maxLength)
	}

	return nil
}

// Validates and normalizes market symbols consistently across entrypoints.
func ValidateAndNormalizeSymbol(symbol Symbol) (Symbol, error) {
	if err := ValidateStringMaxLength(symbol, MaxSymbolLength, "symbol"); err != nil {
		return Symbol_None, err
	}

	normalisedName, err := snx_lib_core.ValidateStringForMarketName(string(symbol))
	if err != nil {
		if strings.TrimSpace(string(symbol)) == "" {
			return Symbol_None, ErrSymbolRequired
		}
		return Symbol_None, snx_lib_core.ErrInvalidSymbol
	}

	return Symbol(normalisedName), nil
}

// Validates and normalizes collateral/asset symbols (e.g. "USDT", "ETH").
func ValidateAndNormalizeCollateralSymbol(assetName Asset) (Asset, error) {
	if err := ValidateStringMaxLength(assetName, MaxSymbolLength, "symbol"); err != nil {
		return AssetName_None, err
	}

	s, err := snx_lib_core.ValidateStringForAssetName(string(assetName))
	if err != nil {
		if strings.TrimSpace(string(assetName)) == "" {
			return AssetName_None, ErrSymbolRequired
		}
		return AssetName_None, snx_lib_core.ErrInvalidSymbol
	}

	return Asset(s), nil
}

// Rejects symbols that only become valid after trimming or uppercasing.
func ValidateCanonicalSymbol[T deferredValidationLabel](symbol Symbol, context T) error {
	normalizedSymbol, err := ValidateAndNormalizeSymbol(symbol)
	if err != nil {
		label := deferredValidationLabelString(context)
		if label == "symbol" {
			return err
		}

		return fmt.Errorf("%s: %w", label, err)
	}
	if normalizedSymbol != symbol {
		label := deferredValidationLabelString(context)

		return fmt.Errorf("%s must use canonical uppercase format", label)
	}

	return nil
}

// Rejects collateral/asset symbols that are valid only after trimming or
// uppercasing. Pass context as for [ValidateCanonicalSymbol].
func ValidateCanonicalCollateralSymbol[T deferredValidationLabel](assetName Asset, context T) error {
	normalisedAssetName, err := ValidateAndNormalizeCollateralSymbol(assetName)
	if err != nil {
		label := deferredValidationLabelString(context)
		if label == "symbol" {
			return err
		}

		return fmt.Errorf("%s: %w", label, err)
	}
	if normalisedAssetName != assetName {
		label := deferredValidationLabelString(context)

		return fmt.Errorf("%s must use canonical uppercase format", label)
	}

	return nil
}

// Rejects client-side strings that only become valid after trimming.
func ValidateCanonicalClientSideString(value string, maxLength int, fieldName string) error {
	validated, err := snx_lib_api_validation_utils.ValidateClientSideString(
		value,
		maxLength,
		snx_lib_api_validation_utils.ClientSideOption_Trim,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	if validated != value {
		return fmt.Errorf("%s must not include leading or trailing whitespace", fieldName)
	}

	return nil
}
