package tools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	shopspring_decimal "github.com/shopspring/decimal"

	"github.com/synthetixio/synthetix-go/types"
	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type subaccountScopedInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount. If provided, must match the authenticated subaccount."`
}

type getPositionsInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Symbol       string    `json:"symbol,omitempty" jsonschema:"Filter positions to a single market symbol, e.g. BTC-USDT. Omit for all positions."`
	StartTime    int64     `json:"startTime,omitempty" jsonschema:"UTC start time filter in milliseconds since epoch."`
	EndTime      int64     `json:"endTime,omitempty" jsonschema:"UTC end time filter in milliseconds since epoch."`
	Offset       int32     `json:"offset,omitempty" jsonschema:"Pagination offset (0-based). Use with limit for paging."`
	Limit        int32     `json:"limit,omitempty" jsonschema:"Maximum number of positions to return per page."`
}

type getOpenOrdersInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Symbol       string    `json:"symbol,omitempty" jsonschema:"Filter orders to a single market symbol, e.g. ETH-USDT. Omit for all markets."`
	StartTime    int64     `json:"startTime,omitempty" jsonschema:"UTC start time filter in milliseconds since epoch."`
	EndTime      int64     `json:"endTime,omitempty" jsonschema:"UTC end time filter in milliseconds since epoch."`
	Offset       int32     `json:"offset,omitempty" jsonschema:"Pagination offset (0-based)."`
	Limit        int32     `json:"limit,omitempty" jsonschema:"Maximum number of orders to return per page."`
}

type getOrderHistoryInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	StartTime    int64     `json:"startTime,omitempty" jsonschema:"UTC start time filter in milliseconds since epoch."`
	EndTime      int64     `json:"endTime,omitempty" jsonschema:"UTC end time filter in milliseconds since epoch."`
	Offset       int32     `json:"offset,omitempty" jsonschema:"Pagination offset (0-based)."`
	Limit        int32     `json:"limit,omitempty" jsonschema:"Maximum number of orders to return per page."`
	StatusFilter []string  `json:"statusFilter,omitempty" jsonschema:"Filter by order status values, e.g. ['filled','cancelled']. Omit for all statuses."`
}

type getTradeHistoryInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Symbol       string    `json:"symbol,omitempty" jsonschema:"Filter trades to a single market symbol. Omit for all markets."`
	StartTime    int64     `json:"startTime,omitempty" jsonschema:"UTC start time filter in milliseconds since epoch."`
	EndTime      int64     `json:"endTime,omitempty" jsonschema:"UTC end time filter in milliseconds since epoch."`
	Offset       int32     `json:"offset,omitempty" jsonschema:"Pagination offset (0-based)."`
	Limit        int32     `json:"limit,omitempty" jsonschema:"Maximum number of trades to return per page."`
}

type getFundingPaymentsInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Symbol       string    `json:"symbol,omitempty" jsonschema:"Filter funding payments to a single market symbol. Omit for all markets."`
	StartTime    int64     `json:"startTime,omitempty" jsonschema:"UTC start time filter in milliseconds since epoch."`
	EndTime      int64     `json:"endTime,omitempty" jsonschema:"UTC end time filter in milliseconds since epoch."`
	Limit        int32     `json:"limit,omitempty" jsonschema:"Maximum number of funding payments to return."`
}

type getPerformanceHistoryInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Period       string    `json:"period,omitempty" jsonschema:"Lookback period for performance snapshots: 1d, 7d, 30d, or 90d."`
}

type collateralOutput struct {
	Collateral              string `json:"collateral"`
	PendingWithdrawalAmount string `json:"pendingWithdrawalAmount"`
	Quantity                string `json:"quantity"`
	WithdrawableAmount      string `json:"withdrawableAmount"`
}

type marginSummaryOutput struct {
	AccountValue         string `json:"accountValue"`
	AdjustedAccountValue string `json:"adjustedAccountValue"`
	AvailableMargin      string `json:"availableMargin"`
	InitialMargin        string `json:"initialMargin"`
	MaintenanceMargin    string `json:"maintenanceMargin"`
	UnrealizedPnl        string `json:"unrealizedPnl"`
	Withdrawable         string `json:"withdrawable"`
}

type feeRatesOutput struct {
	MakerFeeRate string `json:"makerFeeRate"`
	TakerFeeRate string `json:"takerFeeRate"`
	TierName     string `json:"tierName"`
}

type accountSummaryOutput struct {
	Meta            responseMeta        `json:"_meta"`
	Collaterals     []collateralOutput  `json:"collaterals"`
	FeeRates        feeRatesOutput      `json:"feeRates"`
	Leverages       map[string]uint32   `json:"leverages"`
	MarginSummary   marginSummaryOutput `json:"marginSummary"`
	MasterAccountID int64               `json:"masterAccountId,omitempty,string"`
	MaxSubAccounts  int64               `json:"maxSubAccounts"`
	Name            string              `json:"name"`
	OpenOrderCount  int                 `json:"openOrderCount"`
	PositionCount   int                 `json:"positionCount"`
	SubAccountID    int64               `json:"subAccountId,string"`
	WalletAddress   string              `json:"walletAddress"`
}

