package info

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

/*
API Docs:
	XXXX
*/

// MaxLeverage represents the maximum leverage allowed for a symbol
type MaxLeverage struct {
	Symbol      Symbol `json:"symbol"`      // Trading pair symbol (e.g., "BTC-USD")
	MaxLeverage int    `json:"maxLeverage"` // Maximum allowed leverage
	MinLeverage int    `json:"minLeverage"` // Minimum allowed leverage
	StepSize    int    `json:"stepSize"`    // Leverage step size
	IsEnabled   bool   `json:"isEnabled"`   // Whether leverage trading is enabled
}

// MaxLeverages represents the response for max leverage data
type MaxLeverages []MaxLeverage

// generateMockMaxLeverage creates mock max leverage data
func generateMockMaxLeverage() MaxLeverages {
	return MaxLeverages{
		{
			Symbol:      "BTC-USD",
			MaxLeverage: 100,
			MinLeverage: 1,
			StepSize:    1,
			IsEnabled:   true,
		},
		{
			Symbol:      "ETH-USD",
			MaxLeverage: 100,
			MinLeverage: 1,
			StepSize:    1,
			IsEnabled:   true,
		},
		{
			Symbol:      "SOL-USD",
			MaxLeverage: 50,
			MinLeverage: 1,
			StepSize:    1,
			IsEnabled:   true,
		},
	}
}

// HandleMaxLeverage handles max leverage-related info requests
func HandleMaxLeverage(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	maxLeverages := generateMockMaxLeverage()
	ctx.Logger.Info("Generated mock max leverage data", "count", len(maxLeverages))

	resp := snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, maxLeverages)

	return HTTPStatusCode_200_OK, resp
}
