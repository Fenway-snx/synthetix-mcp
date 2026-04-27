# Synthetix MCP Agent Guide

This guide is the canonical source for the `system://agent-guide` resource and is embedded directly into the service binary at build time. A pointer copy lives at `services/mcp/AGENT.md` for repository-level discovery.

## Agent Conduct (read before anything else)

Two non-negotiable rules apply to every LLM agent that connects to this server:

1. **Never ask a human to paste an EIP-712 signature, hex digest, or private key into chat.** Wallet signing is privileged. If you do not hold a key locally and the self-hosted broker is disabled, refuse the trade and surface the operator workaround (enable the self-hosted broker, or run a wrapper from `sample/node-scripts`). Pasting `typedData` from `preview_auth_message` / `preview_trade_signature` into the conversation is an instant onboarding failure.
2. **Use the self-hosted broker path by default.** The canonical MCP flow is: the operator runs this server, configures a delegated trading key, and agents route writes through simple tools like `place_order`. `get_server_info.agentBroker.enabled=true` means this MCP process is ready to sign and submit in one round trip. The wallet path exists only for advanced non-custodial setups where the MCP process must not hold a delegate key.

`get_context` returns the same signal under `capabilities.agentBroker.enabled` plus a `capabilities.recommendedFlow` array — call it once at the start of every session and follow the steps verbatim.

## Canonical Path: Self-Hosted Broker

> Use this section for normal agent operation. `get_server_info.agentBroker.enabled=true` (or `get_context.capabilities.agentBroker.enabled=true`) means this self-hosted MCP process has a configured delegate key and agents should use the canonical broker tools.

1. `get_market_summary` (and `get_orderbook` for limit orders) on the target symbol.
2. `place_order` with `{symbol, side, type, quantity, price?, clientOrderId}`. The self-hosted broker will:
   - auto-authenticate the session if needed (using its own EIP-712 key),
   - apply its default guardrails preset (commonly `standard`),
   - sign the `placeOrders` action with its key,
   - submit the order to the matching engine.
3. Inspect the response's `phase` and `followUp` fields. `accepted=true` with `phase="ACCEPTED"` is a successful resting order; `phase="PENDING_CONFIRMATION"` means poll `get_open_orders` / `get_order_history` with the returned `clientOrderId`; `phase="REJECTED"` carries `errorCode` and `errorDetail`.
4. To unwind: `close_position` (reduce-only) or `cancel_order` / `cancel_all_orders`. Use `arm_dead_man_switch` during unattended sessions and refresh with `keep_alive` before half the timeout elapses.

You should not call `authenticate`, `preview_auth_message`, `preview_trade_signature`, `signed_place_order`, `signed_cancel_order`, `signed_cancel_all_orders`, or `signed_close_position` when the self-hosted broker is enabled. Routing through them duplicates work and tempts you to leak signature payloads to the user.

## Secondary Path: External Wallet Signing

> Skip this section if `get_server_info.agentBroker.enabled = true`; the canonical self-hosted broker path above covers you. Use this section only when the operator explicitly chooses non-custodial signing and **you (the agent) hold a private key locally**. If neither is true, refuse the trade and tell the operator to enable the self-hosted broker (`SNXMCP_AGENT_BROKER_ENABLED=true`, see `sample/node-scripts/scripts/onboard-agent-key.ts`) — never ask the human user to paste a signature.

**Do not hand-roll EIP-712 signing.** Use the bundled SDK at `sample/node-scripts/src/mcp-client.ts` — it wraps `preview_auth_message` / `preview_trade_signature`, calls `signTypedData` with the exact domain/types/message returned by the server, and submits the result. Hand-rolled signers almost always misfire on numeric coercion (`uint256` fields must be passed as `bigint`, not `string` or `number`) and produce `INVALID_SIGNATURE`.

Five-line authenticate (`sample/node-scripts/scripts/*.ts`):

```ts
import { McpClient } from '../src/mcp-client.js';
import { createWalletClient, http } from 'viem';
import { privateKeyToAccount } from 'viem/accounts';

const account = privateKeyToAccount(process.env.PRIVATE_KEY as `0x${string}`);
const walletClient = createWalletClient({ account, transport: http() });
const mcp = new McpClient({ url: process.env.MCP_URL!, clientName: 'my-agent' });
await mcp.initialize();
await mcp.authenticate({ walletClient, subAccountId: BigInt(process.env.SUB_ACCOUNT_ID!) });
```

