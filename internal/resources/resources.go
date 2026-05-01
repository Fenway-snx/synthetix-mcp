package resources

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
)

//go:embed assets/agent_guide.md
var embeddedAgentGuide string

const (
	accountRiskLimitsURI = "account://risk-limits"
	agentGuideURI        = "system://agent-guide"
	feeScheduleURI       = "system://fee-schedule"
	marketSpecsTemplate  = "market://specs/{symbol}"
	routingGuideURI      = "system://routing-guide"
	runbooksURI          = "system://runbooks"
	serverInfoURI        = "system://server-info"
	statusURI            = "system://status"
)

type marketSpecsResource struct {
	FundingRate *tools.FundingRateEntry `json:"fundingRate,omitempty"`
	Market      tools.MarketOutput      `json:"market"`
}

// Wires MCP resources served by the server.
// Missing trade reads fall back to public, unhydrated shapes where possible.
func Register(server *mcp.Server, deps *tools.ToolDeps, tradeReads *tools.TradeReadClient) {
	cfg := deps.Cfg
	clients := deps.Clients
	store := deps.Store
	verifier := deps.Verifier
	// Precompute startup-immutable guide content.
	guideText := agentGuideContents(cfg)
	server.AddResource(&mcp.Resource{
		Description: "High-level operating guide for the Synthetix MCP server.",
		MIMEType:    "text/markdown",
		Name:        "agent_guide",
		Title:       "Agent Guide",
		URI:         agentGuideURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := sessionStateForRead(ctx, store, req, verifier); err != nil {
			return nil, err
		}
		return textResourceResult(agentGuideURI, "text/markdown", guideText), nil
	})

	server.AddResource(&mcp.Resource{
		Description: "Current MCP server identity, limits, and public capabilities.",
		MIMEType:    "application/json",
		Name:        "server_info",
		Title:       "Server Info",
		URI:         serverInfoURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := sessionStateForRead(ctx, store, req, verifier); err != nil {
			return nil, err
		}
		body, err := json.MarshalIndent(map[string]any{
			"name":                       cfg.ServerName,
			"version":                    cfg.ServerVersion,
			"environment":                cfg.Environment,
			"maxSubscriptionsPerSession": cfg.MaxSubscriptionsPerSession,
			"rateLimitGuidance":          rateLimitGuidanceResource(),
			"routingGuide":               routingGuideResource(cfg.AgentBroker.Enabled),
			"sessionStore":               sessionStoreResource(cfg.SessionStorePath),
			"sessionTtlSeconds":          int64(cfg.SessionTTL.Seconds()),
			"delegationCompatibility": map[string]any{
				"activeWriteMode":             routingMode(cfg.AgentBroker.Enabled),
				"delegationCompatibleTools":   delegationCompatibleTools(cfg.AgentBroker.Enabled),
				"externalWalletToolsExposed":  !cfg.AgentBroker.Enabled,
				"selfHostedBrokerToolsActive": cfg.AgentBroker.Enabled,
				"ownerOnlyToolsExcludedFromPhase1": []string{
					"create_subaccount",
					"signed_withdraw_collateral",
					"signed_transfer_collateral",
					"update_subaccount_name",
					"signed_remove_all_delegated_signers",
				},
				"notes": []string{
					"Read system://routing-guide before choosing a write path. Broker mode hides signed_* write tools to avoid dual-path confusion.",
				},
			},
		}, "", "  ")
		if err != nil {
			return nil, err
		}
		return textResourceResult(serverInfoURI, "application/json", string(body)), nil
	})

	server.AddResource(&mcp.Resource{
		Description: "Machine-readable routing guide for choosing broker vs external-wallet tools.",
		MIMEType:    "application/json",
		Name:        "routing_guide",
		Title:       "Routing Guide",
		URI:         routingGuideURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := sessionStateForRead(ctx, store, req, verifier); err != nil {
			return nil, err
		}
		body, err := json.MarshalIndent(routingGuideResource(cfg.AgentBroker.Enabled), "", "  ")
		if err != nil {
			return nil, err
		}
		return textResourceResult(routingGuideURI, "application/json", string(body)), nil
	})

	server.AddResource(&mcp.Resource{
		Description: "Public server liveness flag. Returns 'running' when the MCP server can accept requests and 'not_running' otherwise. Intentionally opaque: no dependency detail is exposed to clients.",
		MIMEType:    "application/json",
		Name:        "status",
		Title:       "Server Status",
		URI:         statusURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := sessionStateForRead(ctx, store, req, verifier); err != nil {
			return nil, err
		}
		state := "running"
		if err := clients.Ready(ctx); err != nil {
			state = "not_running"
		}
		body, err := json.MarshalIndent(map[string]any{
			"status": state,
		}, "", "  ")
		if err != nil {
			return nil, err
		}
		return textResourceResult(statusURI, "application/json", string(body)), nil
	})

	server.AddResource(&mcp.Resource{
		Description: "Current fee schedule for the authenticated subaccount when available.",
		MIMEType:    "application/json",
		Name:        "fee_schedule",
		Title:       "Fee Schedule",
		URI:         feeScheduleURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		state, err := sessionStateForRead(ctx, store, req, verifier)
		if err != nil {
			return nil, err
		}

		payload := map[string]any{
			"authenticated": state != nil && state.SubAccountID > 0,
			"note":          "Authenticate first to hydrate subaccount-specific maker and taker fee rates.",
		}
		if state != nil && state.SubAccountID > 0 {
			// Fee rates require broker-signed reads today.
			// Wallet-authenticated sessions degrade to unhydrated output.
			if tradeReads != nil {
				tc := tools.ToolContext{State: state}
				sub, err := tradeReads.GetSubAccount(ctx, tc)
				switch {
				case err == nil && sub != nil:
					payload["subAccountId"] = sub.SubAccountID
					payload["tierName"] = sub.FeeRates.TierName
					payload["makerFeeRate"] = sub.FeeRates.MakerFeeRate
					payload["takerFeeRate"] = sub.FeeRates.TakerFeeRate
				case errors.Is(err, tools.ErrReadUnavailable), errors.Is(err, tools.ErrBrokerSubAccountMismatch):
					// Leave payload unhydrated; note already explains
					// remediation and a narrower diagnostic belongs in
					// the status resource rather than here.
				case err != nil:
					return nil, sanitizeResourceBackendError("fee schedule is temporarily unavailable")
				}
			}
		}

		body, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, err
		}
		return textResourceResult(feeScheduleURI, "application/json", string(body)), nil
	})

	server.AddResource(&mcp.Resource{
		Description: "Session-level risk and operational constraints for the current connection.",
		MIMEType:    "application/json",
		Name:        "risk_limits",
		Title:       "Risk Limits",
		URI:         accountRiskLimitsURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		state, err := sessionStateForRead(ctx, store, req, verifier)
		if err != nil {
			return nil, err
		}

		body, err := json.MarshalIndent(map[string]any{
			"authenticated":              state != nil && state.SubAccountID > 0,
			"authMode":                   authMode(state),
			"maxSubscriptionsPerSession": cfg.MaxSubscriptionsPerSession,
			"rateLimitGuidance":          rateLimitGuidanceResource(),
			"sessionId":                  sessionIDFromRead(req),
			"sessionStore":               sessionStoreResource(cfg.SessionStorePath),
			"sessionTtlSeconds":          int64(cfg.SessionTTL.Seconds()),
			"subAccountId":               subaccountID(state),
			"walletAddress":              walletAddress(state),
		}, "", "  ")
		if err != nil {
			return nil, err
		}
		return textResourceResult(accountRiskLimitsURI, "application/json", string(body)), nil
	})

	server.AddResource(&mcp.Resource{
		Description: "Operational runbooks for common MCP workflows and failure modes.",
		MIMEType:    "text/markdown",
		Name:        "runbooks",
		Title:       "Runbooks",
		URI:         runbooksURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := sessionStateForRead(ctx, store, req, verifier); err != nil {
			return nil, err
		}
		return textResourceResult(runbooksURI, "text/markdown", runbookContents()), nil
	})

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Description: "Market specification sheet for one Synthetix perp symbol.",
		MIMEType:    "application/json",
		Name:        "market_specs",
		Title:       "Market Specs",
		URITemplate: marketSpecsTemplate,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := sessionStateForRead(ctx, store, req, verifier); err != nil {
			return nil, err
		}
		symbol := req.Params.URI[len("market://specs/"):]
		if symbol == "" {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}

		if clients == nil || clients.RESTInfo == nil {
			return nil, sanitizeResourceBackendError("market specs are temporarily unavailable")
		}

		market, err := clients.RESTInfo.GetMarket(ctx, symbol)
		if err != nil {
			return nil, sanitizeResourceBackendError("market specs are temporarily unavailable")
		}
		// Market specs remain valid without best-effort funding metadata.
		funding, err := clients.RESTInfo.GetFundingRate(ctx, symbol)
		if err != nil {
			funding = nil
		}

		body, err := json.MarshalIndent(marketSpecsResource{
			Market:      tools.MapMarketFromREST(market),
			FundingRate: tools.MapFundingRateFromREST(funding),
		}, "", "  ")
		if err != nil {
			return nil, err
		}
		return textResourceResult(req.Params.URI, "application/json", string(body)), nil
	})

	registerTradeJournal(server, deps, tradeReads)
}

