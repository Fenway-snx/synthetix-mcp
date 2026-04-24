package trade

import (
	"github.com/go-viper/mapstructure/v2"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/trade/getPortfolio
*/

type PortfolioRequest struct {
	User          WalletAddress `json:"user"`
	TimestampFrom Timestamp     `json:"timestampFrom"`
	TimestampTo   Timestamp     `json:"timestampTo"`
	Granularity   string        `json:"granularity"`
}

// PortfolioAsset represents an asset in the portfolio
type PortfolioAsset struct {
	Asset         string `json:"asset"`         // Asset symbol (e.g., "BTC", "ETH", "USD")
	Free          string `json:"free"`          // Available balance
	Locked        string `json:"locked"`        // Locked balance (in orders)
	Total         string `json:"total"`         // Total balance
	UnrealizedPnL string `json:"unrealizedPnL"` // Unrealized profit/loss
	MarkPrice     Price  `json:"markPrice"`     // Current mark price
	Value         string `json:"value"`         // Total value in USD
}

// Portfolio represents the response for portfolio data
type PortfolioSnapshot struct {
	Assets    []PortfolioAsset `json:"assets"`
	Timestamp Timestamp        `json:"timestamp"`
}

type PortfolioResponse []PortfolioSnapshot

// generateMockPortfolio creates mock portfolio data
func generateMockPortfolio() PortfolioResponse {
	return PortfolioResponse{
		{
			Assets: []PortfolioAsset{
				{
					Asset:         "BTC",
					Free:          "0.5",
					Locked:        "0.1",
					Total:         "0.6",
					UnrealizedPnL: "1250.00",
					MarkPrice:     snx_lib_api_types.PriceFromStringUnvalidated("45000.50"),
					Value:         "27000.30",
				},
			},
			Timestamp: 1704067200000,
		},
		{
			Assets: []PortfolioAsset{
				{
					Asset:         "ETH",
					Free:          "5.0",
					Locked:        "1.5",
					Total:         "6.5",
					UnrealizedPnL: "-150.00",
					MarkPrice:     snx_lib_api_types.PriceFromStringUnvalidated("3000.75"),
					Value:         "19500.00",
				},
			},
			Timestamp: 1704067200000,
		},
		{
			Assets: []PortfolioAsset{
				{
					Asset:         "SOL",
					Free:          "100",
					Locked:        "50",
					Total:         "150",
					UnrealizedPnL: "375.00",
					MarkPrice:     snx_lib_api_types.PriceFromStringUnvalidated("100.25"),
					Value:         "15037.50",
				},
			},
			Timestamp: 1704067200000,
		},
		{
			Assets: []PortfolioAsset{
				{
					Asset:         "USD",
					Free:          "10000.00",
					Locked:        "0.00",
					Total:         "10000.00",
					UnrealizedPnL: "0.00",
					MarkPrice:     snx_lib_api_types.PriceFromStringUnvalidated("1.00"),
					Value:         "10000.00",
				},
			},
			Timestamp: 1704067200000,
		},
	}
}

// Handler for "getPortfolio".
//
//dd:span
func Handle_getPortfolio(
	ctx TradeContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	var req PortfolioRequest
	err := mapstructure.Decode(params, &req)
	if err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](ctx.ClientRequestId, snx_lib_api_json.ErrorCodeInvalidFormat, "Invalid request body", nil)
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.User, snx_lib_api_validation.MaxEthAddressLength, "user"); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}
	if err := snx_lib_api_validation.ValidateStringMaxLength(req.Granularity, snx_lib_api_validation.MaxEnumFieldLength, "granularity"); err != nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewValidationErrorResponse[any](ctx.ClientRequestId, err.Error(), nil)
	}

	portfolio := generateMockPortfolio()
	ctx.Logger.Info("Generated mock portfolio", "count", len(portfolio))

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, portfolio)
}