Don't know the `subAccountId` for your wallet? Call the public tool `lookup_subaccount` first (see below) — it returns every subaccount the wallet owns (and, optionally, every subaccount it can act on as a delegate). Agents should **never** drop to the REST API just to discover their subaccount.

For full demos including order placement, ladder strategies, and the delegated-signer flow, read `sample/node-scripts/AGENTS.md` and the scripts under `sample/node-scripts/scripts/`.

## Tool Surface

### Discovery and Context
| Tool | Auth | Purpose |
|------|------|---------|
| `ping` | No | Health-check connectivity |
| `get_system_health` | No | Combined REST, WS, and auth readiness smoke test |
| `get_auth_status` | No | Session and self-hosted broker signing readiness |
| `get_server_info` | No | Discover capabilities, auth modes, disabled features |
| `get_context` | No | Consolidated snapshot: server, session, markets, account |
| `lookup_subaccount` | No | Map a wallet address to its `subAccountId`(s); pass `includeDelegations=true` to also list subaccounts where this wallet is a delegate |

### Market Intelligence (Public)
| Tool | Auth | Purpose |
|------|------|---------|
| `list_markets` | No | Enumerate available markets and trading constraints |
| `get_market_summary` | No | Full snapshot: prices, funding, volume, OI for one market |
| `get_orderbook` | No | Bid/ask depth for liquidity analysis |
| `get_recent_trades` | No | Recent fill history for trade flow analysis |
| `get_funding_rate` | No | Estimated and last-settlement funding rates |
| `get_funding_rate_history` | No | Historical funding-rate observations |
| `get_candles` | No | Historical OHLCV candlestick data |

### Session Management
| Tool | Auth | Purpose |
|------|------|---------|
| `authenticate` | No | Bind session to a delegated subaccount via EIP-712 |
| `preview_auth_message` | No | Return the exact EIP-712 typed-data to sign for `authenticate`. Start here before hand-crafting anything. |
| `get_session` | No | Inspect current session state, expiry, subscriptions |
| `restore_session` | Yes | Extend TTL of the current authenticated session |

### Account (Authenticated)
| Tool | Auth | Purpose |
|------|------|---------|
| `get_account_summary` | Yes | Margin health, collateral, fee tier |
| `get_positions` | Yes | Open positions with unrealized PnL and liquidation prices |
| `get_open_orders` | Yes | Pending orders with type, price, remaining quantity |
| `get_order_history` | Yes | Historical orders across all statuses |
| `get_trade_history` | Yes | Historical fills with fees and closed PnL |
| `get_funding_payments` | Yes | Funding payment history with aggregates |
| `get_performance_history` | Yes | Time-series account value and PnL snapshots |
| `get_balance_updates` | Yes | Collateral and balance ledger updates |
| `get_transfers` | Yes | Collateral transfer history |
| `get_position_history` | Yes | Historical positions |
| `get_portfolio` | Yes | Portfolio snapshots |
| `get_fees` | Yes | Current fee tier and fee schedule |
| `get_trades_for_position` | Yes | Trades tied to a position ID |
| `get_delegated_signers` | Yes | Delegated signers on the subaccount |
| `get_delegations_for_delegate` | Yes | Delegations granted to the current delegate |

### Trading (Secondary External Wallet Path)
| Tool | Auth | Purpose |
|------|------|---------|
| `preview_order` | Yes | Dry-run validation without submission |
| `preview_trade_signature` | Yes | Return the exact EIP-712 typed-data and server-generated nonce/expiresAfter to sign for a `signed_*` write action. Use instead of hand-crafting EIP-712 payloads. |
| `signed_place_order` | Yes + Sig | Submit an order to the matching engine |
| `signed_modify_order` | Yes + Sig | Change price/quantity of an open order |
| `signed_cancel_order` | Yes + Sig | Cancel one order by ID |
| `signed_cancel_all_orders` | Yes + Sig | Cancel all orders, optionally by symbol |
| `signed_close_position` | Yes + Sig | Submit a reduce-only counter-order |
| `signed_update_leverage` | Yes + Sig | Update leverage for one market |
| `signed_withdraw_collateral` | Yes + Sig | Withdraw collateral to an EVM address |
| `signed_transfer_collateral` | Yes + Sig | Transfer collateral between subaccounts |
| `signed_arm_dead_man_switch` | Yes + Sig | Cancel open orders if the session stops refreshing |
| `signed_disarm_dead_man_switch` | Yes + Sig | Clear the dead-man switch |
| `signed_add_delegated_signer` | Yes + Sig | Add delegated signing permissions |
| `signed_remove_delegated_signer` | Yes + Sig | Remove one delegated signer |
| `signed_remove_all_delegated_signers` | Yes + Sig | Remove all delegated signers |

