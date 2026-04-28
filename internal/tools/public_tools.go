package tools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/synthetixio/synthetix-go/restinfo"
	"github.com/synthetixio/synthetix-go/types"
)

type pingOutput struct {
	Meta   responseMeta `json:"_meta"`
	OK     bool         `json:"ok"`
	Status string       `json:"status"`
}

type serverInfoOutput struct {
	Meta                   responseMeta          `json:"_meta"`
	AgentBroker            serverInfoAgentBroker `json:"agentBroker"`
	AuthModes              []string              `json:"authModes"`
	DisabledFeatures       []string              `json:"disabledFeatures"`
	DocumentationResources []string              `json:"documentationResources"`
	EnabledChannels        []string              `json:"enabledChannels"`
	Environment            string                `json:"environment"`
	ServerName             string                `json:"serverName"`
	SupportsPublicMode     bool                  `json:"supportsPublicMode"`
	SupportsReplay         bool                  `json:"supportsReplay"`
	Transports             []string              `json:"transports"`
	Version                string                `json:"version"`
}

// Mirrors the broker capability flag emitted via get_context so an
// agent that only calls get_server_info still gets a clean "is the
// server signing for me?" answer. Wallet/owner/expiry/permissions
// are public data, surfaced for operator diagnostics; omitempty
// fields stay absent until the first broker write so the not-yet-
// bound state is visually distinct.
type serverInfoAgentBroker struct {
	ChainID           int               `json:"chainId,omitempty"`
	DefaultGuardrails *guardrailsOutput `json:"defaultGuardrails,omitempty"`
	DefaultPreset     string            `json:"defaultPreset,omitempty"`
	DelegationID      uint64            `json:"delegationId,omitempty"`
	Enabled           bool              `json:"enabled"`
	ExpiresAtUnix     int64             `json:"expiresAtUnix,omitempty"`
	Note              string            `json:"note"`
	OwnerAddress      string            `json:"ownerAddress,omitempty"`
	Permissions       []string          `json:"permissions,omitempty"`
	BrokerTools       []string          `json:"brokerTools"`
	SubAccountID      int64             `json:"subAccountId,omitempty"`
	SubaccountSource  string            `json:"subaccountSource,omitempty"`
	WalletAddress     string            `json:"walletAddress,omitempty"`
}

type listMarketsInput struct {
	Status string `json:"status,omitempty" jsonschema:"Filter by market status. 'open' returns only actively tradable markets (default). 'all' includes suspended and delisted markets."`
}

type MaintenanceTierOutput struct {
	InitialMarginRatio        string `json:"initialMarginRatio"`
	MaintenanceDeductionValue string `json:"maintenanceDeductionValue"`
	MaintenanceMarginRatio    string `json:"maintenanceMarginRatio"`
	MaxLeverage               uint32 `json:"maxLeverage"`
	MaxPositionSize           string `json:"maxPositionSize"`
	MinPositionSize           string `json:"minPositionSize"`
}

type MarketOutput struct {
	BaseAsset              string                  `json:"baseAsset"`
	ContractSize           string                  `json:"contractSize"`
	DefaultLeverage        uint32                  `json:"defaultLeverage"`
	Description            string                  `json:"description"`
	FundingRateCap         string                  `json:"fundingRateCap"`
	FundingRateFloor       string                  `json:"fundingRateFloor"`
	Id                     uint64                  `json:"id,string"`
	ImpactNotionalUsd      string                  `json:"impactNotionalUsd"`
	IsOpen                 bool                    `json:"isOpen"`
	MaintenanceMarginTiers []MaintenanceTierOutput `json:"maintenanceMarginTiers"`
	MinNotionalValue       string                  `json:"minNotionalValue"`
	MinTradeAmount         string                  `json:"minTradeAmount"`
	QuoteAsset             string                  `json:"quoteAsset"`
	SettleAsset            string                  `json:"settleAsset"`
	Symbol                 string                  `json:"symbol"`
	TickSize               string                  `json:"tickSize"`
}

type marketPriceOutput struct {
	IndexPrice string `json:"indexPrice"`
	LastPrice  string `json:"lastPrice"`
	MarkPrice  string `json:"markPrice"`
	UpdatedAt  int64  `json:"updatedAt,omitempty"`
}

