package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/synthetixio/synthetix-go/types"

	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type historyFilterInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	StartTime    int64     `json:"startTime,omitempty" jsonschema:"UTC start time in milliseconds since epoch."`
	EndTime      int64     `json:"endTime,omitempty" jsonschema:"UTC end time in milliseconds since epoch."`
	Offset       int32     `json:"offset,omitempty" jsonschema:"Pagination offset."`
	Limit        int32     `json:"limit,omitempty" jsonschema:"Maximum rows to return."`
}

type balanceUpdatesInput struct {
	historyFilterInput
	ActionFilter string `json:"actionFilter,omitempty" jsonschema:"Filter by balance update action."`
}

type positionHistoryInput struct {
	historyFilterInput
	Symbol string `json:"symbol,omitempty" jsonschema:"Market symbol filter, e.g. BTC-USDT."`
}

type tradesForPositionInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the authenticated subaccount."`
	PositionID   string    `json:"positionId" jsonschema:"Position ID to fetch trades for."`
	Offset       int32     `json:"offset,omitempty" jsonschema:"Pagination offset."`
	Limit        int32     `json:"limit,omitempty" jsonschema:"Maximum rows to return."`
}

type fundingRateHistoryInput struct {
	Symbol    string `json:"symbol" jsonschema:"Market symbol, e.g. BTC-USDT. Short symbols such as BTC are normalized to BTC-USDT."`
	StartTime int64  `json:"startTime,omitempty" jsonschema:"UTC start time in milliseconds since epoch."`
	EndTime   int64  `json:"endTime,omitempty" jsonschema:"UTC end time in milliseconds since epoch."`
	Limit     int32  `json:"limit,omitempty" jsonschema:"Maximum rows to return."`
}

type balanceUpdatesOutput struct {
	Meta responseMeta `json:"_meta"`
	types.BalanceUpdatesResponse
}

type transfersOutput struct {
	Meta responseMeta `json:"_meta"`
	types.TransfersResponse
}

type positionHistoryOutput struct {
	Meta responseMeta `json:"_meta"`
	types.PositionHistoryResponse
}

type portfolioOutput struct {
	Meta      responseMeta              `json:"_meta"`
	Portfolio []types.PortfolioSnapshot `json:"portfolio"`
}

type feesOutput struct {
	Meta responseMeta `json:"_meta"`
	types.FeesResponse
}

type rateLimitsOutput struct {
	Meta responseMeta `json:"_meta"`
	types.RateLimitsResponse
}

type tradesForPositionOutput struct {
	Meta responseMeta `json:"_meta"`
	types.TradesForPositionResponse
}

type delegatedSignersOutput struct {
	DelegatedSigners types.DelegatedSignersResponse `json:"delegatedSigners"`
	Meta             responseMeta                   `json:"_meta"`
}

type delegationsForDelegateOutput struct {
	Delegations types.DelegationsForDelegateResponse `json:"delegations"`
	Meta        responseMeta                         `json:"_meta"`
}

type fundingRateHistoryOutput struct {
	Meta responseMeta `json:"_meta"`
	types.FundingRateHistoryResponse
}