type orderIDOutput struct {
	ClientID string `json:"clientId,omitempty"`
	VenueID  uint64 `json:"venueId,string"`
}

type positionOutput struct {
	CreatedAt          string          `json:"createdAt,omitempty"`
	EntryPrice         string          `json:"entryPrice"`
	LiquidationPrice   string          `json:"liquidationPrice"`
	MaintenanceMargin  string          `json:"maintenanceMargin"`
	NetFundingPnl      string          `json:"netFundingPnl"`
	OrderAction        string          `json:"orderAction"`
	PositionID         uint64          `json:"positionId,string"`
	Quantity           string          `json:"quantity"`
	RealizedPnl        string          `json:"realizedPnl"`
	Side               string          `json:"side"`
	StopLossOrderIds   []orderIDOutput `json:"stopLossOrderIds"`
	SubAccountID       int64           `json:"subAccountId,string"`
	Symbol             string          `json:"symbol"`
	TakeProfitOrderIds []orderIDOutput `json:"takeProfitOrderIds"`
	UnrealizedPnl      string          `json:"unrealizedPnl"`
	UpdatedAt          string          `json:"updatedAt,omitempty"`
	UsedMargin         string          `json:"usedMargin"`
}

type positionsOutput struct {
	Meta         responseMeta     `json:"_meta"`
	Limit        int32            `json:"limit"`
	Offset       int32            `json:"offset"`
	Positions    []positionOutput `json:"positions"`
	SubAccountID int64            `json:"subAccountId,string"`
}

type openOrderOutput struct {
	ClosePosition     bool          `json:"closePosition"`
	CreatedAt         string        `json:"createdAt,omitempty"`
	Direction         string        `json:"direction"`
	OrderID           orderIDOutput `json:"orderId"`
	PositionID        uint64        `json:"positionId,string"`
	PostOnly          bool          `json:"postOnly"`
	Price             string        `json:"price"`
	Quantity          string        `json:"quantity"`
	ReduceOnly        bool          `json:"reduceOnly"`
	RemainingQuantity string        `json:"remainingQuantity"`
	Side              string        `json:"side"`
	StopLossOrderID   orderIDOutput `json:"stopLossOrderId,omitempty"`
	SubAccountID      int64         `json:"subAccountId,string"`
	Symbol            string        `json:"symbol"`
	TakeProfitOrderID orderIDOutput `json:"takeProfitOrderId,omitempty"`
	TimeInForce       string        `json:"timeInForce"`
	TriggerPrice      string        `json:"triggerPrice"`
	TriggerPriceType  string        `json:"triggerPriceType"`
	Type              string        `json:"type"`
	UpdatedAt         string        `json:"updatedAt,omitempty"`
}

type openOrdersOutput struct {
	Limit        int32             `json:"limit"`
	Meta         responseMeta      `json:"_meta"`
	Offset       int32             `json:"offset"`
	Orders       []openOrderOutput `json:"orders"`
	SubAccountID int64             `json:"subAccountId,string"`
}

type orderHistoryOutputItem struct {
	CreatedAt              string        `json:"createdAt,omitempty"`
	Direction              string        `json:"direction"`
	FilledPrice            string        `json:"filledPrice"`
	FilledQuantity         string        `json:"filledQuantity"`
	ID                     int64         `json:"id,string"`
	OrderID                orderIDOutput `json:"orderId"`
	PostOnly               bool          `json:"postOnly"`
	Price                  string        `json:"price"`
	Quantity               string        `json:"quantity"`
	ReduceOnly             bool          `json:"reduceOnly"`
	Side                   string        `json:"side"`
	Status                 string        `json:"status"`
	SubAccountID           int64         `json:"subAccountId,string"`
	Symbol                 string        `json:"symbol"`
	TimeInForce            string        `json:"timeInForce"`
	TriggeredByLiquidation bool          `json:"triggeredByLiquidation"`
	Type                   string        `json:"type"`
	UpdatedAt              string        `json:"updatedAt,omitempty"`
	Value                  string        `json:"value"`
}

type orderHistoryOutput struct {
	Limit        int32                    `json:"limit"`
	Meta         responseMeta             `json:"_meta"`
	Offset       int32                    `json:"offset"`
	Orders       []orderHistoryOutputItem `json:"orders"`
	SubAccountID int64                    `json:"subAccountId,string"`
}

type tradeHistoryOutputItem struct {
	ClosedPnl              string        `json:"closedPnl"`
	Direction              string        `json:"direction"`
	EntryPrice             string        `json:"entryPrice"`
	Fee                    string        `json:"fee"`
	FeeRate                string        `json:"feeRate"`
	FillType               string        `json:"fillType"`
	FilledPrice            string        `json:"filledPrice"`
	FilledQuantity         string        `json:"filledQuantity"`
	FilledValue            string        `json:"filledValue"`
	ID                     uint64        `json:"id,string"`
	MarkPrice              string        `json:"markPrice"`
	OrderID                orderIDOutput `json:"orderId"`
	OrderType              string        `json:"orderType"`
	PostOnly               bool          `json:"postOnly"`
	ReduceOnly             bool          `json:"reduceOnly"`
	SubAccountID           int64         `json:"subAccountId,string"`
	Symbol                 string        `json:"symbol"`
	TradedAt               string        `json:"tradedAt,omitempty"`
	TriggeredByLiquidation bool          `json:"triggeredByLiquidation"`
}