type listMarketsOutput struct {
	Meta    responseMeta   `json:"_meta"`
	Markets []MarketOutput `json:"markets"`
}

type marketSymbolInput struct {
	Symbol string `json:"symbol" jsonschema:"Synthetix perpetual futures market symbol, e.g. BTC-USDT, ETH-USDT, SOL-USDT. Use list_markets to discover available symbols."`
}

type marketSummaryOutput struct {
	Meta         responseMeta      `json:"_meta"`
	FundingRate  *FundingRateEntry `json:"fundingRate,omitempty"`
	Market       MarketOutput      `json:"market"`
	OpenInterest string            `json:"openInterest"`
	Prices       marketPriceOutput `json:"prices"`
	Summary      summaryOutput     `json:"summary"`
}

type summaryOutput struct {
	BestAskPrice    string `json:"bestAskPrice"`
	BestBidPrice    string `json:"bestBidPrice"`
	LastTradedPrice string `json:"lastTradedPrice"`
	LastTradedTime  int64  `json:"lastTradedTime,omitempty"`
	PrevDayPrice    string `json:"prevDayPrice"`
	QuoteVolume24h  string `json:"quoteVolume24h"`
	Volume24h       string `json:"volume24h"`
}

type orderbookInput struct {
	Limit  int32  `json:"limit,omitempty" jsonschema:"Maximum number of bid and ask price levels per side (bids and asks). Omit for all available levels."`
	Symbol string `json:"symbol" jsonschema:"Synthetix perpetual futures market symbol, e.g. BTC-USDT. Use list_markets to discover available symbols."`
}

type priceLevelOutput struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

type orderbookOutput struct {
	Meta   responseMeta       `json:"_meta"`
	Asks   []priceLevelOutput `json:"asks"`
	Bids   []priceLevelOutput `json:"bids"`
	Symbol string             `json:"symbol"`
}

type recentTradesInput struct {
	Limit  int32  `json:"limit,omitempty" jsonschema:"Maximum number of recent trades to return. Omit for default."`
	Symbol string `json:"symbol" jsonschema:"Synthetix perpetual futures market symbol, e.g. BTC-USDT. Use list_markets to discover available symbols."`
}

type tradeOutput struct {
	Direction      string `json:"direction"`
	FillType       string `json:"fillType"`
	FilledPrice    string `json:"filledPrice"`
	FilledQuantity string `json:"filledQuantity"`
	ID             uint64 `json:"id,string"`
	Symbol         string `json:"symbol"`
	TradedAt       int64  `json:"tradedAt,omitempty"`
}

type recentTradesOutput struct {
	Meta   responseMeta  `json:"_meta"`
	Trades []tradeOutput `json:"trades"`
}

type fundingRateInput struct {
	Symbol string `json:"symbol" jsonschema:"Synthetix perpetual futures market symbol, e.g. ETH-USDT. Use list_markets to discover available symbols."`
}

type FundingRateEntry struct {
	EstimatedFundingRate string `json:"estimatedFundingRate"`
	FundingIntervalMs    int64  `json:"fundingIntervalMs"`
	LastSettlementRate   string `json:"lastSettlementRate"`
	LastSettlementTime   int64  `json:"lastSettlementTime,omitempty"`
	NextFundingTime      int64  `json:"nextFundingTime,omitempty"`
	Symbol               string `json:"symbol"`
}

type fundingRateOutput struct {
	Meta responseMeta `json:"_meta"`
	FundingRateEntry
}

type candlesInput struct {
	EndTime   int64  `json:"endTime,omitempty" jsonschema:"UTC end time in milliseconds since epoch. Omit for current time."`
	Limit     int32  `json:"limit,omitempty" jsonschema:"Maximum number of candles to return. Omit for default."`
	StartTime int64  `json:"startTime,omitempty" jsonschema:"UTC start time in milliseconds since epoch. Omit for server default."`
	Symbol    string `json:"symbol" jsonschema:"Synthetix perpetual futures market symbol, e.g. BTC-USDT."`
	Timeframe string `json:"timeframe" jsonschema:"Candlestick timeframe: 1m, 5m, 15m, 1h, 4h, or 1d."`
}