### Trading (canonical self-hosted broker path)
These tools are only registered when `get_server_info.agentBroker.enabled = true`, which means this self-hosted MCP process has been configured with a delegate key.
| Tool | Auth | Purpose |
|------|------|---------|
| `place_order` | Self-hosted broker | Auto-authenticate, apply default guardrails, sign, and submit one order. No client-side EIP-712. |
| `close_position` | Self-hosted broker | Reduce-only close (full or partial) signed by this MCP process. |
| `close_all_positions` | Self-hosted broker | Reduce-only close for every open position, or a supplied symbol subset, in one batched request. |
| `cancel_order` | Self-hosted broker | Cancel one order by venueOrderId/clientOrderId, signed by this MCP process. |
| `cancel_all_orders` | Self-hosted broker | Cancel all orders (optionally per-symbol), signed by this MCP process. |
| `update_leverage` | Self-hosted broker | Update leverage in one broker-signed call. |
| `withdraw_collateral` | Self-hosted broker | Withdraw collateral in one broker-signed call. |
| `transfer_collateral` | Self-hosted broker | Transfer collateral in one broker-signed call. |
| `arm_dead_man_switch` | Self-hosted broker | Arm the dead-man switch in one broker-signed call. |
| `keep_alive` | Self-hosted broker | Refresh the last armed dead-man-switch timeout. |
| `disarm_dead_man_switch` | Self-hosted broker | Clear the dead-man switch in one broker-signed call. |

### Risk and Operational
| Tool | Auth | Purpose |
|------|------|---------|
| `get_rate_limits` | Yes | Authenticated upstream API rate-limit usage when available |
| `get_dead_man_switch_status` | Yes | Last-known MCP dead-man-switch state |

### Streaming
| Tool | Auth | Purpose |
|------|------|---------|
| `subscribe` | Varies | Add real-time event subscriptions (public or private) |
| `unsubscribe` | No | Remove active subscriptions |

`subscribe` only registers interest — events are pushed as `notifications/event` JSON-RPC frames over a long-lived SSE response on `GET /mcp` with the `Mcp-Session-Id` header. If you don't open that GET stream the server has nowhere to deliver events to. The TypeScript SDK exposes `client.startNotificationStream({ onEvent })` for this; raw clients should issue the GET themselves and parse `data:` lines for `method === "notifications/event"`.

### Resources
| URI | Purpose |
|-----|---------|
| `system://agent-guide` | This operating guide |
| `system://server-info` | Server identity, limits, delegation surface |
| `system://status` | Public liveness flag (`running` / `not_running`) |
| `system://fee-schedule` | Current fee rates (hydrated when authenticated) |
| `system://runbooks` | Operational runbooks for common workflows |
| `account://risk-limits` | Session-level risk and rate constraints |
| `market://specs/{symbol}` | Contract spec and funding for one market |

### Prompts
| Prompt | Purpose |
|--------|---------|
| `quickstart` | End-to-end first-trade walk-through using the preview tools |
| `startup-validation` | Validate session readiness before trading |
| `market-analysis` | Structured analysis of one market |
| `position-risk-report` | Portfolio risk assessment |
| `pre-trade-checklist` | Safety checklist before order submission |
| `find_tightest_spread` | Compare top-of-book spreads across symbols |
| `flatten_all_with_preview` | Safely flatten open positions |
| `monitor_funding_above_threshold` | Flag markets with elevated funding |
| `place_limit_relative_to_mid` | Build and submit a limit order from mid-price |
| `protect_session_with_dead_man_switch` | Arm, refresh, and disarm the dead-man switch |

