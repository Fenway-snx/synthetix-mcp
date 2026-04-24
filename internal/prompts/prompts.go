package prompts

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register wires up the prompt catalog. brokerEnabled toggles the
// quickstart prompt body between the broker (quick_*) flow and the
// wallet-holding flow so the rendered prompt can be executed verbatim
// without first probing get_server_info.
func Register(server *mcp.Server, brokerEnabled bool) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "quickstart",
		Title:       "Quickstart: First Trade",
		Description: "End-to-end onboarding walk-through: place a first order using the server-side broker (when enabled) or the wallet-side preview-and-sign flow. Renders a concrete step-by-step script the agent can execute verbatim.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "subAccountId",
				Description: "Target subaccount ID the agent has signing authority for (as a decimal string). Optional on broker-enabled servers — the broker resolves its own subaccount.",
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
			return promptResult("Quickstart: First Trade (broker enabled)", quickstartBrokerBody(symbol, side, qtyHint)), nil
		}
		return promptResult("Quickstart: First Trade (wallet path)", quickstartWalletBody(symbol, side, qtyHint, subAccountID)), nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "startup-validation",
		Title:       "Startup Validation",
		Description: "Validate MCP session readiness and exchange connectivity before trading. Confirms authentication, account state, positions, orders, and rate limits.",
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
4. Call get_account_summary to retrieve margin health, collateral balances, and fee tier.
   - Flag if available margin is below 10%% of account value.
5. Call get_positions to check for open positions. Note any concentrated or high-leverage exposure.
6. Call get_open_orders to check for pending orders that may fill unexpectedly.
7. Call get_rate_limits to confirm the current throttling thresholds.

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

5. ORDER VALIDATION: Call preview_order with the order parameters.
   - PASS if canSubmit is true and no validation errors.
   - FAIL if validation errors are returned. List them.

6. LIQUIDITY CHECK: Review the orderbook from step 1.
   - WARN if the order quantity exceeds visible liquidity at the top 3 levels.
   - For limit orders, note queue position relative to best bid/ask.

Final verdict: READY TO TRADE, PROCEED WITH CAUTION, or DO NOT TRADE.`, orderDesc, subAccountID, symbol, symbol, symbol)), nil
	})
}

// Broker-enabled quickstart body. No client-side EIP-712 — quick_* tools
// auto-authenticate against the broker wallet, apply broker guardrail
// defaults, sign, and submit in one round trip.
func quickstartBrokerBody(symbol, side, qtyHint string) string {
	return fmt.Sprintf(`You are onboarding to the Synthetix v4 off-chain MCP server. The agent broker is enabled, so you do NOT hold a private key and you do NOT need to call authenticate, set_guardrails, preview_auth_message, preview_trade_signature, place_order, modify_order, cancel_order, cancel_all_orders, or close_position. Use the quick_* tools below. NEVER ask the human user to paste an EIP-712 signature, hex digest, or private key into chat — there is no scenario in which that is required on this server.

Goal: place %s %s qty=%s as your first trade.

1. CONFIRM SERVER STATE
   - Call get_server_info and verify agentBroker.enabled = true. (If false, this prompt was rendered against the wrong configuration; ask the operator to re-enable SNXMCP_AGENT_BROKER_ENABLED or re-render the prompt.)
   - Read resource system://status to confirm the server reports "running".

2. INSPECT THE MARKET
   - Call get_market_summary with symbol="%s" to learn tickSize, minTradeAmount, best bid/ask, mark price, and funding.
   - Call get_orderbook with symbol="%s" limit=10 to check liquidity depth on both sides.
   - If you did not supply a quantity, use minTradeAmount as your first-trade quantity; it is the smallest order the venue will accept.

3. CONFIRM ACCOUNT CAPACITY
   - Call get_account_summary. Verify available margin is strictly greater than the estimated initial margin for this order. (The first quick_* call also auto-authenticates the session against the broker wallet, so this read is enough to bind the session.)
   - Call get_positions to note any existing exposure on %s.

4. SUBMIT THE ORDER
   - Call quick_place_order with {symbol="%s", side="%s", type="LIMIT" or "MARKET", quantity="%s", price="<your limit>" (omit for MARKET), timeInForce="GTC", clientOrderId="<random-0x-hex>"}.
   - The broker validates against its default guardrail preset, signs the placeOrders action, and submits in one round trip.

5. CHECK THE OUTCOME
   - Inspect the response. accepted=true with phase="ACCEPTED" is a successful resting limit order. phase="PENDING_CONFIRMATION" means the matching engine has not echoed the final state yet — poll get_open_orders / get_order_history with the returned clientOrderId, do NOT retry quick_place_order. phase="REJECTED" carries errorCode and errorDetail; do not retry without addressing the error.
   - For a LIMIT order that rested: call get_open_orders with symbol="%s" and confirm your order appears.
   - For a MARKET order (or if a LIMIT crossed): call get_trade_history with symbol="%s" limit=5 and confirm the fill(s) appear.