type candleOutput struct {
	ClosePrice  string `json:"closePrice"`
	CloseTime   int64  `json:"closeTime,omitempty"`
	HighPrice   string `json:"highPrice"`
	LowPrice    string `json:"lowPrice"`
	OpenPrice   string `json:"openPrice"`
	OpenTime    int64  `json:"openTime,omitempty"`
	QuoteVolume string `json:"quoteVolume"`
	Symbol      string `json:"symbol"`
	Timeframe   string `json:"timeframe"`
	TradeCount  int32  `json:"tradeCount"`
	Volume      string `json:"volume"`
}

type candlesOutput struct {
	Candles   []candleOutput `json:"candles"`
	Meta      responseMeta   `json:"_meta"`
	Symbol    string         `json:"symbol"`
	Timeframe string         `json:"timeframe"`
}

type lookupSubaccountInput struct {
	WalletAddress      string `json:"walletAddress" jsonschema:"EVM wallet address (0x-prefixed, 42 chars). Casing does not matter; the server normalises to EIP-55 checksum before lookup."`
	IncludeDelegations bool   `json:"includeDelegations,omitempty" jsonschema:"When true, also list subaccounts where this wallet is a delegate (not the owner). Useful for delegated-signer flows. Defaults to false."`
}

type subaccountSummary struct {
	OwnerAddress string `json:"ownerAddress,omitempty"`
	Relationship string `json:"relationship"`
	SubAccountID int64  `json:"subAccountId,string"`
}

type lookupSubaccountOutput struct {
	Meta             responseMeta        `json:"_meta"`
	Delegated        []subaccountSummary `json:"delegated"`
	Notes            []string            `json:"notes"`
	Owned            []subaccountSummary `json:"owned"`
	WalletAddress    string              `json:"walletAddress"`
	WalletAddressRaw string              `json:"walletAddressRaw"`
}