## Operating Model

- This server does **not** use OAuth. Authentication is done via the `authenticate` MCP tool using an EIP-712 signature. MCP clients that cache OAuth state from other servers may surface a transient "SDK auth failed: HTTP 404" error when probing `/.well-known/oauth-authorization-server`; clear the client's cached credentials and reconnect.
- Start with `get_context` to orient -- it returns server status, session state, active markets with mark prices, and account summary in one call.
- Use public market tools to discover symbols, prices, funding, and market constraints before forming a trading view.
- Authenticate once per MCP session before using account tools, private streams, or trading tools.
- Session authentication is revalidated for private MCP access. If delegated access is revoked, the session is cleared and private tools will fail closed.
- External-wallet tools (`signed_*`) still require a per-action EIP-712 signature even after session authentication. Each signature needs a unique nonce. Canonical broker tools sign inside this MCP process.
- `restore_session` only extends the current MCP session TTL. If a client cannot preserve the same `Mcp-Session-Id`, reconnect and call `authenticate` again.

## Common Workflows

### First Connection
1. Call `ping` to confirm connectivity.
2. Call `get_context` to get the full trading context in one call.
3. Optionally call `list_markets` if you need detailed market constraints.

### Authenticate and Trade
**Recommended path: use the TypeScript SDK** — it handles steps 1–2 and 6–7 below in two method calls (`mcp.authenticate(...)`, `mcp.signAndSubmitTrade(...)`). The raw protocol below is documented for non-TS clients and to explain what the SDK does under the hood.

1. If you only have a wallet address (no `subAccountId` yet), call `lookup_subaccount` with that address. Pick an entry from `owned`. For delegated-signer keys, call again with `includeDelegations=true` and pick from `delegated`.
2. Call `preview_auth_message` with the target `subAccountId` to get the exact EIP-712 typed-data for session auth.
3. Sign the returned `typedData` with your wallet; call `authenticate` with the serialized typedData JSON as `message` and the 0x-prefixed hex signature as `signatureHex`.
4. Call `get_context` to confirm auth status and see account margin in one call.
5. Call `get_positions` and `get_open_orders` to understand current exposure.
6. Call `preview_order` to validate order shape.
7. Call `preview_trade_signature` with the order intent to get the canonical typed-data for the write action plus a server-generated nonce and expiresAfter.
8. Sign `typedData`, split the signature into `{r, s, v}`, and call `signed_place_order` with the same fields plus the echoed `nonce`, `expiresAfter`, and `signature`.

### Monitor Positions
1. Open the notification stream first (TypeScript SDK: `client.startNotificationStream({ onEvent })`; raw HTTP: `GET /mcp` with `Accept: text/event-stream` and the `Mcp-Session-Id` header). Without an open stream `subscribe` succeeds but no events are delivered.
2. Call `subscribe` with `accountEvents` for real-time fills, margin, and order updates.
3. Call `subscribe` with `marketPrices` or `orderbook` for live market data.
4. Use `get_positions` and `get_account_summary` periodically to reconcile.

### Close a Position
1. Call `get_positions` to confirm the current position side and quantity.
2. Call `preview_trade_signature` with `action="closePosition"` and `closePosition={symbol, method?, limitPrice?, quantity?}`. The server reads the live position to derive the counter-side (BUY for short, SELL for long) and the default close quantity (full open position when `quantity` is omitted), so the typed-data you sign matches what `signed_close_position` will submit.
3. Sign the returned `typedData` and call `signed_close_position` with the echoed `nonce`/`expiresAfter` and split `{r, s, v}` signature.
4. If the position changes between preview and submission, re-preview to avoid `INVALID_SIGNATURE`.

## Error Codes

| Code | Meaning | Action |
|------|---------|--------|
| `AUTH_REQUIRED` | Session not authenticated | Call `authenticate` |
| `INVALID_SIGNATURE` | EIP-712 payload/signature malformed | Rebuild and retry |
| `PERMISSION_DENIED` | Wallet not authorized for subaccount | Verify delegation |
| `INVALID_ARGUMENT` | Request fields failed validation | Fix fields, retry |
| `NOT_FOUND` | Market, order, or resource missing | Verify identifier |
| `RATE_LIMITED` | Upstream throttled | Back off with jitter; confirm write state before replaying |
| `TIMEOUT` | Upstream timed out | Retry, check `system://status` |
| `BACKEND_UNAVAILABLE` | Upstream service down | Retry, check `system://status` |
| `NOT_IMPLEMENTED` | Tool stubbed in Phase 1 | Use documented fallback |

