# Getting Started

This guide starts from a fresh clone and gets Synthetix MCP working in Claude
Code. It covers two paths:

- **Market data only:** no wallet or key required.
- **Trading mode:** self-hosted broker mode with a delegated trading key.

Never paste private keys, signatures, EIP-712 payloads, or digests into Claude,
Cursor, ChatGPT, or any agent chat.

## 1. Clone The Repo

```bash
git clone https://github.com/Fenway-snx/synthetix-mcp.git
cd synthetix-mcp
```

If you already cloned it, make sure you have the latest schema compatibility
fixes:

```bash
git pull
```

## 2. Zero To Market Data

Use this first. It proves the MCP server and Claude tool registration work
before you configure any trading key.

Start the server:

```bash
go run ./cmd/server --no-broker
```

Leave that terminal running. In a second terminal:

```bash
cd synthetix-mcp
./scripts/setup-claude-code.sh
claude mcp list
```

You should see:

```text
synthetix-dex-mcp: http://localhost:8096/mcp (HTTP) - Connected
```

You can also run the direct MCP smoke test:

```bash
node scripts/smoke-mcp-tools.mjs
```

This exercises tool/resource/prompt discovery, public market data tools,
session state, streaming subscribe/unsubscribe bookkeeping, and routing rules.
It does not place, cancel, or modify orders.

Restart Claude Code if the setup script just added or changed the server.
Then paste this into a new Claude session:

```text
Use the synthetix-dex-mcp MCP. Call ping, then get_server_info, then list_markets, and tell me the top 3 markets by 24h volume.
```

No wallet, broker key, or authentication is needed for public market data.

## 3. Trading Mode

Trading uses the canonical self-hosted broker path. The MCP process runs on your machine, holds a delegated trading key in process memory, applies operator-configured guardrails, signs EIP-712 payloads locally, and submits orders through tools like `place_order`.

Stop any server already using port `8096`:

```bash
lsof -ti tcp:8096 | xargs kill
```

Create local config and set up the broker key:

```bash
cp config.env.example config.env
./scripts/setup-broker-key.sh
```

The script prompts for your delegated trading private key with hidden terminal
input and writes it to `.secrets/agent-broker-private-key` by default. It also
updates `config.env` to point broker mode at that key file. Both `config.env`
and `.secrets/` are gitignored, and the key file is written with `0600`
permissions. Use a delegated trading key, not an owner key.

Optional but recommended, persist authenticated session bindings and guardrails across local restarts:

```bash
echo 'SNXMCP_SESSION_STORE_PATH=.sessions/sessions.json' >> config.env
```

Start the server in broker mode:

```bash
set -a && source config.env && set +a
go run ./cmd/server
```

Leave that terminal running. In a second terminal:

```bash
cd synthetix-mcp
./scripts/setup-claude-code.sh
claude mcp list
```

Restart Claude Code if needed, then verify the connection:

```text
You have MCP tools available from synthetix-dex-mcp. Before doing anything else, call ping to verify the connection. If ping fails or you see "unknown tool", stop, do not use Bash as a fallback, and tell me to restart Claude Code and run: claude mcp list.
```

Then start with a guided trading prompt:

```text
Use the synthetix-dex-mcp MCP. Read system://routing-guide, then run the quickstart prompt for BTC-USDT. Before placing any order, show me the market summary, orderbook, account status, guardrails, and the exact order you plan to submit.
```

For first real trades, keep size tiny and require explicit confirmation before
submission. Ask for confirmation once per trade or operation: include order
details, account context, and guardrails in the same prompt.

## 4. What Claude Should Use

In broker mode, Claude should use canonical tools:

```text
place_order
modify_order
cancel_order
cancel_all_orders
close_position
update_leverage
withdraw_collateral
transfer_collateral
arm_dead_man_switch
disarm_dead_man_switch
```

Claude should not use `signed_*` tools or `preview_trade_signature` in broker
mode. Read `system://routing-guide` for the machine-readable routing rules.

## 5. Client Setup

### Claude Code Onboarding

Register the local MCP server with Claude Code:

```bash
./scripts/setup-claude-code.sh
claude mcp list
```

You should see:

```text
synthetix-dex-mcp: http://localhost:8096/mcp (HTTP) - Connected
```

Restart Claude Code if the setup script just added or changed the server. Open a
new Claude session and start with:

```text
You have MCP tools available from synthetix-dex-mcp. Before doing anything else, call ping to verify the connection. If ping fails or you see "unknown tool", stop, do not use Bash as a fallback, and tell me to restart Claude Code and run: claude mcp list.
```

For broker-mode trading sessions, use the guided trading prompt in
[Trading Mode](#3-trading-mode) after `ping` succeeds.

### Cursor Setup

Add this to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "synthetix-dex-mcp": {
      "url": "http://localhost:8096/mcp"
    }
  }
}
```

Reload Cursor so the MCP server is discovered.

## 6. Troubleshooting

### Address Already In Use

Another server is already listening on port `8096`.

```bash
lsof -nP -iTCP:8096 -sTCP:LISTEN
lsof -ti tcp:8096 | xargs kill
```

Then restart:

```bash
set -a && source config.env && set +a
go run ./cmd/server
```

### Claude Shows Resources But Uses Bash Instead Of Tools

This means Claude is connected to the MCP server but did not ingest the tool
registry in the current session.

First make sure you are on the latest repo:

```bash
git pull
```

Restart the MCP server:

```bash
lsof -ti tcp:8096 | xargs kill
set -a && source config.env && set +a
go run ./cmd/server
```

Reset Claude's registration in another terminal:

```bash
claude mcp remove -s local synthetix-dex-mcp || true
claude mcp remove -s user synthetix-dex-mcp || true
claude mcp add -s user --transport http synthetix-dex-mcp http://localhost:8096/mcp
claude mcp list
```

Fully restart Claude Code and open a new session.

### Old `synthetix-offchain` Alias Appears

Remove it and use the current alias:

```bash
claude mcp remove synthetix-offchain || true
claude mcp add -s user --transport http synthetix-dex-mcp http://localhost:8096/mcp
```

### Server Exits With Broker Key Error

Broker mode is enabled by default. Either configure a broker key:

```bash
./scripts/setup-broker-key.sh
```

or start read-only/external-wallet mode:

```bash
go run ./cmd/server --no-broker
```

### WebSocket Bad Handshake

Use the public PAPI endpoint:

```bash
SNXMCP_API_BASE_URL=https://papi.synthetix.io
```

Older examples using `https://api.synthetix.io` will fail.

# Appendix

## Advanced/Optional External-Wallet Mode

External-wallet mode is for users who intentionally do not want broker signing.
Start without broker signing:

```bash
go run ./cmd/server --no-broker
```

Claude cannot sign EIP-712 payloads by itself. Use the terminal sidecar instead
of pasting signatures into chat:

```bash
cd sample/node-scripts
npm install
node authenticate-external-wallet.mjs \
  --session-id <SESSION_ID_FROM_CLAUDE_GET_SESSION> \
  --subaccount-id <SUBACCOUNT_ID> \
  --private-key-file ~/.synthetix/delegate-key
```

Flow:

```text
1. In Claude: call get_session and show only sessionId.
2. In terminal: run the node script above with that sessionId.
3. In Claude: call get_session again and confirm authMode is authenticated.
```

If `--session-id` is omitted, the script authenticates its own standalone MCP
session. That is useful for scripts, but it will not authenticate Claude's
separate session.