func RegisterPublicTools(server *mcp.Server, deps *ToolDeps) {
	addUnauthenticatedTool(server, deps, &mcp.Tool{
		Name:        "ping",
		Description: "Health-check the MCP server. Returns ok:true when the server is reachable. Call this first after connecting to confirm connectivity before using other tools.",
	}, func(_ context.Context, _ struct{}) (*mcp.CallToolResult, pingOutput, error) {
		return nil, pingOutput{
			Meta:   newResponseMeta(string(session.AuthModePublic)),
			OK:     true,
			Status: "ok",
		}, nil
	})

	addUnauthenticatedTool(server, deps, &mcp.Tool{
		Name:        "get_server_info",
		Description: "Return server identity, supported auth modes, enabled streaming channels, agent-broker availability, and disabled features. Use this after ping to discover available capabilities before trading. agentBroker.enabled=true means the server signs EIP-712 internally and you should call canonical broker tools rather than authenticate / preview_trade_signature / signed_place_order.",
	}, func(_ context.Context, _ struct{}) (*mcp.CallToolResult, serverInfoOutput, error) {
		return nil, serverInfoOutput{
			AgentBroker:      buildServerInfoAgentBroker(deps),
			AuthModes:        []string{string(session.AuthModePublic), string(session.AuthModeAuthenticated)},
			DisabledFeatures: []string{"replay", "schedule_cancel_all"},
			DocumentationResources: []string{
				"system://agent-guide",
				"system://server-info",
				"system://status",
				"system://fee-schedule",
				"system://runbooks",
				"account://risk-limits",
				"market://specs/{symbol}",
			},
			EnabledChannels:    []string{"candles", "marketPrices", "trades", "orderbook"},
			Environment:        deps.Cfg.Environment,
			Meta:               newResponseMeta(string(session.AuthModePublic)),
			ServerName:         deps.Cfg.ServerName,
			SupportsPublicMode: true,
			SupportsReplay:     false,
			Transports:         []string{"streamable-http"},
			Version:            deps.Cfg.ServerVersion,
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "list_markets",
		Description: "List available Synthetix perpetual futures markets with trading constraints including tick size, minimum trade amount, leverage tiers, and funding rate caps. Filter by status to see only open markets (default) or all markets.",
	}, func(ctx context.Context, tc ToolContext, input listMarketsInput) (*mcp.CallToolResult, listMarketsOutput, error) {
		status := input.Status
		if status == "" {
			status = "open"
		}
		if status != "open" && status != "all" {
			return newToolErrorResult(
				"INVALID_ARGUMENT",
				"status must be either open or all.",
				"Retry list_markets with status=open or status=all.",
			), listMarketsOutput{}, nil
		}

		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[listMarketsOutput](err)
		}

		marketsResp, err := restInfo.GetMarkets(ctx, status == "open")
		if err != nil {
			return toolErrorResponse[listMarketsOutput](fmt.Errorf("list markets: %w", err))
		}

		markets := make([]MarketOutput, 0, len(marketsResp))
		for i := range marketsResp {
			markets = append(markets, MapMarketFromREST(&marketsResp[i]))
		}

		return nil, listMarketsOutput{
			Meta:    newResponseMeta(authModeForState(tc.State)),
			Markets: markets,
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_market_summary",
		Description: "Return a comprehensive snapshot for one market: trading constraints, index/mark/last prices, best bid/ask, 24h volume, estimated funding rate, and total open interest. Use this before placing orders to understand current market conditions.",
	}, func(ctx context.Context, tc ToolContext, input marketSymbolInput) (*mcp.CallToolResult, marketSummaryOutput, error) {
		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[marketSummaryOutput](err)
		}

		var (
			marketResp  *types.MarketResponse
			pricesResp  map[string]types.MarketPriceResponse
			fundingResp *types.FundingRateResponse
		)

		g, gCtx := errgroup.WithContext(ctx)
		g.Go(func() error {
			var err error
			marketResp, err = restInfo.GetMarket(gCtx, input.Symbol)
			if err != nil {
				return fmt.Errorf("get market: %w", err)
			}
			return nil
		})
		g.Go(func() error {
			var err error
			pricesResp, err = restInfo.GetMarketPrices(gCtx)
			if err != nil {
				return fmt.Errorf("get market prices: %w", err)
			}
			return nil
		})
		g.Go(func() error {
			var err error
			fundingResp, err = restInfo.GetFundingRate(gCtx, input.Symbol)
			if err != nil {
				return fmt.Errorf("get funding rate: %w", err)
			}
			return nil
		})

		if err := g.Wait(); err != nil {
			return toolErrorResponse[marketSummaryOutput](err)
		}

		priceForSymbol := pricesResp[input.Symbol]
		// openInterest ships on the getMarketPrices payload, so we
		// get it for free without a separate call.
		return nil, marketSummaryOutput{
			Meta:         newResponseMeta(authModeForState(tc.State)),
			FundingRate:  MapFundingRateFromREST(fundingResp),
			Market:       MapMarketFromREST(marketResp),
			OpenInterest: priceForSymbol.OpenInterest,
			Prices:       MarketPriceFromREST(&priceForSymbol),
			Summary:      MarketSummaryFromREST(&priceForSymbol),
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_orderbook",
		Description: "Return current orderbook bid and ask levels for one market. Use limit to control depth (default returns all available levels). Inspect this before limit orders to gauge available liquidity and spread.",
	}, func(ctx context.Context, tc ToolContext, input orderbookInput) (*mcp.CallToolResult, orderbookOutput, error) {
		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[orderbookOutput](err)
		}

		resp, err := restInfo.GetOrderbook(ctx, input.Symbol, int(input.Limit))
		if err != nil {
			return toolErrorResponse[orderbookOutput](fmt.Errorf("get orderbook: %w", err))
		}

		return nil, orderbookOutput{
			Meta:   newResponseMeta(authModeForState(tc.State)),
			Asks:   MapPriceLevelsFromREST(resp.Asks),
			Bids:   MapPriceLevelsFromREST(resp.Bids),
			Symbol: resp.Symbol,
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_recent_trades",
		Description: "Return recent public trade executions for one market, including fill price, quantity, direction, and fill type. Use limit to control how many trades are returned.",
	}, func(ctx context.Context, tc ToolContext, input recentTradesInput) (*mcp.CallToolResult, recentTradesOutput, error) {
		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[recentTradesOutput](err)
		}

		resp, err := restInfo.GetLastTrades(ctx, input.Symbol, int(input.Limit), 0)
		if err != nil {
			return toolErrorResponse[recentTradesOutput](fmt.Errorf("get recent trades: %w", err))
		}

		// REST getLastTrades only carries the aggressor side, not a
		// separate fillType; derive a best-effort label that keeps
		// clients of the legacy shape from crashing on an empty
		// string.
		trades := make([]tradeOutput, 0, len(resp.Trades))
		for _, trade := range resp.Trades {
			parsedID, _ := parseTradeID(trade.TradeID)
			trades = append(trades, tradeOutput{
				Direction:      trade.Side,
				FillType:       "",
				FilledPrice:    trade.Price,
				FilledQuantity: trade.Quantity,
				ID:             parsedID,
				Symbol:         trade.Symbol,
				TradedAt:       trade.TimestampMs,
			})
		}

		return nil, recentTradesOutput{
			Meta:   newResponseMeta(authModeForState(tc.State)),
			Trades: trades,
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_funding_rate",
		Description: "Return the latest estimated funding rate, last settlement rate, funding interval, and next funding time for one market. Positive rates mean longs pay shorts; negative means shorts pay longs.",
	}, func(ctx context.Context, tc ToolContext, input fundingRateInput) (*mcp.CallToolResult, fundingRateOutput, error) {
		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[fundingRateOutput](err)
		}

		resp, err := restInfo.GetFundingRate(ctx, input.Symbol)
		if err != nil {
			return toolErrorResponse[fundingRateOutput](fmt.Errorf("get funding rate: %w", err))
		}
		entry := MapFundingRateFromREST(resp)
		if entry == nil {
			return newToolErrorResult(
				"NOT_FOUND",
				"The requested market funding rate was not found.",
				"Verify the symbol exists and retry get_funding_rate.",
			), fundingRateOutput{}, nil
		}

		return nil, fundingRateOutput{
			Meta:             newResponseMeta(authModeForState(tc.State)),
			FundingRateEntry: *entry,
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_candles",
		Description: "Return historical OHLCV candlestick data for one market. Specify timeframe (1m, 5m, 15m, 1h, 4h, 1d) and optional startTime/endTime as UTC millisecond timestamps. Includes trade count and quote volume per candle.",
	}, func(ctx context.Context, tc ToolContext, input candlesInput) (*mcp.CallToolResult, candlesOutput, error) {
		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[candlesOutput](err)
		}

		resp, err := restInfo.GetCandles(ctx, input.Symbol, input.Timeframe, int(input.Limit), input.StartTime, input.EndTime)
		if err != nil {
			return toolErrorResponse[candlesOutput](fmt.Errorf("get candles: %w", err))
		}

		candles := make([]candleOutput, 0, len(resp.Candles))
		for _, candle := range resp.Candles {
			candles = append(candles, candleOutput{
				ClosePrice:  candle.ClosePrice,
				CloseTime:   candle.CloseTime,
				HighPrice:   candle.HighPrice,
				LowPrice:    candle.LowPrice,
				OpenPrice:   candle.OpenPrice,
				OpenTime:    candle.OpenTime,
				QuoteVolume: candle.QuoteVolume,
				Symbol:      resp.Symbol,
				Timeframe:   resp.Interval,
				TradeCount:  candle.TradeCount,
				Volume:      candle.Volume,
			})
		}

		return nil, candlesOutput{
			Candles:   candles,
			Meta:      newResponseMeta(authModeForState(tc.State)),
			Symbol:    resp.Symbol,
			Timeframe: resp.Interval,
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "lookup_subaccount",
		Description: "Discover the subAccountId(s) belonging to a wallet address. Use this BEFORE preview_auth_message / authenticate when you have a private key (or wallet address) but do not yet know which subaccount to bind the MCP session to. Returns owned subaccounts and, when includeDelegations=true, also subaccounts where the wallet is a delegate (so a delegate-signer key can find the master account it can act on).",
	}, func(ctx context.Context, tc ToolContext, input lookupSubaccountInput) (*mcp.CallToolResult, lookupSubaccountOutput, error) {
		raw := strings.TrimSpace(input.WalletAddress)
		if raw == "" {
			return newToolErrorResult(
				"INVALID_ARGUMENT",
				"walletAddress is required.",
				"Pass the 0x-prefixed 42-character EVM address derived from your private key.",
			), lookupSubaccountOutput{}, nil
		}
		if !common.IsHexAddress(raw) {
			return newToolErrorResult(
				"INVALID_ARGUMENT",
				"walletAddress is not a valid EVM address.",
				"Pass a 0x-prefixed 42-character hex string. Casing does not matter — the server normalises to EIP-55.",
			), lookupSubaccountOutput{}, nil
		}
		// Normalise to EIP-55 checksum so we always hit the canonical
		// row in the subaccount store, regardless of how the agent
		// (or its private-key library) cased the address.
		checksum := common.HexToAddress(raw).Hex()

		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[lookupSubaccountOutput](err)
		}

		var (
			ownedIDs     []string
			delegatedIDs []string
		)
		if input.IncludeDelegations {
			resp, err := restInfo.GetSubAccountIdsWithDelegations(ctx, checksum)
			if err != nil {
				return toolErrorResponse[lookupSubaccountOutput](fmt.Errorf("lookup subaccounts: %w", err))
			}
			ownedIDs = resp.SubAccountIDs
			delegatedIDs = resp.DelegatedSubAccountIDs
		} else {
			ids, err := restInfo.GetSubAccountIds(ctx, checksum)
			if err != nil {
				return toolErrorResponse[lookupSubaccountOutput](fmt.Errorf("lookup subaccounts: %w", err))
			}
			ownedIDs = ids
		}

		owned := make([]subaccountSummary, 0, len(ownedIDs))
		for _, idStr := range ownedIDs {
			id, convErr := strconvParseInt(idStr)
			if convErr != nil {
				continue
			}
			owned = append(owned, subaccountSummary{
				OwnerAddress: checksum,
				Relationship: "owner",
				SubAccountID: id,
			})
		}

		// Always emit a slice (never nil) so JSON consumers can
		// iterate without a presence check; matches the convention
		// used by other public tools in this file. REST
		// getSubAccountIds does not carry the delegated owner
		// address today, so OwnerAddress is omitted (agent-facing
		// drift noted in partA-docs).
		delegated := make([]subaccountSummary, 0, len(delegatedIDs))
		for _, idStr := range delegatedIDs {
			id, convErr := strconvParseInt(idStr)
			if convErr != nil {
				continue
			}
			delegated = append(delegated, subaccountSummary{
				Relationship: "delegate",
				SubAccountID: id,
			})
		}

		notes := make([]string, 0, 2)
		if len(owned) == 0 && len(delegated) == 0 {
			notes = append(notes,
				"No subaccounts found for this wallet. Create one via the trading API "+
					"(see sample/node-scripts/scripts/create-trader-accounts.ts) or "+
					"verify the wallet address.",
			)
		} else if len(owned) > 0 {
			notes = append(notes,
				"Pass any owned subAccountId to preview_auth_message / authenticate to "+
					"bind this MCP session.",
			)
		}
		if input.IncludeDelegations && len(delegated) > 0 {
			notes = append(notes,
				"Delegated subaccounts are owned by another wallet but this wallet can "+
					"sign trade actions for them when authenticated as the delegate.",
			)
		}

		return nil, lookupSubaccountOutput{
			Meta:             newResponseMeta(authModeForState(tc.State)),
			Delegated:        delegated,
			Notes:            notes,
			Owned:            owned,
			WalletAddress:    checksum,
			WalletAddressRaw: raw,
		}, nil
	})
}