func RegisterAccountExtraTools(
	server *mcp.Server,
	deps *ToolDeps,
	tradeReads *TradeReadClient,
) {
	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_balance_updates",
		Description: "Return authenticated collateral and balance ledger updates with optional time, action, offset, and limit filters.",
	}, func(in balanceUpdatesInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input balanceUpdatesInput) (*mcp.CallToolResult, balanceUpdatesOutput, error) {
			if tradeReads == nil {
				return toolErrorResponse[balanceUpdatesOutput](ErrReadUnavailable)
			}
			resp, err := tradeReads.GetBalanceUpdates(ctx, tc, balanceUpdateParams(input))
			if err != nil {
				return toolErrorResponse[balanceUpdatesOutput](fmt.Errorf("get balance updates: %w", err))
			}
			return nil, balanceUpdatesOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated)), BalanceUpdatesResponse: resp}, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_transfers",
		Description: "Return authenticated collateral transfer history with optional time, offset, and limit filters.",
	}, func(in historyFilterInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input historyFilterInput) (*mcp.CallToolResult, transfersOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetTransfers(ctx, tc, historyParams(input))
			if err != nil {
				return toolErrorResponse[transfersOutput](fmt.Errorf("get transfers: %w", err))
			}
			return nil, transfersOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated)), TransfersResponse: resp}, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_position_history",
		Description: "Return historical positions with optional symbol, time, offset, and limit filters.",
	}, func(in positionHistoryInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input positionHistoryInput) (*mcp.CallToolResult, positionHistoryOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetPositionHistory(ctx, tc, positionHistoryParams(input))
			if err != nil {
				return toolErrorResponse[positionHistoryOutput](fmt.Errorf("get position history: %w", err))
			}
			return nil, positionHistoryOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated)), PositionHistoryResponse: resp}, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_portfolio",
		Description: "Return authenticated portfolio snapshots.",
	}, func(in subaccountScopedInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, _ subaccountScopedInput) (*mcp.CallToolResult, portfolioOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetPortfolio(ctx, tc)
			if err != nil {
				return toolErrorResponse[portfolioOutput](fmt.Errorf("get portfolio: %w", err))
			}
			var portfolio []types.PortfolioSnapshot
			if resp != nil {
				portfolio = []types.PortfolioSnapshot(*resp)
			}
			return nil, portfolioOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated)), Portfolio: portfolio}, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{Name: "get_fees", Description: "Return fee tiers and the authenticated subaccount fee tier."},
		func(in subaccountScopedInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, _ subaccountScopedInput) (*mcp.CallToolResult, feesOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetFees(ctx, tc)
			if err != nil {
				return toolErrorResponse[feesOutput](fmt.Errorf("get fees: %w", err))
			}
			out := feesOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated))}
			if resp != nil {
				out.FeesResponse = *resp
			}
			return nil, out, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{Name: "get_rate_limits", Description: "Return authenticated API rate-limit usage and reset information."},
		func(in subaccountScopedInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, _ subaccountScopedInput) (*mcp.CallToolResult, rateLimitsOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetRateLimits(ctx, tc)
			if err != nil {
				return toolErrorResponse[rateLimitsOutput](fmt.Errorf("get rate limits: %w", err))
			}
			out := rateLimitsOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated))}
			if resp != nil {
				out.RateLimitsResponse = *resp
			}
			return nil, out, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{Name: "get_trades_for_position", Description: "Return trades associated with one position ID."},
		func(in tradesForPositionInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input tradesForPositionInput) (*mcp.CallToolResult, tradesForPositionOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetTradesForPosition(ctx, tc, tradesForPositionParams(input))
			if err != nil {
				return toolErrorResponse[tradesForPositionOutput](fmt.Errorf("get trades for position: %w", err))
			}
			out := tradesForPositionOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated))}
			if resp != nil {
				out.TradesForPositionResponse = *resp
			}
			return nil, out, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{Name: "get_delegated_signers", Description: "Return delegated signers configured on the authenticated subaccount."},
		func(in subaccountScopedInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, _ subaccountScopedInput) (*mcp.CallToolResult, delegatedSignersOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetDelegatedSigners(ctx, tc)
			if err != nil {
				return toolErrorResponse[delegatedSignersOutput](fmt.Errorf("get delegated signers: %w", err))
			}
			return nil, delegatedSignersOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated)), DelegatedSigners: resp}, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{Name: "get_delegations_for_delegate", Description: "Return delegations granted to the authenticated delegate wallet."},
		func(in subaccountScopedInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, _ subaccountScopedInput) (*mcp.CallToolResult, delegationsForDelegateOutput, error) {
			resp, err := requireTradeReads(tradeReads).GetDelegationsForDelegate(ctx, tc)
			if err != nil {
				return toolErrorResponse[delegationsForDelegateOutput](fmt.Errorf("get delegations for delegate: %w", err))
			}
			return nil, delegationsForDelegateOutput{Meta: newResponseMeta(string(session.AuthModeAuthenticated)), Delegations: resp}, nil
		})

	addPublicTool(server, deps, &mcp.Tool{Name: "get_funding_rate_history", Description: "Return public funding-rate history for a market."},
		func(ctx context.Context, tc ToolContext, input fundingRateHistoryInput) (*mcp.CallToolResult, fundingRateHistoryOutput, error) {
			if deps == nil || deps.Clients == nil || deps.Clients.RESTInfo == nil {
				return toolErrorResponse[fundingRateHistoryOutput](errors.New("REST info backend is not configured"))
			}
			resp, err := deps.Clients.RESTInfo.GetFundingRateHistory(ctx, normalizeSymbol(input.Symbol), int(input.Limit), input.StartTime, input.EndTime)
			if err != nil {
				return toolErrorResponse[fundingRateHistoryOutput](fmt.Errorf("get funding rate history: %w", err))
			}
			out := fundingRateHistoryOutput{Meta: newResponseMeta(authModeFromContext(tc))}
			if resp != nil {
				out.FundingRateHistoryResponse = *resp
			}
			return nil, out, nil
		})
}

func requireTradeReads(tradeReads *TradeReadClient) *TradeReadClient {
	if tradeReads == nil {
		return &TradeReadClient{}
	}
	return tradeReads
}

func historyParams(input historyFilterInput) map[string]any {
	params := make(map[string]any, 5)
	if input.StartTime > 0 {
		params["startTime"] = input.StartTime
	}
	if input.EndTime > 0 {
		params["endTime"] = input.EndTime
	}
	if input.Offset > 0 {
		params["offset"] = input.Offset
	}
	if input.Limit > 0 {
		params["limit"] = input.Limit
	}
	return params
}

func balanceUpdateParams(input balanceUpdatesInput) map[string]any {
	params := historyParams(input.historyFilterInput)
	if input.ActionFilter != "" {
		params["actionFilter"] = input.ActionFilter
	}
	return params
}

func positionHistoryParams(input positionHistoryInput) map[string]any {
	params := historyParams(input.historyFilterInput)
	if input.Symbol != "" {
		params["symbol"] = normalizeSymbol(input.Symbol)
	}
	return params
}

func tradesForPositionParams(input tradesForPositionInput) map[string]any {
	params := make(map[string]any, 3)
	params["positionId"] = input.PositionID
	if input.Offset > 0 {
		params["offset"] = input.Offset
	}
	if input.Limit > 0 {
		params["limit"] = input.Limit
	}
	return params
}
