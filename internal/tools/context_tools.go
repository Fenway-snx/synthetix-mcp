package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/synthetixio/synthetix-go/types"
)

type contextOutput struct {
	Meta         responseMeta        `json:"_meta"`
	Account      *contextAccount     `json:"account,omitempty"`
	Capabilities contextCapabilities `json:"capabilities"`
	Markets      []contextMarket     `json:"markets"`
	Session      contextSession      `json:"session"`
	Server       contextServer       `json:"server"`
}

type contextServer struct {
	Environment string `json:"environment"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Version     string `json:"version"`
}

// Tells the agent what the server can do for it without making it
// read system://agent-guide. `agentBroker` is the "you don't need to
// sign EIP-712 yourself" signal; `recommendedFlow` is the explicit
// playbook so the agent doesn't derive one by trial and error.
type contextCapabilities struct {
	AgentBroker     contextAgentBroker `json:"agentBroker"`
	RecommendedFlow []string           `json:"recommendedFlow"`
	SigningPolicy   string             `json:"signingPolicy"`
}

type contextAgentBroker struct {
	DefaultGuardrails *guardrailsOutput `json:"defaultGuardrails,omitempty"`
	DefaultPreset     string            `json:"defaultPreset,omitempty"`
	Enabled           bool              `json:"enabled"`
	Note              string            `json:"note"`
	BrokerTools       []string          `json:"brokerTools"`
}

type contextSession struct {
	ActiveSubscriptions []string `json:"activeSubscriptions"`
	Authenticated       bool     `json:"authenticated"`
	AuthMode            string   `json:"authMode"`
	SessionID           string   `json:"sessionId"`
	SubAccountID        int64    `json:"subAccountId,omitempty,string"`
	WalletAddress       string   `json:"walletAddress,omitempty"`
}

type contextMarket struct {
	IsOpen    bool   `json:"isOpen"`
	MarkPrice string `json:"markPrice,omitempty"`
	Symbol    string `json:"symbol"`
}

type contextAccount struct {
	AvailableMargin string `json:"availableMargin"`
	AccountValue    string `json:"accountValue"`
	OpenOrderCount  int    `json:"openOrderCount"`
	PositionCount   int    `json:"positionCount"`
	UnrealizedPnl   string `json:"unrealizedPnl"`
}

// RegisterContextTools wires the get_context tool. tradeReads is the
// shim used to pull per-subaccount summary data from /v1/trade when
// the session is authenticated. It may be nil during the REST
// migration (or in tests that don't need the account block); in that
// case the account block stays empty and the rest of the snapshot
// still renders.
func RegisterContextTools(
	server *mcp.Server,
	deps *ToolDeps,
	subscriptions SessionSubscriptionReader,
	tradeReads *TradeReadClient,
) {
	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_context",
		Description: "Return a consolidated snapshot of the current trading context: server status, session state, available markets with mark prices, and (if authenticated) account margin summary. Use this as an efficient single call to orient before making decisions.",
	}, func(ctx context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, contextOutput, error) {
		sessionID := tc.SessionID
		state := tc.State

		authMode := authModeForState(state)

		serverStatus := "ready"
		if err := deps.Clients.Ready(ctx); err != nil {
			serverStatus = "degraded"
		}

		output := contextOutput{
			Meta: newResponseMeta(authMode),
			Capabilities: contextCapabilitiesFromFlags(
				deps.Cfg.AgentBroker.Enabled,
				brokerDefaultGuardrailsConfig(deps),
			),
			Server: contextServer{
				Environment: deps.Cfg.Environment,
				Name:        deps.Cfg.ServerName,
				Status:      serverStatus,
				Version:     deps.Cfg.ServerVersion,
			},
			Session: contextSession{
				ActiveSubscriptions: activeSubscriptions(subscriptions, sessionID),
				Authenticated:       state != nil && state.AuthMode == session.AuthModeAuthenticated,
				AuthMode:            authMode,
				SessionID:           sessionID,
			},
		}

		if state != nil {
			output.Session.SubAccountID = state.SubAccountID
			output.Session.WalletAddress = state.WalletAddress
		}

		restInfo, err := requireRESTInfo(deps)
		if err != nil {
			return toolErrorResponse[contextOutput](err)
		}

		var (
			marketsResp []types.MarketResponse
			pricesResp  map[string]types.MarketPriceResponse
		)

		g, gCtx := errgroup.WithContext(ctx)

		g.Go(func() error {
			var err error
			marketsResp, err = restInfo.GetMarkets(gCtx, true)
			return err
		})

		g.Go(func() error {
			var err error
			pricesResp, err = restInfo.GetMarketPrices(gCtx)
			return err
		})

		// Authenticated account block. Only the broker-read path is
		// wired today; wallet-authenticated sessions (or broker-off
		// configurations) deliberately leave Account=nil rather than
		// surfacing a per-call error on what is otherwise a
		// public-safe tool.
		var (
			subAccountData *types.SubAccountResponse
			openOrdersData []types.OpenOrder
			accountLookup  = tradeReads != nil && state != nil && state.AuthMode == session.AuthModeAuthenticated
		)
		if accountLookup {
			g.Go(func() error {
				res, accErr := tradeReads.GetSubAccount(gCtx, tc)
				if accErr != nil {
					if errors.Is(accErr, ErrReadUnavailable) || errors.Is(accErr, ErrBrokerSubAccountMismatch) {
						return nil
					}
					return accErr
				}
				subAccountData = res
				return nil
			})
			g.Go(func() error {
				res, oErr := tradeReads.GetOpenOrders(gCtx, tc)
				if oErr != nil {
					if errors.Is(oErr, ErrReadUnavailable) || errors.Is(oErr, ErrBrokerSubAccountMismatch) {
						return nil
					}
					return oErr
				}
				openOrdersData = res
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return toolErrorResponse[contextOutput](fmt.Errorf("get context: %w", err))
		}

		if accountLookup && subAccountData != nil {
			output.Account = &contextAccount{
				AccountValue:    subAccountData.MarginSummary.AccountValue,
				AvailableMargin: subAccountData.MarginSummary.AvailableMargin,
				OpenOrderCount:  len(openOrdersData),
				PositionCount:   len(subAccountData.Positions),
				UnrealizedPnl:   subAccountData.MarginSummary.UnrealizedPnl,
			}
		}

		markets := make([]contextMarket, 0, len(marketsResp))
		for i := range marketsResp {
			m := marketsResp[i]
			markPrice := ""
			if p, ok := pricesResp[m.Symbol]; ok {
				markPrice = p.MarkPrice
			}
			markets = append(markets, contextMarket{
				IsOpen:    m.IsOpen,
				MarkPrice: markPrice,
				Symbol:    m.Symbol,
			})
		}
		output.Markets = markets

		return nil, output, nil
	})
}

// Renders the "what should I call next" block embedded in
// get_context. A single machine-readable nudge so the agent never
// resorts to asking the user to paste signatures.
func contextCapabilitiesFromFlags(
	brokerEnabled bool,
	defaultGuardrails *guardrails.Config,
) contextCapabilities {
	caps := contextCapabilities{
		AgentBroker: contextAgentBroker{
			Enabled: brokerEnabled,
			// BrokerTools is intentionally always a (possibly empty)
			// slice so JSON consumers can iterate without nil checks.
			BrokerTools: []string{},
		},
	}
	if brokerEnabled {
		defaultGuardrailsOut := guardrailsOutputForConfig(defaultGuardrails)
		defaultPreset := ""
		if defaultGuardrailsOut != nil {
			defaultPreset = defaultGuardrailsOut.EffectivePreset
		}
		caps.AgentBroker.DefaultPreset = defaultPreset
		caps.AgentBroker.DefaultGuardrails = defaultGuardrailsOut
		caps.AgentBroker.BrokerTools = []string{
			"place_order",
			"close_position",
			"cancel_order",
			"cancel_all_orders",
		}
		caps.AgentBroker.Note = "Broker holds the trading key " +
			"server-side. Call canonical broker tools for one-shot " +
			"sign+submit; guardrails are optional operator limits, not " +
			"a prerequisite. You do not need to call authenticate, " +
			"set_guardrails, preview_trade_signature, or signed_place_order."
		caps.SigningPolicy = "broker"
		caps.RecommendedFlow = []string{
			"1. Call get_market_summary (and get_orderbook for limit orders) on your target symbol.",
			"2. Include the broker default or active session guardrails in the single confirmation for the operation if the user has not already confirmed it.",
			"3. Call place_order with {symbol, side, type, quantity, price?, clientOrderId}. " +
				"It will auto-authenticate, apply the broker's default guardrails (preset='" + defaultPresetOrFallback(defaultPreset) + "'), sign with the broker's key, and submit in one call.",
			"4. Inspect the returned phase and followUp; call get_open_orders / get_order_history with the returned clientOrderId to confirm the final state.",
			"5. To unwind: call close_position (reduce-only) or cancel_order — never simulate signing in chat.",
		}
	} else {
		caps.AgentBroker.Note = "Broker disabled. To trade you must " +
			"sign EIP-712 locally with your own key. Never ask the " +
			"user to paste signatures into chat — if you cannot sign " +
			"locally, ask the operator to enable the broker " +
			"(SNXMCP_AGENT_BROKER_ENABLED=true) or run a wrapper " +
			"from sample/node-scripts."
		caps.SigningPolicy = "client"
		caps.RecommendedFlow = []string{
			"1. Call lookup_subaccount with your wallet address to discover its subaccountId(s).",
			"2. Call preview_auth_message and sign the returned typedData with your local key (viem.signTypedData / eth_signTypedData_v4 / ethers Wallet._signTypedData / Web3.py sign_typed_data).",
			"3. Call authenticate with the serialized typedData and 0x-prefixed signature.",
			"4. Optionally call set_guardrails if the operator wants tighter per-session limits or read_only mode.",
			"5. For each trade: preview_trade_signature → sign locally → signed_place_order (or signed_modify_order / signed_cancel_order / signed_cancel_all_orders / signed_close_position) with the echoed nonce + expiresAfter + split signature.",
			"6. Ask for confirmation at most once per trade or operation; combine order details, account context, and guardrails in that single prompt.",
			"7. If you have no local signer, do NOT prompt the user — refuse the trade and explain that the operator must enable the broker or run sample/node-scripts.",
		}
	}
	return caps
}

func defaultPresetOrFallback(preset string) string {
	if preset == "" {
		return "standard"
	}
	return preset
}