// Best-effort parse of the REST trade payload's trade_id (string)
// into the uint64 shape retained for output-wire compatibility.
// Malformed IDs land as zero.
func parseTradeID(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseUint(raw, 10, 64)
}

// strconvParseInt parses a REST-decimal subAccountId string into the
// int64 carried in subaccountSummary. The REST getSubAccountIds
// payload returns bigint-shaped strings; anything non-numeric is
// dropped by the caller rather than surfaced as a tool error.
func strconvParseInt(raw string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

// Projects broker config into the get_server_info payload. Always
// returns a populated struct so `enabled:false` is unambiguous. When
// bound, extends with the live Status() snapshot and rewrites Note
// in-place so expiry and own-key warnings are loud without needing
// a separate field.
func buildServerInfoAgentBroker(deps *ToolDeps) serverInfoAgentBroker {
	if deps == nil || deps.Cfg == nil || !deps.Cfg.AgentBroker.Enabled {
		return serverInfoAgentBroker{
			Enabled: false,
			Note: "Broker disabled. You must sign EIP-712 with a key " +
				"you already hold; never ask the human user to paste " +
				"signatures into chat. If you cannot sign locally, " +
				"refuse the trade and ask the operator to enable the " +
				"broker (SNXMCP_AGENT_BROKER_ENABLED=true).",
			BrokerTools: []string{},
		}
	}
	out := serverInfoAgentBroker{
		DefaultGuardrails: guardrailsOutputForConfig(brokerDefaultGuardrailsConfig(deps)),
		DefaultPreset:     deps.Cfg.AgentBroker.DefaultPreset,
		Enabled:           true,
		Note: "Broker enabled. Call place_order / " +
			"close_position / cancel_order / " +
			"cancel_all_orders for one-shot sign+submit. " +
			"Guardrails are optional operator limits; the standard preset " +
			"allows trading unless configured tighter. " +
			"Do not call authenticate, preview_trade_signature, or " +
			"signed_place_order unless you also hold a private key locally.",
		BrokerTools: []string{
			"place_order",
			"close_position",
			"cancel_order",
			"cancel_all_orders",
		},
	}
	if deps.BrokerStatus == nil {
		return out
	}
	status := deps.BrokerStatus.Status()
	out.ChainID = status.ChainID
	out.DelegationID = status.DelegationID
	out.ExpiresAtUnix = status.ExpiresAtUnix
	out.OwnerAddress = status.OwnerAddress
	out.Permissions = status.Permissions
	out.SubAccountID = status.SubAccountID
	out.SubaccountSource = status.SubaccountSource
	out.WalletAddress = status.WalletAddress
	if status.SubaccountSource == string(brokerSubaccountSourceOwned) {
		// Own-key posture grants every scope the auth manager
		// accepts; surface the warning here so operators see it
		// without reading the agent guide.
		out.Note += " WARNING: broker key is the registered owner of " +
			"the subaccount (subaccountSource=owned), so it can " +
			"withdraw collateral and manage delegations in addition " +
			"to trading. Prefer the delegated posture: rotate to a " +
			"dedicated trading key via " +
			"sample/node-scripts/scripts/onboard-agent-key.ts."
	}
	if status.ExpiresAtUnix > 0 {
		expiry := time.Unix(status.ExpiresAtUnix, 0)
		hours := time.Until(expiry).Hours()
		out.Note += fmt.Sprintf(
			" Delegation expires %s (~%.0f hours from now); ask the "+
				"operator to re-run onboard-agent-key.ts before then "+
				"to avoid a quiet outage.",
			expiry.UTC().Format(time.RFC3339), hours,
		)
	}
	return out
}

// Wire identifier mirroring agentbroker.SubaccountSourceOwned;
// duplicated to keep agentbroker out of this package's imports.
const brokerSubaccountSourceOwned = "owned"

// requireRESTInfo returns the RESTInfo client from deps, or an error
// shaped for surface to the agent when the service was started
// Defensive: api_base_url is required at Load(); if somehow the clients are
// absent we refuse cleanly rather than NPE.
func requireRESTInfo(deps *ToolDeps) (*restinfo.Client, error) {
	if deps == nil || deps.Clients == nil || deps.Clients.RESTInfo == nil {
		return nil, errors.New("REST info client unavailable: set SNXMCP_API_BASE_URL to the public api-service URL")
	}
	return deps.Clients.RESTInfo, nil
}
