package prompts

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Wires the prompt catalog with a quickstart body matching broker availability.
func Register(server *mcp.Server, brokerEnabled bool) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "quickstart",
		Title:       "Quickstart: First Trade",
		Description: "End-to-end onboarding walk-through. Defaults to the canonical self-hosted broker flow; uses the wallet-side preview-and-sign flow only when the broker is disabled.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "subAccountId",
				Description: "Target subaccount ID the agent has signing authority for (as a decimal string). Optional when the self-hosted broker is enabled because the broker resolves its own subaccount.",
			},
			{
				Name:        "symbol",
				Description: "Market symbol to probe and trade on, e.g. BTC-USDT.",
				Required:    true,
			},
			{
				Name:        "side",
				Description: "Order side for the first trade: BUY or SELL. Defaults to BUY.",
			},
			{
				Name:        "quantity",
				Description: "Order quantity as a decimal string (base-asset units). Defaults to the market's minimum trade amount.",
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		subAccountID := req.Params.Arguments["subAccountId"]
		symbol := req.Params.Arguments["symbol"]
		side := req.Params.Arguments["side"]
		if side == "" {
			side = "BUY"
		}
		qty := req.Params.Arguments["quantity"]
		qtyHint := qty
		if qtyHint == "" {
			qtyHint = "<minTradeAmount from get_market_summary.market.minTradeAmount>"
		}

		if brokerEnabled {
			return promptResult("Quickstart: First Trade (self-hosted broker enabled)", quickstartBrokerBody(symbol, side, qtyHint)), nil
		}
		return promptResult("Quickstart: First Trade (advanced wallet path)", quickstartWalletBody(symbol, side, qtyHint, subAccountID)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "startup-validation",
		Title:       "Startup Validation",
		Description: "Validate MCP session readiness and exchange connectivity before trading. Confirms authentication, account state, positions, orders, and upstream pacing guidance.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "subAccountId",
				Description: "Target authenticated subaccount ID",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		subAccountID := req.Params.Arguments["subAccountId"]
		return promptResult("Startup Validation", fmt.Sprintf(`You are validating readiness for subaccount %s on the Synthetix MCP trading server. Follow these steps in order and report any failures:

1. Call ping to confirm the MCP server is reachable.
2. Call get_server_info to discover the environment, supported auth modes, and any disabled features.
3. Call get_session to verify the session is authenticated for subaccount %s.
   - If not authenticated, call authenticate first.
4. Call get_session and inspect agentGuardrails. Show active or default guardrails in plain language, including allowed symbols, allowed order types, max order notional/quantity, max position notional/quantity, and whether writes are enabled. Guardrails are optional operator limits, not a prerequisite.
   - If the operator asks to tighten limits, call set_guardrails with the revised values and then call get_session again to show the updated guardrails.
5. Call get_account_summary to retrieve margin health, collateral balances, and fee tier.
   - Flag if available margin is below 10%% of account value.
6. Call get_positions to check for open positions. Note any concentrated or high-leverage exposure.
7. Call get_open_orders to check for pending orders that may fill unexpectedly.
8. If authenticated rate-limit usage is available, call get_rate_limits; otherwise use system://server-info rateLimitGuidance for retry behavior.

Summarize: session status, margin health, number of open positions and orders, and any warnings.`, subAccountID, subAccountID)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "market-analysis",
		Title:       "Market Analysis",
		Description: "Analyze a specific market's trading conditions, price structure, liquidity, funding, and recent trade flow.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "symbol",
				Description: "Market symbol such as BTC-USDT",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		symbol := req.Params.Arguments["symbol"]
		return promptResult("Market Analysis", fmt.Sprintf(`You are analyzing the %s perpetual futures market on Synthetix. Gather data and produce a structured analysis:

1. Read market://specs/%s to understand contract specifications, margin tiers, and tick size.
2. Call get_market_summary for %s to get current prices (index, mark, last), 24h volume, open interest, and funding.
3. Call get_orderbook for %s to assess liquidity depth and bid-ask spread.
4. Call get_recent_trades for %s to observe recent trade flow (direction, size, fill type).
5. Call get_funding_rate for %s to check the current and estimated funding rate.

Produce a summary with:
- Price context: mark vs index premium/discount, last traded price trend.
- Liquidity: bid-ask spread, depth at top 3 levels.
- Funding: current rate direction, whether it favors longs or shorts.
- Volume: 24h volume relative to open interest.
- Trade flow: recent net direction bias and average trade size.
- Risk factors: any unusual spread widening, low liquidity, or extreme funding.`, symbol, symbol, symbol, symbol, symbol, symbol)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "position-risk-report",
		Title:       "Position Risk Report",
		Description: "Generate a comprehensive portfolio risk report covering margin health, directional exposure, concentration risk, and liquidation proximity.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "subAccountId",
				Description: "Target authenticated subaccount ID",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		subAccountID := req.Params.Arguments["subAccountId"]
		return promptResult("Position Risk Report", fmt.Sprintf(`You are creating a risk report for subaccount %s. Gather data and analyze:

1. Call get_account_summary to retrieve margin health:
   - Available margin, initial margin, maintenance margin, unrealized PnL, account value.
2. Call get_positions to get all open positions:
   - Entry price, current unrealized PnL, liquidation price, position size, leverage.
3. For each position, call get_market_summary to get current mark price and funding.

Report:
- Margin utilization: initial margin / account value as a percentage. Flag if above 80%%.
- Liquidation proximity: for each position, distance from mark price to liquidation price as a percentage. Flag any position within 10%%.
- Directional exposure: net long vs short notional value across all positions.
- Concentration risk: any single position using more than 50%% of total margin.
- Funding exposure: net funding rate impact across positions (paying or receiving).
- Unrealized PnL: total and per-position, highlighting largest winners and losers.
- Recommendations: specific actions to reduce risk if any thresholds are exceeded.`, subAccountID)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "pre-trade-checklist",
		Title:       "Pre-Trade Checklist",
		Description: "Execute a structured safety checklist before submitting an order. Validates market conditions, account capacity, existing exposure, and order shape.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "symbol",
				Description: "Market symbol such as ETH-USDT",
				Required:    true,
			},
			{
				Name:        "side",
				Description: "Order side: BUY or SELL",
				Required:    true,
			},
			{
				Name:        "quantity",
				Description: "Order quantity as a decimal string",
				Required:    true,
			},
			{
				Name:        "price",
				Description: "Limit price (omit for market orders)",
			},
			{
				Name:        "subAccountId",
				Description: "Target authenticated subaccount ID",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		symbol := req.Params.Arguments["symbol"]
		side := req.Params.Arguments["side"]
		quantity := req.Params.Arguments["quantity"]
		price := req.Params.Arguments["price"]
		subAccountID := req.Params.Arguments["subAccountId"]

		orderDesc := fmt.Sprintf("%s %s x%s", side, symbol, quantity)
		if price != "" {
			orderDesc += fmt.Sprintf(" @ %s", price)
		}

		return promptResult("Pre-Trade Checklist", fmt.Sprintf(`You are running a pre-trade safety checklist for: %s on subaccount %s.

Complete each check. Report PASS, WARN, or FAIL for each:

1. MARKET STATUS: Call get_market_summary for %s.
   - PASS if market is open. FAIL if closed or suspended.
   - Note the current mark price and best bid/ask for price reasonableness.

2. ACCOUNT CAPACITY: Call get_account_summary.
   - PASS if available margin can absorb the estimated initial margin for this order.
   - WARN if available margin is below 20%% of account value after this trade.
   - FAIL if insufficient margin.

3. EXISTING EXPOSURE: Call get_positions for %s.
   - WARN if this order would increase an already large position.
   - WARN if this creates a position on the same side as existing positions (concentration).
   - Note if this is a new position vs adding to existing.

4. OPEN ORDERS: Call get_open_orders for %s.
   - WARN if there are existing orders on the same side that could compound fills.
   - Note total pending order exposure.

5. GUARDRAIL REVIEW: Call get_session and show the active or default guardrails to the user.
   - Include allowed symbols, allowed order types, max order notional/quantity, max position notional/quantity, and whether writes are enabled.
   - Guardrails are optional operator limits. Fold this information into the one trade confirmation; if the user asks to edit limits, call set_guardrails and then call get_session again to confirm the updated guardrails.

6. ORDER VALIDATION: Call preview_order with the order parameters.
   - PASS if canSubmit is true and no validation errors.
   - FAIL if validation errors are returned. List them.

7. LIQUIDITY CHECK: Review the orderbook from step 1.
   - WARN if the order quantity exceeds visible liquidity at the top 3 levels.
   - For limit orders, note queue position relative to best bid/ask.

Final verdict: READY TO TRADE, PROCEED WITH CAUTION, or DO NOT TRADE.`, orderDesc, subAccountID, symbol, symbol, symbol)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "find_tightest_spread",
		Title:       "Find Tightest Spread",
		Description: "Compare top-of-book spreads across symbols and identify the most liquid candidate.",
		Arguments: []*mcp.PromptArgument{
			{Name: "symbols", Description: "Comma-separated market symbols, e.g. BTC-USDT,ETH-USDT,SOL-USDT", Required: true},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		symbols := req.Params.Arguments["symbols"]
		return promptResult("Find Tightest Spread", fmt.Sprintf(`Compare spreads for: %s.

1. Call get_system_health and stop if REST is not healthy.
2. For each symbol, call get_orderbook with limit=5.
3. Compute absolute spread = bestAsk - bestBid and relative spread = absoluteSpread / mid.
4. Rank symbols from tightest to widest relative spread.
5. Return a concise table with symbol, bestBid, bestAsk, mid, absolute spread, relative spread, and a recommendation.`, symbols)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "flatten_all_with_preview",
		Title:       "Flatten All With Preview",
		Description: "Preview then flatten all open positions using reduce-only self-hosted broker tools when available.",
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return promptResult("Flatten All With Preview", `Flatten the authenticated account safely.

1. Call get_system_health and get_auth_status.
2. Call get_positions and summarize every non-zero position.
3. Ask for confirmation before sending any order.
4. If self-hosted broker signing is enabled, call close_position once per symbol with method="market" and omit quantity for a full close.
5. Use the wallet-signing fallback only when the operator intentionally disabled the self-hosted broker. In that mode, use preview_trade_signature action=closePosition for each symbol, sign locally, then call signed_close_position.
6. After submissions, call get_positions and get_open_orders to confirm exposure and resting orders.`), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "monitor_funding_above_threshold",
		Title:       "Monitor Funding Threshold",
		Description: "Scan funding rates and flag markets above a caller-provided threshold.",
		Arguments: []*mcp.PromptArgument{
			{Name: "threshold", Description: "Funding-rate threshold as a decimal string", Required: true},
			{Name: "symbols", Description: "Optional comma-separated symbols. Omit to call list_markets first."},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		threshold := req.Params.Arguments["threshold"]
		symbols := req.Params.Arguments["symbols"]
		if symbols == "" {
			symbols = "<all open markets from list_markets>"
		}
		return promptResult("Monitor Funding Threshold", fmt.Sprintf(`Monitor funding rates above %s for %s.

1. Call get_system_health.
2. If no symbols were supplied, call list_markets with status="open".
3. Call get_funding_rate for each symbol.
4. Flag markets where abs(estimatedFundingRate) exceeds %s.
5. For flagged markets, call get_funding_rate_history to show recent trend.
6. Return symbol, estimated rate, last settlement rate, next funding time, and whether longs or shorts pay.`, threshold, symbols, threshold)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "place_limit_relative_to_mid",
		Title:       "Place Limit Relative To Mid",
		Description: "Place a limit order offset from current mid-price with market checks and self-hosted broker-aware routing.",
		Arguments: []*mcp.PromptArgument{
			{Name: "symbol", Description: "Market symbol, e.g. BTC-USDT", Required: true},
			{Name: "side", Description: "BUY, SELL, long, or short", Required: true},
			{Name: "quantity", Description: "Order quantity", Required: true},
			{Name: "offsetBps", Description: "Price offset from mid in basis points", Required: true},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		symbol := req.Params.Arguments["symbol"]
		side := req.Params.Arguments["side"]
		quantity := req.Params.Arguments["quantity"]
		offsetBps := req.Params.Arguments["offsetBps"]
		return promptResult("Place Limit Relative To Mid", fmt.Sprintf(`Prepare a limit order for %s %s quantity=%s offset=%s bps from mid.

1. Call get_system_health and get_auth_status.
2. Call get_orderbook for %s with limit=5 and compute mid=(bestBid+bestAsk)/2.
3. For BUY, limit price = mid * (1 - offsetBps/10000). For SELL, limit price = mid * (1 + offsetBps/10000).
4. Call preview_order with type="LIMIT", timeInForce="GTC", and the computed price.
5. If canSubmit=false, stop and report validation errors.
6. Submit with place_order when the self-hosted broker is enabled. Use preview_trade_signature action=placeOrders, local signing, and signed_place_order only for the advanced wallet path.
7. Confirm outcome with get_open_orders or get_order_history.`, side, symbol, quantity, offsetBps, symbol)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "protect_session_with_dead_man_switch",
		Title:       "Protect Session With Dead-Man Switch",
		Description: "Arm, refresh, and disarm the Synthetix dead-man switch during an automated trading session.",
		Arguments: []*mcp.PromptArgument{
			{Name: "timeoutSeconds", Description: "Timeout in seconds before open orders are cancelled", Required: true},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		timeout := req.Params.Arguments["timeoutSeconds"]
		return promptResult("Protect Session With Dead-Man Switch", fmt.Sprintf(`Protect this trading session with the Synthetix dead-man switch using timeoutSeconds=%s.

1. Call get_system_health and get_auth_status.
2. If self-hosted broker signing is enabled, call arm_dead_man_switch with timeoutSeconds=%s.
3. Use the wallet-signing fallback only when the operator intentionally disabled the self-hosted broker: call preview_trade_signature action=scheduleCancel with scheduleCancel.timeoutSeconds=%s, sign locally, then call signed_arm_dead_man_switch.
4. During automation, refresh with keep_alive before half the timeout elapses.
5. Before shutdown, call disarm_dead_man_switch when self-hosted broker signing is enabled, or sign scheduleCancel with timeoutSeconds=0 and call signed_disarm_dead_man_switch.
6. Call get_dead_man_switch_status after arm, refresh, and disarm to confirm local MCP state.`, timeout, timeout, timeout)), nil
	})
}

// Self-hosted broker quickstart body. No client-side EIP-712: canonical tools
// auto-authenticate against the broker wallet, apply broker guardrail
// defaults, sign, and submit in one round trip.
func quickstartBrokerBody(symbol, side, qtyHint string) string {
	return fmt.Sprintf(`You are onboarding to a self-hosted Synthetix v4 MCP server. The self-hosted broker is enabled, so this MCP process has a configured delegate key. You do NOT hold a private key and you do NOT need to call authenticate, set_guardrails, preview_auth_message, preview_trade_signature, signed_place_order, signed_modify_order, signed_cancel_order, signed_cancel_all_orders, or signed_close_position. Use the canonical broker tools below. NEVER ask the human user to paste an EIP-712 signature, hex digest, or private key into chat — there is no scenario in which that is required on this server.

Goal: place %s %s qty=%s as your first trade.

1. CONFIRM SERVER STATE
   - Call get_server_info and verify agentBroker.enabled = true. (If false, this prompt was rendered against the wrong configuration; ask the operator to re-enable the self-hosted broker with SNXMCP_AGENT_BROKER_ENABLED or re-render the prompt.)
   - Read resource system://status to confirm the server reports "running".

2. INSPECT THE MARKET
   - Call get_market_summary with symbol="%s" to learn tickSize, minTradeAmount, best bid/ask, mark price, and funding.
   - Call get_orderbook with symbol="%s" limit=10 to check liquidity depth on both sides.
   - If you did not supply a quantity, use minTradeAmount as your first-trade quantity; it is the smallest order the venue will accept.

3. CONFIRM ACCOUNT CAPACITY
   - Call get_account_summary. Verify available margin is strictly greater than the estimated initial margin for this order. (The first broker write also auto-authenticates the session against the self-hosted broker wallet, so this read is enough to bind the session.)
   - Call get_positions to note any existing exposure on %s.

4. REVIEW GUARDRAILS
   - Call get_session and inspect agentGuardrails. If guardrails have not been materialized yet, show the broker default guardrails advertised by get_server_info / get_context and state that the first broker write will apply those defaults.
   - Present allowed symbols, allowed order types, max order notional/quantity, max position notional/quantity, and whether writes are enabled as part of the single trade confirmation. Guardrails are optional operator limits, not a prerequisite.
   - Ask for confirmation at most once for this trade. Combine order details, account capacity, and guardrails into that single prompt; do not ask separately to approve guardrails and then again to approve the order.

5. SUBMIT THE ORDER
   - Call place_order with {symbol="%s", side="%s", type="LIMIT" or "MARKET", quantity="%s", price="<your limit>" (omit for MARKET), timeInForce="GTC"}. Only include clientOrderId if you can generate a valid 0x-prefixed 32-hex value.
   - The self-hosted broker validates against its default guardrail preset, signs the placeOrders action, and submits in one round trip.

6. CHECK THE OUTCOME
   - Inspect the response. accepted=true with phase="ACCEPTED" is a successful resting limit order. phase="PENDING_CONFIRMATION" means the matching engine has not echoed the final state yet — poll get_open_orders / get_order_history with the returned clientOrderId, do NOT retry place_order. phase="REJECTED" carries errorCode and errorDetail; do not retry without addressing the error.
   - For a LIMIT order that rested: call get_open_orders with symbol="%s" and confirm your order appears.
   - For a MARKET order (or if a LIMIT crossed): call get_trade_history with symbol="%s" limit=5 and confirm the fill(s) appear.

7. UNWIND (optional)
   - To close the position: close_position with {symbol="%s", quantity?, method?="market"}. Reduce-only is enforced server-side; omit quantity for a full close.
   - To cancel one open order: cancel_order with venueOrderId or clientOrderId from step 5.
   - To cancel everything: cancel_all_orders (optionally per-symbol).
   - There is no broker modify_order today; cancel and re-place instead.

OUTPUT FORMAT
Report PASS/FAIL for each numbered step along with:
- The clientOrderId you assigned and the venueOrderId returned by place_order.
- The phase, accepted, and (if present) followUp fields from the response.
- Fill status (resting vs filled) and any resulting position delta.
- Any step that required a retry and why.

SAFETY RULES
- Never ask the user for a signature, private key, or hex digest. The self-hosted broker in this MCP process is the signer.
- Do not call the wallet-path tools (authenticate, preview_auth_message, preview_trade_signature, signed_place_order, signed_modify_order, signed_cancel_order, signed_cancel_all_orders, signed_close_position). Routing through them when the self-hosted broker is enabled duplicates work and tempts you to leak signature payloads to the user.
- If a broker write returns phase="REJECTED" with errorCode="GUARDRAIL_VIOLATION", treat it as intentional — do not silently call set_guardrails to loosen the limits without confirming with the operator.`,
		side, symbol, qtyHint,
		symbol, symbol,
		symbol,
		symbol, side, qtyHint,
		symbol, symbol,
		symbol,
	)
}

// Advanced wallet-path quickstart body. Used when the self-hosted broker is
// intentionally disabled and the agent must hold a private key locally.
func quickstartWalletBody(symbol, side, qtyHint, subAccountID string) string {
	if subAccountID == "" {
		subAccountID = "<your subaccount ID — call lookup_subaccount with your wallet address if unknown>"
	}
	return fmt.Sprintf(`You are onboarding to a Synthetix v4 MCP server and want to land your first authenticated trade: %s %s qty=%s on subaccount %s. The canonical path is the self-hosted broker using tools like place_order, but the self-hosted broker is DISABLED on this server. Continue only if the operator intentionally chose external wallet signing and you (the agent) hold a private key locally. If you do not, refuse the trade and tell the operator to enable the self-hosted broker (SNXMCP_AGENT_BROKER_ENABLED=true, see sample/node-scripts/scripts/onboard-agent-key.ts). NEVER ask the human user to paste an EIP-712 signature, hex digest, or private key into chat.

Execute the steps below in order. Do NOT hand-craft EIP-712 payloads — always use the preview tools.

1. DISCOVER SERVER CAPABILITIES
   - Call get_server_info to learn the environment, supported auth modes, chain id, and domain name/version. Confirm agentBroker.enabled = false.
   - Read resource system://agent-guide for the full tool catalog and auth rules.
   - Read resource system://status to confirm the server reports "running" before proceeding.

2. AUTHENTICATE
   - Call preview_auth_message with subAccountId="%s". The server returns the exact EIP-712 typed-data object and digest you must sign. Do not modify it.
   - Sign the returned typedData locally with an EIP-712-capable signer (viem.signTypedData, eth_signTypedData_v4, ethers Wallet._signTypedData, or Web3.py sign_typed_data) using the wallet that owns or has delegation for the subaccount.
   - Call authenticate with message=<serialized typedData JSON> and signatureHex=<0x-prefixed 65-byte signature>.
   - Call get_session and confirm authMode="authenticated" and subAccountId="%s".

3. OPTIONAL SAFETY NET
   - Guardrails are optional operator limits, not a prerequisite. If the operator wants tighter limits, propose preset="standard", allowedSymbols=["%s"], allowed order types, max order notional/quantity, and max position notional/quantity.
   - If the operator asks to apply or edit limits, call set_guardrails with the agreed values and call get_session after set_guardrails to show the active agentGuardrails.
   - Ask for confirmation at most once for the trade. Combine order details, account capacity, and any guardrails into that single prompt.

4. INSPECT THE MARKET
   - Call get_market_summary with symbol="%s" to learn tickSize, minTradeAmount, best bid/ask, mark price, and funding.
   - Call get_orderbook with symbol="%s" limit=10 to check liquidity depth on both sides.
   - If you did not supply a quantity, use minTradeAmount as your first-trade quantity; it is the smallest order the venue will accept.

5. CONFIRM ACCOUNT CAPACITY
   - Call get_account_summary. Verify available margin is strictly greater than the estimated initial margin for this order.
   - Call get_positions to note any existing exposure on %s.

6. PREVIEW + SIGN THE TRADE
   - Call preview_order with your proposed order parameters to validate shape (canSubmit, remaining limits, estimated fees). Fix any validation errors before signing.
   - Call preview_trade_signature with action="placeOrders" and placeOrder={symbol:"%s", side:"%s", type:"LIMIT" (or "MARKET"), timeInForce:"GTC", quantity:"%s", price:"<your limit>" (omit for MARKET), reduceOnly:false, postOnly:false}. Only include clientOrderId if you can generate a valid 0x-prefixed 32-hex value. The server returns a fresh nonce, expiresAfter, digest, and canonical typedData.
   - Sign the returned typedData locally (same signer as step 2).
   - Split the 65-byte signature into {r, s, v} where v is 27 or 28.

7. SUBMIT THE ORDER
   - Call signed_place_order with the same symbol/side/type/timeInForce/quantity/price/clientOrderId from step 6, PLUS the exact nonce and expiresAfter from the preview response, PLUS signature={r, s, v}.
   - On success, note the returned orderId (venueId is the authoritative id for signed_cancel_order/signed_modify_order).

8. VERIFY
   - For a LIMIT order that rested: call get_open_orders with symbol="%s" and confirm your order appears with the expected price and remaining quantity.
   - For a MARKET order (or if the LIMIT crossed): call get_trade_history with symbol="%s" limit=5 and confirm the fill(s) appear.
   - Call get_positions to see the resulting position delta.

9. CLEANUP (optional)
   - To cancel the order: preview_trade_signature action="cancelOrders" cancelOrder={venueOrderId:"<venueId>"}, sign locally, then call signed_cancel_order with the echoed nonce/expiresAfter/signature.
   - To unwind any position: preview_trade_signature action="closePosition" closePosition={symbol:"%s"}, sign locally, then call signed_close_position. signed_close_position guarantees reduce-only semantics.

OUTPUT FORMAT
Report PASS/FAIL for each numbered step along with:
- The nonce and digest returned by preview_trade_signature.
- The venueId of the placed order.
- Fill status (resting vs filled) and any resulting position.
- Any step that required a retry and why.

SAFETY RULES
- Never reuse a nonce; always re-call preview_trade_signature between attempts.
- Never skip the preview tools: their typedData is the exact bytes the server validates.
- Never ask the user for a signature, private key, or hex digest. If you cannot sign locally, refuse the trade.
- If any tool returns error=INVALID_SIGNATURE: re-run step 6 (the digest is nonce-bound, and re-signing an old nonce will fail).
- If set_guardrails blocks an order (GUARDRAIL_VIOLATION), treat it as intentional — do not silently loosen guardrails without confirming with the user.`,
		side, symbol, qtyHint, subAccountID,
		subAccountID, subAccountID,
		symbol,
		symbol, symbol,
		symbol,
		symbol, side, qtyHint,
		symbol, symbol,
		symbol,
	)
}

func promptResult(description string, text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: text},
			},
		},
	}
}