func routingMode(brokerEnabled bool) string {
	if brokerEnabled {
		return "broker"
	}
	return "external_wallet"
}

func sessionStoreResource(path string) map[string]any {
	if path == "" {
		return map[string]any{
			"type":       "memory",
			"persistent": false,
			"note":       "sessions are lost on server restart",
		}
	}
	return map[string]any{
		"type":       "file",
		"path":       path,
		"persistent": true,
		"note":       "authenticated session bindings and guardrails survive server restarts; private keys, signatures, and request nonces are not stored",
	}
}

func delegationCompatibleTools(brokerEnabled bool) []string {
	if brokerEnabled {
		return []string{
			"preview_order",
			"place_order",
			"modify_order",
			"cancel_order",
			"cancel_all_orders",
			"close_position",
		}
	}
	return []string{
		"preview_order",
		"preview_trade_signature",
		"signed_place_order",
		"signed_modify_order",
		"signed_cancel_order",
		"signed_cancel_all_orders",
		"signed_close_position",
	}
}

func routingGuideResource(brokerEnabled bool) map[string]any {
	if brokerEnabled {
		return map[string]any{
			"brokerEnabled": true,
			"mode":          "broker",
			"firstTool":     "ping",
			"useTools": []string{
				"preview_order",
				"place_order",
				"modify_order",
				"cancel_order",
				"cancel_all_orders",
				"close_position",
				"update_leverage",
				"withdraw_collateral",
				"transfer_collateral",
				"arm_dead_man_switch",
				"disarm_dead_man_switch",
			},
			"doNotUse": []string{
				"preview_trade_signature",
				"signed_*",
			},
			"notes": []string{
				"Broker mode is active. Use canonical broker tools; they authenticate, apply guardrails, sign EIP-712 payloads server-side, and submit in one call.",
				"Do not use signed_* tools or preview_trade_signature in broker mode.",
			},
		}
	}
	return map[string]any{
		"brokerEnabled": false,
		"mode":          "external_wallet",
		"firstTool":     "ping",
		"useTools": []string{
			"preview_auth_message",
			"authenticate",
			"preview_order",
			"preview_trade_signature",
			"signed_place_order",
			"signed_modify_order",
			"signed_cancel_order",
			"signed_cancel_all_orders",
			"signed_close_position",
		},
		"doNotUse": []string{
			"place_order",
			"modify_order",
			"cancel_order",
			"cancel_all_orders",
			"close_position",
		},
		"notes": []string{
			"Broker mode is disabled. Claude cannot sign EIP-712 payloads by itself; use sample/node-scripts/authenticate-external-wallet.mjs from a terminal to authenticate this MCP session.",
			"Use signed_* tools only with a local sidecar signer or SDK wrapper that holds the key outside chat.",
			"Never ask a human to paste a private key, digest, EIP-712 payload, or signature into chat.",
		},
	}
}