type tradeHistoryOutput struct {
	HasMore      bool                     `json:"hasMore"`
	Limit        int32                    `json:"limit"`
	Meta         responseMeta             `json:"_meta"`
	Offset       int32                    `json:"offset"`
	SubAccountID int64                    `json:"subAccountId,string"`
	TotalCount   int32                    `json:"totalCount"`
	Trades       []tradeHistoryOutputItem `json:"trades"`
}

type fundingSummaryOutput struct {
	AveragePaymentSize   string `json:"averagePaymentSize"`
	NetFunding           string `json:"netFunding"`
	TotalFundingPaid     string `json:"totalFundingPaid"`
	TotalFundingReceived string `json:"totalFundingReceived"`
	TotalPayments        string `json:"totalPayments"`
}

type fundingPaymentOutput struct {
	FundingRate  string `json:"fundingRate"`
	FundingTime  string `json:"fundingTime,omitempty"`
	Payment      string `json:"payment"`
	PaymentID    string `json:"paymentId"`
	PaymentTime  string `json:"paymentTime,omitempty"`
	PositionSize string `json:"positionSize"`
	Symbol       string `json:"symbol"`
}

type fundingPaymentsOutput struct {
	Meta         responseMeta           `json:"_meta"`
	Payments     []fundingPaymentOutput `json:"payments"`
	SubAccountID int64                  `json:"subAccountId,string"`
	Summary      fundingSummaryOutput   `json:"summary"`
}

type performancePointOutput struct {
	AccountValue string `json:"accountValue"`
	Pnl          string `json:"pnl"`
	SampledAt    int64  `json:"sampledAt"`
}

type performanceHistoryOutput struct {
	Meta         responseMeta             `json:"_meta"`
	Performance  []performancePointOutput `json:"performance"`
	Period       string                   `json:"period"`
	SubAccountID int64                    `json:"subAccountId,string"`
	Volume       string                   `json:"volume"`
}

