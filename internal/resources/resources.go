package resources

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/metrics"
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
	runbooksURI          = "system://runbooks"
	serverInfoURI        = "system://server-info"
	statusURI            = "system://status"
)

type marketSpecsResource struct {
	FundingRate *tools.FundingRateEntry `json:"fundingRate,omitempty"`
	Market      tools.MarketOutput      `json:"market"`
}

// Register wires the MCP resources served by the server. tradeReads
// is the shim used to hydrate /v1/trade-backed payloads (fee schedule
// today; additional authenticated resources later). Pass nil when
// the service boots without REST trade access — those resources fall
// back to their public/unhydrated shape instead of erroring.
func Register(server *mcp.Server, deps *tools.ToolDeps, tradeReads *tools.TradeReadClient) {
	cfg := deps.Cfg
	clients := deps.Clients
	store := deps.Store
	verifier := deps.Verifier
	limiter := deps.Limiter
	// agent_guide and the per-IP rate-limit sub-map are fully determined by
	// startup-immutable config, so precompute them once here instead of
	// rebuilding per read. risk_limits still builds its full map per request
	// because its per-sub-account branch depends on session state.
	guideText := agentGuideContents(cfg)
	server.AddResource(&mcp.Resource{
		Description: "High-level operating guide for the Synthetix MCP server.",
		MIMEType:    "text/markdown",
		Name:        "agent_guide",
		Title:       "Agent Guide",
		URI:         agentGuideURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := maybeRateLimitRead(ctx, store, req, verifier, limiter, "read_agent_guide"); err != nil {
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
		if _, err := maybeRateLimitRead(ctx, store, req, verifier, limiter, "read_server_info"); err != nil {
			return nil, err
		}
		body, err := json.MarshalIndent(map[string]any{
			"name":                       cfg.ServerName,
			"version":                    cfg.ServerVersion,
			"environment":                cfg.Environment,
			"maxSubscriptionsPerSession": cfg.MaxSubscriptionsPerSession,
			"publicRpsPerIp":             cfg.PublicRPSPerIP,
			"authRpsPerSubAccount":       cfg.AuthRPSPerSubAccount,
			"rateLimiting":               rateLimitResource(cfg, nil),
			"sessionTtlSeconds":          int64(cfg.SessionTTL.Seconds()),
			"delegationCompatibility": map[string]any{
				"delegationCompatibleTools": []string{
					"preview_order",
					"place_order",
					"modify_order",
					"cancel_order",
					"cancel_all_orders",
					"close_position",
				},
				"ownerOnlyToolsExcludedFromPhase1": []string{
					"create_subaccount",
					"withdraw_collateral",
					"transfer_collateral",
					"update_subaccount_name",
					"remove_all_delegated_signers",
				},
				"notes": []string{
					"schedule_cancel_all is listed in disabledFeatures and is not registered as a tool — call cancel_all_orders / quick_cancel_all_orders manually until backend support lands.",
				},
			},
		}, "", "  ")
		if err != nil {
			return nil, err
		}
		return textResourceResult(serverInfoURI, "application/json", string(body)), nil
	})

	server.AddResource(&mcp.Resource{
		Description: "Public server liveness flag. Returns 'running' when the MCP server can accept requests and 'not_running' otherwise. Intentionally opaque: no dependency detail is exposed to clients.",
		MIMEType:    "application/json",
		Name:        "status",
		Title:       "Server Status",
		URI:         statusURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if _, err := maybeRateLimitRead(ctx, store, req, verifier, limiter, "read_status"); err != nil {
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
		state, err := maybeRateLimitRead(ctx, store, req, verifier, limiter, "read_fee_schedule")
		if err != nil {
			return nil, err
		}

		payload := map[string]any{
			"authenticated": state != nil && state.SubAccountID > 0,
			"note":          "Authenticate first to hydrate subaccount-specific maker and taker fee rates.",
		}
		if state != nil && state.SubAccountID > 0 {
			// Fee rates live on the authenticated /v1/trade
			// getSubAccount response. We can only reach that endpoint
			// through the broker-signed read path today; a
			// wallet-authenticated session with no in-process broker
			// deliberately degrades to the public (authenticated=true
			// but unhydrated) payload instead of failing the read.
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
		state, err := maybeRateLimitRead(ctx, store, req, verifier, limiter, "read_risk_limits")
		if err != nil {
			return nil, err
		}

		body, err := json.MarshalIndent(map[string]any{
			"authenticated":              state != nil && state.SubAccountID > 0,
			"authMode":                   authMode(state),
			"maxSubscriptionsPerSession": cfg.MaxSubscriptionsPerSession,
			"publicRpsPerIp":             cfg.PublicRPSPerIP,
			"authRpsPerSubAccount":       cfg.AuthRPSPerSubAccount,
			"rateLimiting":               rateLimitResource(cfg, state),
			"sessionId":                  sessionIDFromRead(req),
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
		if _, err := maybeRateLimitRead(ctx, store, req, verifier, limiter, "read_runbooks"); err != nil {
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
		if _, err := maybeRateLimitRead(ctx, store, req, verifier, limiter, "read_market_specs"); err != nil {
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
		// Funding rate is best-effort metadata on this resource;
		// markets are valid even when funding hasn't settled yet, so
		// surface a nil funding rate rather than failing the whole
		// read.
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
			deleted, delErr := store.DeleteIfExists(ctx, sessionID)
			if delErr != nil {
				return nil, errors.Join(err, fmt.Errorf("clear revoked session: %w", delErr))
			}
			if deleted {
				metrics.ActiveSessions().Dec()
			}
			return nil, nil
		}
	}
	return state, nil
}

func maybeRateLimitRead(
	ctx context.Context,
	store session.Store,
	req *mcp.ReadResourceRequest,
	verifier tools.SessionAccessVerifier,
	limiter tools.ToolRateLimiter,
	operationName string,
) (*session.State, error) {
	state, err := sessionStateForRead(ctx, store, req, verifier)
	if err != nil {
		return nil, err
	}
	if err := tools.MaybeRateLimitOperation(ctx, limiter, state, operationName, 1); err != nil {
		return nil, err
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

// subaccountID returns the authenticated subaccount ID as a decimal string
// so downstream JSON parsers that bucket numbers as IEEE-754 doubles
// (notably JavaScript) can't round the low-order digits. Synthetix
// composite subaccount IDs can exceed 2^53, which is Number.MAX_SAFE_INTEGER
// in JS. An empty string is returned for unauthenticated sessions so the
// field is still present with a stable shape.
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

func rateLimitResource(cfg *config.Config, state *session.State) map[string]any {
	effectiveSubaccountLimit := effectiveSubaccountRateLimit(cfg, state)
	return map[string]any{
		"perIP": map[string]any{
			"requestsPerSecond": cfg.PublicRPSPerIP,
			"tokenCosts":        tools.CopyTokenCosts(cfg.IPHandlerTokenCosts),
			"tokensPerWindow":   int64(cfg.IPRateLimiterConfig.IPRateLimit),
			"windowMs":          cfg.IPRateLimiterConfig.WindowMs,
		},
		"perSubAccount": map[string]any{
			"hasSpecificOverridesConfigured": len(cfg.OrderRateLimiterConfig.SpecificRateLimits) > 0,
			"requestsPerSecond":              config.DerivedTokensPerSecond(effectiveSubaccountLimit, cfg.OrderRateLimiterConfig.WindowMs),
			"tokenCosts":                     tools.CopyTokenCosts(cfg.HandlerTokenCosts),
			"tokensPerWindow":                int64(effectiveSubaccountLimit),
			"windowMs":                       cfg.OrderRateLimiterConfig.WindowMs,
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

func effectiveSubaccountRateLimit(cfg *config.Config, state *session.State) int64 {
	if cfg == nil {
		return 0
	}
	if state != nil && state.SubAccountID > 0 {
		if limit, ok := cfg.OrderRateLimiterConfig.SpecificRateLimits[snx_lib_core.SubAccountId(state.SubAccountID)]; ok {
			return int64(limit)
		}
	}
	return int64(cfg.OrderRateLimiterConfig.GeneralRateLimit)
}

func runbookContents() string {
	return `# MCP Runbooks

> **Pick your signing path first.** Call get_server_info once per session and inspect agentBroker.enabled. If true, follow the "Quick path" sections below; the broker auto-authenticates and signs server-side, so you should NOT call authenticate, set_guardrails, preview_auth_message, preview_trade_signature, place_order, modify_order, cancel_order, cancel_all_orders, or close_position. If false, you must hold a private key locally and follow the "Wallet path" sections. Never ask a human user to paste an EIP-712 signature into chat.

## Session bootstrap

### Quick path (broker enabled)
1. Call get_context for a consolidated snapshot of server, session, markets, and account. The first quick_* call you make also auto-authenticates the session against the broker's wallet.
2. Confirm capabilities.agentBroker.enabled = true and capabilities.recommendedFlow lists the quick_* tools.

### Wallet path (broker disabled)
1. Call get_context for a consolidated snapshot of server, session, markets, and account.
2. If not yet authenticated, call preview_auth_message to get the EIP-712 typed-data to sign for session auth, sign it locally with your private key, then call authenticate with the serialized typedData and 65-byte signature.
3. Call set_guardrails (preset='standard' is a safe default) so trading tools don't fall back to read_only.
4. Call get_context again to confirm authMode='authenticated' and review account margin.

## Market analysis (signing-path agnostic)
1. Call get_market_summary for the target symbol.
2. Call get_orderbook to assess liquidity and spread.
3. Call get_recent_trades to observe trade flow direction.
4. Call get_funding_rate to understand funding cost/benefit.
5. Read market://specs/{symbol} for contract specifications.

## Pre-trade preparation

### Quick path (broker enabled)
1. Call get_account_summary to check margin capacity.
2. Call get_positions to understand existing exposure.
3. Call get_open_orders to check for pending orders on the same market.
4. Call quick_place_order with {symbol, side, type, quantity, price?, clientOrderId}. The broker validates guardrails, signs the placeOrders action, and submits in one round trip.
5. Inspect the response's phase and followUp fields. phase='ACCEPTED' is a successful resting order; phase='PENDING_CONFIRMATION' means poll get_open_orders / get_order_history with the returned clientOrderId; phase='REJECTED' carries errorCode and errorDetail.

### Wallet path (broker disabled)
1. Call get_account_summary to check margin capacity.
2. Call get_positions to understand existing exposure.
3. Call get_open_orders to check for pending orders on the same market.
4. Call preview_order to validate order shape before submission.
5. Call preview_trade_signature with action='placeOrders' to get the EIP-712 typed-data plus server-generated nonce and expiresAfter.
6. Sign typedData, split signature into {r, s, v}, and call place_order with the echoed nonce and expiresAfter.

## Position unwind

### Quick path (broker enabled)
- To close a position: quick_close_position with {symbol, quantity?, method?='market', limitPrice?}. Reduce-only is enforced server-side; omit quantity for a full close.
- To cancel one open order: quick_cancel_order with venueOrderId or clientOrderId.
- To cancel all open orders (optionally per symbol): quick_cancel_all_orders.
- There is no quick equivalent for modify_order today; cancel and re-place instead.

### Wallet path (broker disabled)
- Use preview_trade_signature with action='closePosition' / 'cancelOrders' / 'cancelAllOrders' / 'modifyOrder', sign locally, then call the matching write tool with the echoed nonce/expiresAfter and {r,s,v} signature.

## Session refresh
1. Call restore_session to extend the current MCP session state when a client preserves the same MCP session context.
2. Open a new connection and call authenticate (wallet path) or any quick_* tool (broker path) again if the client cannot preserve the current MCP session ID.
3. Call get_session to confirm the current authenticated identity and active subscriptions.

## Failure handling
1. If a tool returns TIMEOUT or BACKEND_UNAVAILABLE, check system://status before retrying.
2. If an authenticated call fails with AUTH_REQUIRED, re-authenticate (wallet path) or re-issue the quick_* call so the broker re-authenticates the session (broker path).
3. If RATE_LIMITED, check get_rate_limits and back off accordingly.
4. If delegation changes are suspected, re-authenticate to refresh session authority. On the broker path this happens automatically on the next quick_* call.
`
}