func textResourceResult(uri, mimeType, body string) *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: mimeType,
			Text:     body,
		}},
	}
}

func sanitizeResourceBackendError(message string) error {
	return errors.New(message)
}

func sessionStateForRead(
	ctx context.Context,
	store session.Store,
	req *mcp.ReadResourceRequest,
	verifier tools.SessionAccessVerifier,
) (*session.State, error) {
	return sessionStateForID(ctx, store, sessionIDFromRead(req), verifier)
}

func sessionStateForID(
	ctx context.Context,
	store session.Store,
	sessionID string,
	verifier tools.SessionAccessVerifier,
) (*session.State, error) {
	if store == nil || sessionID == "" {
		return nil, nil
	}

	state, err := store.Get(ctx, sessionID)
	if errors.Is(err, session.ErrSessionNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if state != nil && state.AuthMode == session.AuthModeAuthenticated && state.SubAccountID > 0 && verifier != nil {
		if err := verifier.VerifySessionAccess(ctx, state.WalletAddress, state.SubAccountID); err != nil {
			_, delErr := store.DeleteIfExists(ctx, sessionID)
			if delErr != nil {
				return nil, errors.Join(err, fmt.Errorf("clear revoked session: %w", delErr))
			}
			return nil, nil
		}
	}
	return state, nil
}

func sessionIDFromRead(req *mcp.ReadResourceRequest) string {
	if req == nil || req.Session == nil {
		return ""
	}
	return req.Session.ID()
}

func authMode(state *session.State) string {
	if state == nil || state.AuthMode == "" {
		return string(session.AuthModePublic)
	}
	return string(state.AuthMode)
}

// Returns the authenticated subaccount ID as a decimal string.
// String form preserves precision for JavaScript parsers.
func subaccountID(state *session.State) string {
	if state == nil || state.SubAccountID == 0 {
		return ""
	}
	return strconv.FormatInt(state.SubAccountID, 10)
}

func walletAddress(state *session.State) string {
	if state == nil {
		return ""
	}
	return state.WalletAddress
}

func rateLimitGuidanceResource() map[string]any {
	return map[string]any{
		"enforcedBy": "upstream Synthetix API and client-side callers; this MCP does not enforce local request quotas",
		"retryPlan": []string{
			"Back off with jitter after RATE_LIMITED or upstream 429-style errors.",
			"For write tools, confirm order/session state before replaying a request.",
			"Prefer consolidated reads over tight polling loops.",
		},
	}
}

func agentGuideContents(cfg *config.Config) string {
	guide := embeddedAgentGuide
	if cfg != nil {
		guide += fmt.Sprintf("\nCurrent server: %s %s in %s.\n", cfg.ServerName, cfg.ServerVersion, cfg.Environment)
	}
	return guide
}

func runbookContents() string {
	return `# MCP Runbooks

> **Pick your signing path first.** Call get_server_info once per session and inspect agentBroker.enabled. If true, follow the "Self-hosted broker path" sections below; the broker auto-authenticates and signs server-side, so you should NOT call authenticate, preview_auth_message, preview_trade_signature, or any signed_* write tool. Guardrails are optional operator limits; call set_guardrails only to tighten a session or switch it to read_only. If broker is false, you must hold a private key locally and follow the "Wallet path" sections. Never ask a human user to paste an EIP-712 signature into chat.

## Session bootstrap

### Self-hosted broker path (broker enabled)
1. Call get_context for a consolidated snapshot of server, session, markets, and account. The first broker write you make also auto-authenticates the session against the broker's wallet.
2. Confirm capabilities.agentBroker.enabled = true and capabilities.recommendedFlow lists the canonical broker tools.

### Wallet path (broker disabled)
1. Call get_context for a consolidated snapshot of server, session, markets, and account.
2. If not yet authenticated, do not ask the user to paste a signature. Ask the user to run sample/node-scripts/authenticate-external-wallet.mjs in a terminal with this MCP session ID; the sidecar calls preview_auth_message, signs locally, and calls authenticate.
3. Optionally call set_guardrails if the operator wants tighter per-session limits or read_only mode.
4. Call get_context again to confirm authMode='authenticated' and review account margin.

## Market analysis (signing-path agnostic)
1. Call get_market_summary for the target symbol.
2. Call get_orderbook to assess liquidity and spread.
3. Call get_recent_trades to observe trade flow direction.
4. Call get_funding_rate to understand funding cost/benefit.
5. Read market://specs/{symbol} for contract specifications.

## Pre-trade preparation

### Self-hosted broker path (broker enabled)
1. Call get_account_summary to check margin capacity.
2. Call get_positions to understand existing exposure.
3. Call get_open_orders to check for pending orders on the same market.
4. Ask for confirmation at most once for this operation, combining order details, account context, and guardrails.
5. Call place_order with {symbol, side, type, quantity, price?, clientOrderId}. The broker validates configured guardrails, signs the placeOrders action, and submits in one round trip.
6. Inspect the response's phase and followUp fields. phase='ACCEPTED' is a successful resting order; phase='PENDING_CONFIRMATION' means poll get_open_orders / get_order_history with the returned clientOrderId; phase='REJECTED' carries errorCode and errorDetail.

### Wallet path (broker disabled)
1. Call get_account_summary to check margin capacity.
2. Call get_positions to understand existing exposure.
3. Call get_open_orders to check for pending orders on the same market.
4. Call preview_order to validate order shape before submission.
5. Call preview_trade_signature with action='placeOrders' to get the EIP-712 typed-data plus server-generated nonce and expiresAfter.
6. Sign typedData, split signature into {r, s, v}, and call signed_place_order with the echoed nonce and expiresAfter.

## Position unwind

### Self-hosted broker path (broker enabled)
- To close a position: close_position with {symbol, quantity?, method?='market', limitPrice?}. Reduce-only is enforced server-side; omit quantity for a full close.
- To cancel one open order: cancel_order with venueOrderId or clientOrderId.
- To cancel all open orders (optionally per symbol): cancel_all_orders.
- There is no broker equivalent for signed_modify_order today; cancel and re-place instead.

### Wallet path (broker disabled)
- Use preview_trade_signature with action='closePosition' / 'cancelOrders' / 'cancelAllOrders' / 'modifyOrder', sign locally, then call the matching signed_* write tool with the echoed nonce/expiresAfter and {r,s,v} signature.

## Performance review

1. Read account://trade-journal to get the last 14 days of fills aggregated into a daily PnL bar chart, win-rate stats, per-symbol breakdown, and a list of recent closed trades. The resource is authenticated-only and cheap to re-read; it issues one upstream getTrades call per read and returns a markdown body with a compact card at the top.
2. For deeper analysis, fan out to get_trade_history (raw fills, paginated) and get_funding_payments (funding-only series).
3. Use account://trade-journal as the canonical "how have I traded recently?" answer; do not poll it on a tight loop.

## Session refresh
1. Call restore_session to extend the current MCP session state when a client preserves the same MCP session context.
2. Open a new connection and call authenticate (wallet path) or any broker write tool (broker path) again if the client cannot preserve the current MCP session ID.
3. Call get_session to confirm the current authenticated identity and active subscriptions.

## Failure handling
1. If a tool returns TIMEOUT or BACKEND_UNAVAILABLE, check system://status before retrying.
2. If an authenticated call fails with AUTH_REQUIRED, re-authenticate (wallet path) or re-issue the broker write so the broker re-authenticates the session (broker path).
3. If RATE_LIMITED, back off with jitter. For write tools, confirm state with get_open_orders / get_order_history before replaying.
4. If delegation changes are suspected, re-authenticate to refresh session authority. On the broker path this happens automatically on the next broker write.
`
}