func RegisterAccountTools(
	server *mcp.Server,
	deps *ToolDeps,
	tradeReads *TradeReadClient,
) {
	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_account_summary",
		Description: "Return a comprehensive account overview: collateral balances, margin health (available/initial/maintenance margin, unrealized PnL), fee tier, position count, and open order count. Requires an authenticated session. Call this before trading to assess margin capacity.",
	}, func(in subaccountScopedInput) *int64 { return int64Optional(in.SubAccountID.Int64()) }, func(ctx context.Context, tc ToolContext, input subaccountScopedInput) (*mcp.CallToolResult, accountSummaryOutput, error) {
		if tradeReads == nil {
			return toolErrorResponse[accountSummaryOutput](errors.New("get account summary: REST trade backend is not configured; authenticated account data is unavailable"))
		}
		subaccount, err := tradeReads.GetSubAccount(ctx, tc)
		if err != nil {
			if errors.Is(err, ErrReadUnavailable) || errors.Is(err, ErrBrokerSubAccountMismatch) {
				return toolErrorResponse[accountSummaryOutput](fmt.Errorf("get account summary: %w", err))
			}
			return toolErrorResponse[accountSummaryOutput](fmt.Errorf("get account summary: %w", err))
		}
		if subaccount == nil {
			return toolErrorResponse[accountSummaryOutput](errors.New("get account summary: subaccount response was empty"))
		}

		openOrders, err := tradeReads.GetOpenOrders(ctx, tc)
		if err != nil {
			return toolErrorResponse[accountSummaryOutput](fmt.Errorf("get open orders for summary: %w", err))
		}

		var masterID int64
		if subaccount.MasterAccountID != nil {
			masterID = strconvParseInt64Silent(*subaccount.MasterAccountID)
		}

		output := accountSummaryOutput{
			Meta:            newResponseMeta(string(session.AuthModeAuthenticated)),
			Collaterals:     mapRESTCollaterals(subaccount.Collaterals),
			FeeRates:        mapRESTFeeRates(subaccount.FeeRates),
			Leverages:       mapLeverages(subaccount.MarketPreferences.Leverages),
			MarginSummary:   mapRESTMarginSummary(subaccount.MarginSummary),
			MasterAccountID: masterID,
			MaxSubAccounts:  subaccount.AccountLimits.MaxSubAccounts,
			Name:            subaccount.Name,
			OpenOrderCount:  len(openOrders),
			PositionCount:   len(subaccount.Positions),
			SubAccountID:    strconvParseInt64Silent(subaccount.SubAccountID),
			WalletAddress:   tc.State.WalletAddress,
		}
		card := renderAccountSummaryCard(output)
		if res, err := cards.Attach(card, output); err == nil && res != nil {
			return res, output, nil
		}
		return nil, output, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_positions",
		Description: "Return open positions with entry price, unrealized PnL, liquidation price, used margin, and associated TP/SL orders. Filter by symbol for a single market or omit for all positions. Supports offset/limit pagination.",
	}, func(in getPositionsInput) *int64 { return int64Optional(in.SubAccountID.Int64()) }, func(ctx context.Context, tc ToolContext, input getPositionsInput) (*mcp.CallToolResult, positionsOutput, error) {
		if tradeReads == nil {
			return toolErrorResponse[positionsOutput](errors.New("get positions: REST trade backend is not configured"))
		}
		all, err := tradeReads.GetPositions(ctx, tc)
		if err != nil {
			return toolErrorResponse[positionsOutput](fmt.Errorf("get positions: %w", err))
		}

		// REST /v1/trade getPositions returns the full list per call with
		// no symbol filter or time-bucket filter — we emulate both
		// client-side. startTime/endTime filter on `createdAt` milliseconds.
		filtered := make([]types.Position, 0, len(all))
		symbolFilter := strings.ToUpper(strings.TrimSpace(input.Symbol))
		for _, p := range all {
			if symbolFilter != "" && strings.ToUpper(p.Symbol) != symbolFilter {
				continue
			}
			if input.StartTime > 0 && p.CreatedAt < input.StartTime {
				continue
			}
			if input.EndTime > 0 && p.CreatedAt > input.EndTime {
				continue
			}
			filtered = append(filtered, p)
		}

		// Client-side offset/limit (Offset=0 default, Limit=0 means
		// "no cap"): the REST list endpoint returns every position in
		// a single call.
		offset := input.Offset
		if offset < 0 {
			offset = 0
		}
		end := int32(len(filtered))
		if input.Limit > 0 && offset+input.Limit < end {
			end = offset + input.Limit
		}
		var page []types.Position
		if int(offset) < len(filtered) {
			page = filtered[offset:end]
		}

		output := positionsOutput{
			Meta:         newResponseMeta(string(session.AuthModeAuthenticated)),
			Limit:        input.Limit,
			Offset:       offset,
			Positions:    mapRESTPositions(page),
			SubAccountID: tc.State.SubAccountID,
		}
		card := renderPositionsCard(output)
		if res, err := cards.Attach(card, output); err == nil && res != nil {
			return res, output, nil
		}
		return nil, output, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_open_orders",
		Description: "Return pending open orders with type, side, price, quantity, remaining quantity, time-in-force, and trigger conditions. Filter by symbol, time range, or paginate with offset/limit. Call this before cancel_all_orders to review what will be cancelled.",
	}, func(in getOpenOrdersInput) *int64 { return int64Optional(in.SubAccountID.Int64()) }, func(ctx context.Context, tc ToolContext, input getOpenOrdersInput) (*mcp.CallToolResult, openOrdersOutput, error) {
		if tradeReads == nil {
			return toolErrorResponse[openOrdersOutput](errors.New("get open orders: REST trade backend is not configured"))
		}
		all, err := tradeReads.GetOpenOrders(ctx, tc)
		if err != nil {
			return toolErrorResponse[openOrdersOutput](fmt.Errorf("get open orders: %w", err))
		}

		// Symbol + time filters applied client-side; REST getOpenOrders
		// supports server-side filters but calling them would add extra
		// round-trips and force the shim to grow a params-carrying
		// variant we haven't wired yet. Cost is bounded because the
		// server already caps the response at 50 items by default.
		filtered := make([]types.OpenOrder, 0, len(all))
		symbolFilter := strings.ToUpper(strings.TrimSpace(input.Symbol))
		for _, o := range all {
			if symbolFilter != "" && strings.ToUpper(o.Symbol) != symbolFilter {
				continue
			}
			if input.StartTime > 0 && o.CreatedTime < input.StartTime {
				continue
			}
			if input.EndTime > 0 && o.CreatedTime > input.EndTime {
				continue
			}
			filtered = append(filtered, o)
		}

		offset := input.Offset
		if offset < 0 {
			offset = 0
		}
		end := int32(len(filtered))
		if input.Limit > 0 && offset+input.Limit < end {
			end = offset + input.Limit
		}
		var page []types.OpenOrder
		if int(offset) < len(filtered) {
			page = filtered[offset:end]
		}

		output := openOrdersOutput{
			Limit:        input.Limit,
			Meta:         newResponseMeta(string(session.AuthModeAuthenticated)),
			Offset:       offset,
			Orders:       mapRESTOpenOrders(page),
			SubAccountID: tc.State.SubAccountID,
		}
		card := renderOpenOrdersCard(output)
		if res, err := cards.Attach(card, output); err == nil && res != nil {
			return res, output, nil
		}
		return nil, output, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_order_history",
		Description: "Return historical orders across all statuses (filled, cancelled, expired, rejected). Filter by time range or status. Includes fill price, filled quantity, and order value. Supports offset/limit pagination.",
	}, func(in getOrderHistoryInput) *int64 { return int64Optional(in.SubAccountID.Int64()) }, func(ctx context.Context, tc ToolContext, input getOrderHistoryInput) (*mcp.CallToolResult, orderHistoryOutput, error) {
		if tradeReads == nil {
			return toolErrorResponse[orderHistoryOutput](errors.New("get order history: REST trade backend is not configured; authenticated account data is unavailable"))
		}
		params := buildOrderHistoryParams(input)
		items, err := tradeReads.GetOrderHistory(ctx, tc, params)
		if err != nil {
			return toolErrorResponse[orderHistoryOutput](fmt.Errorf("get order history: %w", err))
		}

		// The REST getOrderHistory endpoint has no `offset` filter —
		// it returns the upstream slice already sorted desc by
		// createdTime. Apply client-side offset+limit so the tool
		// contract (offset-based pagination) is preserved while the
		// backing transport swaps.
		limit := input.Limit
		offset := input.Offset
		page := pageOrderHistory(items, offset, limit)

		return nil, orderHistoryOutput{
			Limit:        limit,
			Meta:         newResponseMeta(string(session.AuthModeAuthenticated)),
			Offset:       offset,
			Orders:       mapRESTOrderHistory(page),
			SubAccountID: tc.State.SubAccountID,
		}, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_trade_history",
		Description: "Return historical trade executions with fill price, quantity, fee, fee rate, closed PnL, and fill type. Filter by symbol or time range. Returns hasMore and totalCount for pagination.",
	}, func(in getTradeHistoryInput) *int64 { return int64Optional(in.SubAccountID.Int64()) }, func(ctx context.Context, tc ToolContext, input getTradeHistoryInput) (*mcp.CallToolResult, tradeHistoryOutput, error) {
		if tradeReads == nil {
			return toolErrorResponse[tradeHistoryOutput](errors.New("get trade history: REST trade backend is not configured; authenticated account data is unavailable"))
		}
		params := buildTradeHistoryParams(input)
		resp, err := tradeReads.GetTrades(ctx, tc, params)
		if err != nil {
			return toolErrorResponse[tradeHistoryOutput](fmt.Errorf("get trade history: %w", err))
		}

		return nil, tradeHistoryOutput{
			HasMore:      resp.HasMore,
			Limit:        input.Limit,
			Meta:         newResponseMeta(string(session.AuthModeAuthenticated)),
			Offset:       input.Offset,
			SubAccountID: tc.State.SubAccountID,
			TotalCount:   int32(resp.Total),
			Trades:       mapRESTTradeHistory(resp.Trades),
		}, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_funding_payments",
		Description: "Return funding payment history with per-payment details (rate, payment amount, position size) and an aggregate summary (net funding, total paid/received, average payment size). Filter by symbol or time range.",
	}, func(in getFundingPaymentsInput) *int64 { return int64Optional(in.SubAccountID.Int64()) }, func(ctx context.Context, tc ToolContext, input getFundingPaymentsInput) (*mcp.CallToolResult, fundingPaymentsOutput, error) {
		if tradeReads == nil {
			return toolErrorResponse[fundingPaymentsOutput](errors.New("get funding payments: REST trade backend is not configured; authenticated account data is unavailable"))
		}
		params := buildFundingPaymentsParams(input)
		resp, err := tradeReads.GetFundingPayments(ctx, tc, params)
		if err != nil {
			return toolErrorResponse[fundingPaymentsOutput](fmt.Errorf("get funding payments: %w", err))
		}

		return nil, fundingPaymentsOutput{
			Meta:         newResponseMeta(string(session.AuthModeAuthenticated)),
			Payments:     mapRESTFundingPayments(resp.FundingHistory),
			SubAccountID: tc.State.SubAccountID,
			Summary:      mapRESTFundingSummary(resp.Summary),
		}, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_performance_history",
		Description: "Return time-series account value and PnL snapshots for a period (e.g., 1d, 7d, 30d). Includes total volume and per-sample account value and PnL for charting or trend analysis.",
	}, func(in getPerformanceHistoryInput) *int64 { return int64Optional(in.SubAccountID.Int64()) }, func(ctx context.Context, tc ToolContext, input getPerformanceHistoryInput) (*mcp.CallToolResult, performanceHistoryOutput, error) {
		if tradeReads == nil {
			return toolErrorResponse[performanceHistoryOutput](errors.New("get performance history: REST trade backend is not configured; authenticated account data is unavailable"))
		}
		var params map[string]any
		if input.Period != "" {
			params = map[string]any{"period": input.Period}
		}
		resp, err := tradeReads.GetPerformanceHistory(ctx, tc, params)
		if err != nil {
			return toolErrorResponse[performanceHistoryOutput](fmt.Errorf("get performance history: %w", err))
		}

		return nil, performanceHistoryOutput{
			Meta:         newResponseMeta(string(session.AuthModeAuthenticated)),
			Performance:  mapRESTPerformanceHistory(resp.Performance),
			Period:       resp.Period,
			SubAccountID: tc.State.SubAccountID,
			Volume:       resp.Performance.Volume,
		}, nil
	})
}