## Anti-Patterns

- Do not retry `cancel_all_orders` without checking `get_open_orders` first.
- Do not assume `preview_order` guarantees margin or matching acceptance.
- Do not cache market config indefinitely; markets can open, close, or change constraints.
- Do not reuse nonces across different write tool calls.
- Do not call owner-only platform actions through MCP; they are excluded from Phase 1.
- Do not use `float64` for financial values; all prices, quantities, and margins are decimal strings.

## Common Pitfalls

These are the failure modes that consume the most onboarding time. Read them before writing your own MCP client.

### 1. `node -e "..."` fails to import `viem` / `@modelcontextprotocol/sdk`
Inline scripts run with the current working directory's `package.json` and `node_modules`. If you launch `node -e` from `/tmp` (or any directory that does not contain the project's deps), ESM resolution will fail with `ERR_MODULE_NOT_FOUND`.

**Fix:** Always run inside `sample/node-scripts/` (which has the SDK, `viem`, and `tsx` installed) and call a `.ts` file directly: `cd sample/node-scripts && npx tsx scripts/your-script.ts`. Do not paste TypeScript into `node -e`. If you need a one-off, drop a `.ts` file in `sample/node-scripts/scripts/` and execute it from there.

### 2. `signTypedData` succeeds but server returns `INVALID_SIGNATURE`
The most common cause is **numeric coercion**. EIP-712 `uint256` fields (e.g. `subAccountId`, `timestamp`, `nonce`, `expiresAfter`, every order quantity/price field) must be passed to `signTypedData` as `bigint`. Strings, JS `number`s, or hex strings will sign a different message than the server expects, even though `signTypedData` returns a valid-looking 0x signature.

**Fix:** Use the SDK (it handles coercion). If you must hand-roll, call `preview_auth_message` / `preview_trade_signature` and convert every numeric field in `message` to `BigInt(...)` *before* calling `signTypedData`. Never edit the `domain`, `types`, or `primaryType` returned by the preview tools — sign exactly what the server gave you.

### 3. Decimal scaling mismatches on order fields
Quantities and prices in `signed_place_order` / `signed_modify_order` / `signed_close_position` are 18-decimal scaled integers (the on-chain representation), not human-readable decimals. `preview_trade_signature` returns these in the canonical scaled form — just pass them through.

**Fix:** Always derive `quantity`, `price`, `triggerPrice`, etc. from `preview_trade_signature`. Do not multiply human-readable strings by `1e18` yourself; small floating-point errors will produce `INVALID_SIGNATURE`.

### 4. Lost `Mcp-Session-Id` after a network blip
`restore_session` extends the TTL of the *current* session; it cannot recover a session whose `Mcp-Session-Id` your client has dropped. Reconnects that get a new `Mcp-Session-Id` start unauthenticated.

**Fix:** Persist `Mcp-Session-Id` across HTTP retries. If you genuinely lost it, call `authenticate` again — `preview_auth_message` + `authenticate` is cheap.

### 5. Lowercase wallet addresses returning empty subaccount lists
Older clients and ad-hoc scripts often pass `walletAddress` in lowercase. The MCP server normalises to EIP-55 checksum before query, so `lookup_subaccount` works either way; **but** the public REST `/getSubAccountIds` endpoint enforces checksum and 400s on lowercase.

**Fix:** Prefer `lookup_subaccount` (case-insensitive) over the REST endpoint. If you must hit REST, run the address through `viem`'s `getAddress(...)` first.

### 6. Reusing a nonce across two write tools
Each `signed_place_order`, `signed_modify_order`, `signed_cancel_order`, `signed_cancel_all_orders`, `signed_close_position` consumes its own nonce. Calling `preview_trade_signature` once and trying to use the same `nonce` for two different submissions will fail the second one.

**Fix:** Call `preview_trade_signature` immediately before each write action. The SDK's `signAndSubmitTrade` does this automatically.