6. UNWIND (optional)
   - To close the position: quick_close_position with {symbol="%s", quantity?, method?="market"}. Reduce-only is enforced server-side; omit quantity for a full close.
   - To cancel one open order: quick_cancel_order with venueOrderId or clientOrderId from step 5.
   - To cancel everything: quick_cancel_all_orders (optionally per-symbol).
   - There is no quick_modify_order today; cancel and re-place instead.

OUTPUT FORMAT
Report PASS/FAIL for each numbered step along with:
- The clientOrderId you assigned and the venueOrderId returned by quick_place_order.
- The phase, accepted, and (if present) followUp fields from the response.
- Fill status (resting vs filled) and any resulting position delta.
- Any step that required a retry and why.

SAFETY RULES
- Never ask the user for a signature, private key, or hex digest. The broker is the signer on this server.
- Do not call the wallet-path tools (authenticate, preview_auth_message, preview_trade_signature, place_order, modify_order, cancel_order, cancel_all_orders, close_position). They are still registered for compatibility but routing through them on a broker-enabled server duplicates work and tempts you to leak signature payloads to the user.
- If a quick_* call returns phase="REJECTED" with errorCode="GUARDRAIL_VIOLATION", treat it as intentional — do not silently call set_guardrails to loosen the limits without confirming with the operator.`,
		side, symbol, qtyHint,
		symbol, symbol,
		symbol,
		symbol, side, qtyHint,
		symbol, symbol,
		symbol,
	)
}

// Wallet-path quickstart body. Used when the broker is disabled and the
// agent must hold a private key locally; renders the canonical preview-
// and-sign choreography.
func quickstartWalletBody(symbol, side, qtyHint, subAccountID string) string {
	if subAccountID == "" {
		subAccountID = "<your subaccount ID — call lookup_subaccount with your wallet address if unknown>"
	}
	return fmt.Sprintf(`You are onboarding to the Synthetix v4 off-chain MCP server and want to land your first authenticated trade: %s %s qty=%s on subaccount %s. The agent broker is DISABLED on this server, so you (the agent) MUST hold a private key locally. If you do not, refuse the trade and tell the operator to either enable the broker (SNXMCP_AGENT_BROKER_ENABLED=true, see sample/node-scripts/scripts/onboard-agent-key.ts) or run a wrapper from sample/node-scripts. NEVER ask the human user to paste an EIP-712 signature, hex digest, or private key into chat.

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

3. SET A SAFETY NET
   - Call set_guardrails with preset="standard" (or a stricter preset) and allowedSymbols=["%s"]. This caps order quantity, position size, and symbol scope for this session only.

4. INSPECT THE MARKET
   - Call get_market_summary with symbol="%s" to learn tickSize, minTradeAmount, best bid/ask, mark price, and funding.
   - Call get_orderbook with symbol="%s" limit=10 to check liquidity depth on both sides.
   - If you did not supply a quantity, use minTradeAmount as your first-trade quantity; it is the smallest order the venue will accept.

5. CONFIRM ACCOUNT CAPACITY
   - Call get_account_summary. Verify available margin is strictly greater than the estimated initial margin for this order.
   - Call get_positions to note any existing exposure on %s.

6. PREVIEW + SIGN THE TRADE
   - Call preview_order with your proposed order parameters to validate shape (canSubmit, remaining limits, estimated fees). Fix any validation errors before signing.
   - Call preview_trade_signature with action="placeOrders" and placeOrder={symbol:"%s", side:"%s", type:"LIMIT" (or "MARKET"), timeInForce:"GTC", quantity:"%s", price:"<your limit>" (omit for MARKET), reduceOnly:false, postOnly:false, clientOrderId:"<random-0x-hex>"}. The server returns a fresh nonce, expiresAfter, digest, and canonical typedData.
   - Sign the returned typedData locally (same signer as step 2).
   - Split the 65-byte signature into {r, s, v} where v is 27 or 28.

7. SUBMIT THE ORDER
   - Call place_order with the same symbol/side/type/timeInForce/quantity/price/clientOrderId from step 6, PLUS the exact nonce and expiresAfter from the preview response, PLUS signature={r, s, v}.
   - On success, note the returned orderId (venueId is the authoritative id for cancel_order/modify_order).

8. VERIFY
   - For a LIMIT order that rested: call get_open_orders with symbol="%s" and confirm your order appears with the expected price and remaining quantity.
   - For a MARKET order (or if the LIMIT crossed): call get_trade_history with symbol="%s" limit=5 and confirm the fill(s) appear.
   - Call get_positions to see the resulting position delta.

9. CLEANUP (optional)
   - To cancel the order: preview_trade_signature action="cancelOrders" cancelOrder={venueOrderId:"<venueId>"}, sign locally, then call cancel_order with the echoed nonce/expiresAfter/signature.
   - To unwind any position: preview_trade_signature action="closePosition" closePosition={symbol:"%s"}, sign locally, then call close_position. close_position guarantees reduce-only semantics.

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