func int64Optional(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func mapLeverages(items map[string]uint32) map[string]uint32 {
	if items == nil {
		return map[string]uint32{}
	}
	return items
}

// mapRESTCollaterals maps the REST /v1/trade getSubAccount collaterals
// block to the tool's public output. The REST wire uses
// `pendingWithdraw` / `withdrawable`; we surface them on the existing
// tool-output field names `pendingWithdrawalAmount` / `withdrawableAmount`
// to preserve the Phase-0 contract.
func mapRESTCollaterals(items []types.CollateralResponse) []collateralOutput {
	out := make([]collateralOutput, 0, len(items))
	for _, c := range items {
		out = append(out, collateralOutput{
			Collateral:              c.Symbol,
			PendingWithdrawalAmount: c.PendingWithdraw,
			Quantity:                c.Quantity,
			WithdrawableAmount:      c.Withdrawable,
		})
	}
	return out
}

func mapRESTFeeRates(fr types.FeeRateInfo) feeRatesOutput {
	return feeRatesOutput{
		MakerFeeRate: fr.MakerFeeRate,
		TakerFeeRate: fr.TakerFeeRate,
		TierName:     fr.TierName,
	}
}

// mapRESTMarginSummary maps the REST crossMarginSummary block. The
// wire field `totalUnrealizedPnl` is surfaced as `unrealizedPnl` in
// the tool output (Phase-0 field name).
func mapRESTMarginSummary(s types.MarginSummary) marginSummaryOutput {
	return marginSummaryOutput{
		AccountValue:         s.AccountValue,
		AdjustedAccountValue: s.AdjustedAccountValue,
		AvailableMargin:      s.AvailableMargin,
		InitialMargin:        s.InitialMargin,
		MaintenanceMargin:    s.MaintenanceMargin,
		UnrealizedPnl:        s.UnrealizedPnl,
		Withdrawable:         s.Withdrawable,
	}
}

// mapRESTOrderIdentifier narrows the REST OrderIdentifier wire shape
// (VenueID string, ClientID string) to the tool's orderIDOutput
// (VenueID uint64, ClientID string). Out-of-range / non-numeric venue
// ids degrade to 0 rather than erroring the whole tool — the invariant
// the REST contract asserts is that venue ids fit in uint64.
func mapRESTOrderIdentifier(id types.OrderIdentifier) orderIDOutput {
	return orderIDOutput{
		ClientID: id.ClientID,
		VenueID:  strconvParseUint(id.VenueID),
	}
}

func mapRESTOrderIdentifiers(items []types.OrderIdentifier) []orderIDOutput {
	out := make([]orderIDOutput, 0, len(items))
	for _, id := range items {
		out = append(out, mapRESTOrderIdentifier(id))
	}
	return out
}

// mapRESTPositions maps REST getPositions items to the Phase-0
// positionOutput shape. Fields the REST wire omits (NetFundingPnl,
// OrderAction) are left as empty strings; the
// response meta already flags the migration.
func mapRESTPositions(items []types.Position) []positionOutput {
	out := make([]positionOutput, 0, len(items))
	for _, p := range items {
		out = append(out, positionOutput{
			CreatedAt:          formatMillis(p.CreatedAt),
			EntryPrice:         p.EntryPrice,
			LiquidationPrice:   p.LiquidationPrice,
			MaintenanceMargin:  p.MaintenanceMargin,
			NetFundingPnl:      p.NetFunding,
			OrderAction:        p.Status,
			PositionID:         strconvParseUint(p.PositionID),
			Quantity:           p.Quantity,
			RealizedPnl:        p.RealizedPnl,
			Side:               strings.ToUpper(p.Side),
			StopLossOrderIds:   mapRESTOrderIdentifiers(p.StopLossOrders),
			SubAccountID:       strconvParseInt64Silent(p.SubAccountID),
			Symbol:             p.Symbol,
			TakeProfitOrderIds: mapRESTOrderIdentifiers(p.TakeProfitOrders),
			UnrealizedPnl:      p.UnrealizedPnl,
			UpdatedAt:          formatMillis(p.UpdatedAt),
			UsedMargin:         p.UsedMargin,
		})
	}
	return out
}

// mapRESTOpenOrders maps REST getOpenOrders items to the Phase-0
// openOrderOutput shape. The REST wire carries
// `filledQuantity` (not `remainingQuantity`); we derive
// RemainingQuantity via Quantity-FilledQuantity when both fields are
// decimal-parseable, falling back to the raw Quantity when
// FilledQuantity is empty. Direction / PositionID / StopLoss/TakeProfit
// flat venue-id fields are left unset or derived from the
// OrderIdentifier object shape.
func mapRESTOpenOrders(items []types.OpenOrder) []openOrderOutput {
	out := make([]openOrderOutput, 0, len(items))
	for _, o := range items {
		var stopLossOut, takeProfitOut orderIDOutput
		if o.StopLossOrder != nil {
			stopLossOut = mapRESTOrderIdentifier(*o.StopLossOrder)
		}
		if o.TakeProfitOrder != nil {
			takeProfitOut = mapRESTOrderIdentifier(*o.TakeProfitOrder)
		}
		out = append(out, openOrderOutput{
			ClosePosition:     o.ClosePosition,
			CreatedAt:         formatMillis(o.CreatedTime),
			Direction:         "",
			OrderID:           mapRESTOrderIdentifier(o.Order),
			PositionID:        0,
			PostOnly:          o.PostOnly,
			Price:             o.Price,
			Quantity:          o.Quantity,
			ReduceOnly:        o.ReduceOnly,
			RemainingQuantity: remainingFromFilled(o.Quantity, o.FilledQuantity),
			Side:              strings.ToUpper(o.Side),
			StopLossOrderID:   stopLossOut,
			SubAccountID:      0,
			Symbol:            o.Symbol,
			TakeProfitOrderID: takeProfitOut,
			TimeInForce:       o.TimeInForce,
			TriggerPrice:      o.TriggerPrice,
			TriggerPriceType:  o.TriggerPriceType,
			Type:              o.Type,
			UpdatedAt:         formatMillis(o.UpdatedTime),
		})
	}
	return out
}

// strconvParseUint silently converts a decimal string to uint64,
// returning 0 on any parse failure. Callers in the REST-mapping path
// use this to coerce wire-format venue ids / position ids into the
// uint64 the Phase-0 tool contract exposes; they cannot propagate
// errors out because the mapping happens inside a one-shot tool
// response and the invariant (venue ids fit in uint64) is enforced
// upstream by the api-service.
func strconvParseUint(raw string) uint64 {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// strconvParseInt64Silent is the int64 twin of strconvParseUint —
// same rationale, used for SubAccountID / MasterAccountID fields that
// arrive as decimal strings over the wire but flow through tool
// outputs as int64.
func strconvParseInt64Silent(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// formatMillis renders a Unix millisecond timestamp as RFC-3339 UTC.
// Zero (the JSON "unset" sentinel for int64 timestamps on the REST
// wire) maps to the empty string so the `omitempty` tag on
// positionOutput.CreatedAt / openOrderOutput.CreatedAt actually drops
// the field instead of serialising "1970-01-01T00:00:00Z".
func formatMillis(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

// remainingFromFilled reconstructs the Phase-0 "remainingQuantity"
// field from the REST `quantity` + `filledQuantity` pair. Uses
// shopspring/decimal subtraction to stay precision-safe; falls back
// to the raw quantity when FilledQuantity is empty (i.e. no fills
// yet) or when either side isn't a well-formed decimal.
func remainingFromFilled(quantity, filled string) string {
	q := strings.TrimSpace(quantity)
	f := strings.TrimSpace(filled)
	if f == "" {
		return q
	}
	qd, qErr := shopspring_decimal.NewFromString(q)
	fd, fErr := shopspring_decimal.NewFromString(f)
	if qErr != nil || fErr != nil {
		return q
	}
	return qd.Sub(fd).String()
}

// ---------------------------------------------------------------------
// REST history-read helpers
// ---------------------------------------------------------------------

// buildOrderHistoryParams assembles the /v1/trade getOrderHistory
// params object from the tool input. The upstream handler reads
// symbol (canonical-uppercase), side, type (OrderType), status
// (array), startTime, endTime, limit, and clientOrderId. Fields
// the Phase-0 tool did not expose are left unset so api-service
// defaults apply.
func buildOrderHistoryParams(input getOrderHistoryInput) map[string]any {
	params := make(map[string]any, 8)
	if input.StartTime > 0 {
		params["startTime"] = input.StartTime
	}
	if input.EndTime > 0 {
		params["endTime"] = input.EndTime
	}
	if len(input.StatusFilter) > 0 {
		params["status"] = input.StatusFilter
	}
	// The REST endpoint caps a single call at limit=500 by default;
	// pass the tool-side limit through so a small caller limit
	// minimises upstream payload. Offset is NOT a REST knob — we
	// apply it client-side in pageOrderHistory.
	if input.Limit > 0 {
		params["limit"] = input.Limit
	}
	return params
}

// buildTradeHistoryParams assembles the /v1/trade getTrades params
// object. The upstream handler supports full offset+limit
// pagination natively, so — unlike getOrderHistory — we can thread
// the tool's offset through to the server and trust the returned
// hasMore/total fields.
func buildTradeHistoryParams(input getTradeHistoryInput) map[string]any {
	params := make(map[string]any, 6)
	if input.Symbol != "" {
		params["symbol"] = input.Symbol
	}
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

// buildFundingPaymentsParams assembles the /v1/trade
// getFundingPayments params object. Only symbol / time range /
// limit are honoured upstream; there is no server-side offset
// knob, so the tool's historical offset field (removed from the
// REST migration) is intentionally NOT threaded through.
func buildFundingPaymentsParams(input getFundingPaymentsInput) map[string]any {
	params := make(map[string]any, 4)
	if input.Symbol != "" {
		params["symbol"] = input.Symbol
	}
	if input.StartTime > 0 {
		params["startTime"] = input.StartTime
	}
	if input.EndTime > 0 {
		params["endTime"] = input.EndTime
	}
	if input.Limit > 0 {
		params["limit"] = input.Limit
	}
	return params
}

// pageOrderHistory applies client-side offset+limit to a REST
// getOrderHistory response. The upstream endpoint returns the full
// filter-matched slice (capped by the upstream limit knob) in
// creation-time-desc order; the Phase-0 tool contract exposes
// offset-based pagination. Keeping the pagination in the tool
// layer avoids a REST contract change.
func pageOrderHistory(items types.OrderHistoryResponse, offset, limit int32) types.OrderHistoryResponse {
	n := int32(len(items))
	if offset >= n {
		return types.OrderHistoryResponse{}
	}
	start := offset
	if start < 0 {
		start = 0
	}
	end := n
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return items[start:end]
}

// mapRESTOrderHistory maps REST getOrderHistory items to the
// Phase-0 orderHistoryOutputItem shape. Fields the REST wire omits
// (Direction / Value / SubAccountID on a per-item basis — the
// REST response carries them at the envelope level or drops them
// entirely) are left empty; the response _meta already flags
// migration.
func mapRESTOrderHistory(items types.OrderHistoryResponse) []orderHistoryOutputItem {
	out := make([]orderHistoryOutputItem, 0, len(items))
	for _, item := range items {
		out = append(out, orderHistoryOutputItem{
			CreatedAt:              formatMillis(item.CreatedTime),
			FilledPrice:            item.FilledPrice,
			FilledQuantity:         item.FilledQuantity,
			OrderID:                mapRESTOrderIdentifier(item.Order),
			PostOnly:               item.PostOnly,
			Price:                  item.Price,
			Quantity:               item.Quantity,
			ReduceOnly:             item.ReduceOnly,
			Side:                   item.Side,
			Status:                 item.Status,
			Symbol:                 item.Symbol,
			TimeInForce:            item.TimeInForce,
			TriggeredByLiquidation: item.TriggeredByLiquidation,
			Type:                   item.Type,
			UpdatedAt:              formatMillis(item.UpdateTime),
		})
	}
	return out
}

// mapRESTTradeHistory maps REST getTrades items to the Phase-0
// tradeHistoryOutputItem shape. The REST `maker` bool becomes the
// `fillType` string; map it back to the Phase-0
// string label so existing agents keep parsing correctly.
func mapRESTTradeHistory(items []types.TradeHistoryItem) []tradeHistoryOutputItem {
	out := make([]tradeHistoryOutputItem, 0, len(items))
	for _, item := range items {
		out = append(out, tradeHistoryOutputItem{
			ClosedPnl:              item.RealizedPnl,
			Direction:              item.Direction,
			EntryPrice:             item.EntryPrice,
			Fee:                    item.Fee,
			FeeRate:                item.FeeRate,
			FillType:               fillTypeFromMaker(item.Maker),
			FilledPrice:            item.Price,
			FilledQuantity:         item.Quantity,
			ID:                     strconvParseUint(item.TradeID),
			MarkPrice:              item.MarkPrice,
			OrderID:                mapRESTOrderIdentifier(item.Order),
			OrderType:              item.OrderType,
			PostOnly:               item.PostOnly,
			ReduceOnly:             item.ReduceOnly,
			Symbol:                 item.Symbol,
			TradedAt:               formatMillis(item.Timestamp),
			TriggeredByLiquidation: item.TriggeredByLiquidation,
		})
	}
	return out
}

// fillTypeFromMaker reconstructs the Phase-0 `fillType` label from
// the REST `maker` boolean. The REST wire only
// surfaces the maker bit + the triggered-by-liquidation bit; the
// other distinctions are not load-bearing for tool consumers.
func fillTypeFromMaker(maker bool) string {
	if maker {
		return "MAKER"
	}
	return "TAKER"
}

// mapRESTFundingPayments maps REST getFundingPayments rows to the
// Phase-0 fundingPaymentOutput shape. Uses the canonical `PaymentTime`
// / `FundingTime` fields in preference to the deprecated aliases so
// tool consumers see the same ISO-8601 rendering across refresh
// cycles even after the upstream drops the `timestamp` /
// `fundingTimestamp` legacy keys.
func mapRESTFundingPayments(items []types.FundingPayment) []fundingPaymentOutput {
	out := make([]fundingPaymentOutput, 0, len(items))
	for _, item := range items {
		out = append(out, fundingPaymentOutput{
			FundingRate:  item.FundingRate,
			FundingTime:  formatMillis(item.FundingTime),
			Payment:      item.Payment,
			PaymentID:    item.PaymentID,
			PaymentTime:  formatMillis(item.PaymentTime),
			PositionSize: item.PositionSize,
			Symbol:       item.Symbol,
		})
	}
	return out
}

// mapRESTFundingSummary maps REST getFundingPayments.summary to the
// Phase-0 fundingSummaryOutput shape. Straight rename from REST fields.
func mapRESTFundingSummary(s types.FundingSummary) fundingSummaryOutput {
	return fundingSummaryOutput{
		AveragePaymentSize:   s.AveragePaymentSize,
		NetFunding:           s.NetFunding,
		TotalFundingPaid:     s.TotalFundingPaid,
		TotalFundingReceived: s.TotalFundingReceived,
		TotalPayments:        s.TotalPayments,
	}
}

// mapRESTPerformanceHistory maps REST getPerformanceHistory points
// to the Phase-0 performancePointOutput shape. The REST wire uses
// `sampledAt` as a Unix millis int (matching Phase-0); no
// reshaping needed beyond field-by-field renaming.
func mapRESTPerformanceHistory(period types.PerformanceHistoryPeriod) []performancePointOutput {
	out := make([]performancePointOutput, 0, len(period.History))
	for _, p := range period.History {
		out = append(out, performancePointOutput{
			AccountValue: p.AccountValue,
			Pnl:          p.Pnl,
			SampledAt:    p.SampledAt,
		})
	}
	return out
}
