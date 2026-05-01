# synthetix-mcp

Streamable-HTTP MCP server for the public Synthetix V4
trading API. Lets AI agents (Claude, Cursor, ChatGPT, anything that
speaks MCP) read market data, manage subaccounts, sign orders, and
stream live updates from `papi.synthetix.io`.

- Built on top of [`github.com/synthetixio/synthetix-go`](https://github.com/synthetixio/synthetix-go)
- EIP-712 signing via MCP tools — no OAuth and no central key server.
  In self-hosted broker mode, the MCP process you run holds an
  in-memory delegate key
- Loopback-only: ships with localhost binding and HTTP-source guards
  by default
- Tool surface includes `list_markets`, `get_account_summary`,
  `place_order`, `cancel_order`, `subscribe`, and `signed_*` tools for
  advanced external-wallet signing

> Companion repo: the [`synthetix-go`](https://github.com/synthetixio/synthetix-go)
> SDK is the underlying client this server is built on. If you want to
> drive the API from a Go program directly (no LLM in the loop), use
> the SDK alone.

## Local endpoint

```text
http://localhost:8096/mcp           # MCP transport
http://localhost:8096/health        # liveness
http://localhost:8096/health/ready  # readiness
```

## Run it

For the guided first-run path, see [`GETTING_STARTED.md`](./GETTING_STARTED.md).

### Zero To Market Data

From a fresh clone, this is the shortest read-only path:

```bash
go run ./cmd/server --no-broker
```

Then register/check Claude in another terminal. The script starts the server
itself if it is not already reachable:

```bash
./scripts/setup-claude-code.sh
```

Open Claude Code, restart it if the script just added the server, then paste:

```text
Use the synthetix-dex-mcp MCP. Call ping, then get_server_info, then list_markets, and tell me the top 3 markets by 24h volume.
```

No wallet, broker key, or authentication is needed for public market data.

Read-only / external-wallet mode:

```bash
go run ./cmd/server --no-broker
# or: make run-readonly
```

Canonical broker mode:

```bash
./scripts/setup-broker-key.sh # terminal-only hidden prompt
set -a && source config.env && set +a
go run ./cmd/server
# or: make run
```

Claude Code setup/check:

```bash
./scripts/setup-claude-code.sh
./scripts/setup-claude-code.sh --dry-run   # preview only
./scripts/setup-claude-code.sh --no-start  # do not auto-start the server
./scripts/setup-claude-code.sh --verify    # health check + Claude ping if supported
```

The setup script can start the MCP server in the background if it is not
already running, registers the server with Claude Code when possible, then
prints exact verification prompts and common first-error fixes.

Comprehensive direct MCP smoke test:

```bash
node scripts/smoke-mcp-tools.mjs
node scripts/smoke-mcp-tools.mjs --symbol ETH-USDT
node scripts/smoke-mcp-tools.mjs --list-only
```

The smoke test talks to `http://localhost:8096/mcp` directly and exercises
discovery, public market tools, resources, prompts, session state, streaming
subscribe/unsubscribe bookkeeping, and broker-vs-signed routing. It never
places, cancels, or modifies live orders.

Copy-paste this as a new Claude session opener:

```text
You have MCP tools available from synthetix-dex-mcp. Before doing anything else, call ping to verify the connection. If ping fails or you see "unknown tool", stop, do not use Bash as a fallback, and tell me to restart Claude Code and run: claude mcp list
```

## Agent-friendly onboarding

The fastest smoke test after the server starts is:

```text
ping -> get_system_health -> get_auth_status -> get_context
```

For machine-readable tool routing, read `system://routing-guide`. In broker
mode the server only exposes the canonical write path (`place_order`,
`cancel_order`, `close_position`, etc.) and hides `signed_*` write tools plus
`preview_trade_signature` to avoid dual-path confusion.

The canonical onboarding path is the self-hosted broker path. Run this
MCP server yourself, configure a delegated trading key, then have agents
use canonical tools like `place_order`. They auto-authenticate, apply broker guardrails,
sign EIP-712 payloads inside your MCP process, and submit in one call. A
first guarded trade is:

```text
get_market_summary(symbol="BTC-USDT")
get_orderbook(symbol="BTC-USDT", limit=10)
place_order(symbol="BTC-USDT", side="buy", type="LIMIT",
  quantity="0.001", price="<chosen limit>", timeInForce="GTC")
```

The wallet path is secondary. Use it only when you intentionally do not
want this MCP process to hold a delegate key. In that mode, the agent
must use a local sidecar signer such as [`sample/node-scripts`](./sample/node-scripts)
to call `preview_auth_message`, sign EIP-712 locally, call `authenticate`,
then use `preview_trade_signature` and the signed write tools. Never ask a
human to paste private keys, typed-data payloads, digests, or signatures into
chat.

For unattended sessions, use the Synthetix dead-man switch:

```text
arm_dead_man_switch(timeoutSeconds=60)
keep_alive()        # refresh before 30 seconds elapse
disarm_dead_man_switch()
```

Useful prompt recipes are available through MCP prompts:
`quickstart`, `startup-validation`, `pre-trade-checklist`,
`find_tightest_spread`, `place_limit_relative_to_mid`,
`flatten_all_with_preview`, `monitor_funding_above_threshold`, and
`protect_session_with_dead_man_switch`.

### From source

```bash
git clone https://github.com/Fenway-snx/synthetix-mcp.git
cd synthetix-mcp
cp config.env.example config.env
# Fill SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX or SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE.
# Set SNXMCP_AGENT_BROKER_ENABLED=false only if you want the signed_* wallet path.
./scripts/setup-broker-key.sh
set -a && source config.env && set +a
go run ./cmd/server
```

### Docker

```bash
docker build -t synthetix-mcp:dev .
docker run --rm -p 8096:8096 \
  -e SNXMCP_API_BASE_URL=https://papi.synthetix.io \
  -e SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX=0x... \
  synthetix-mcp:dev
```

## Configuration

All env vars use the `SNXMCP_` prefix. See [`config.env.example`](./config.env.example)
for the full list and defaults. Most useful in practice:

| Var | Default | Meaning |
| --- | --- | --- |
| `SNXMCP_API_BASE_URL` | `https://papi.synthetix.io` | Public REST + WS endpoint to talk to. |
| `SNXMCP_SERVER_ADDRESS` | `127.0.0.1:8096` | MCP transport bind. **Keep loopback** unless you put a TLS proxy in front. |
| `SNXMCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |
| `SNXMCP_AGENT_BROKER_ENABLED` | `true` | Enables the canonical self-hosted broker tools. Set to `false` for external-wallet `signed_*` tools only. |
| `SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX` / `SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE` | empty | Required when broker mode is enabled. Use a delegated trading key, not an owner key. |
| `SNXMCP_SESSION_STORE_PATH` | empty | Optional file-backed session store, e.g. `.sessions/sessions.json`, so authenticated session bindings and guardrails survive local restarts. |

Never paste private keys into an agent chat. Use
`./scripts/setup-broker-key.sh` from a terminal, or set the env var / key
file yourself before starting the server.

`mcp-service` is standalone. By default, sessions are kept in memory and are
lost on restart. All runtime data comes from the public Synthetix REST +
WebSocket API. If `SNXMCP_SESSION_STORE_PATH` is set, authenticated session
bindings and guardrails are restored from that local file on restart. Private
keys, signatures, and request nonces are not persisted there.

## Authentication

This server does **not** speak OAuth. Authentication is handled via
EIP-712 signatures through the `authenticate` MCP tool after
connecting. Clients that probe RFC 8414 / RFC 9728 / OIDC discovery
endpoints under `/.well-known/` receive a JSON 404 with an
`auth_method: "mcp_tool"` hint and should fall through to a direct
connection automatically.

### Troubleshooting "SDK auth failed: HTTP 404: Invalid OAuth error response"

This error comes from a client (typically Claude Code) that cached
OAuth state from a previous server on the same host and is now trying
to refresh a stale token against this server. The server never
supported OAuth, so the probe correctly returns 404 — but the client's
OAuth code path surfaces it as a confusing error instead of falling
back.

Fix on the client side: clear the client's stored MCP auth state
(e.g. `claude mcp remove synthetix-dex-mcp && claude mcp add ...`)
and reconnect. Then authenticate via the `authenticate` tool using
`preview_auth_message`, or run the `quickstart` prompt for a guided
first-trade walk-through.

## Example client configs

### Cursor (`.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "synthetix-dex-mcp": {
      "url": "http://localhost:8096/mcp"
    }
  }
}
```

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "synthetix-dex-mcp": {
      "url": "http://localhost:8096/mcp"
    }
  }
}
```

### Claude Code (`.claude.json` or project settings)

Preferred setup:

```bash
./scripts/setup-claude-code.sh
```

Manual config:

```json
{
  "mcpServers": {
    "synthetix-dex-mcp": {
      "url": "http://localhost:8096/mcp"
    }
  }
}
```

> **Note:** Claude Code may show a transient OAuth error on first
> connection. This is expected — the client probes for OAuth
> endpoints, receives a 404, and falls back to a direct connection.
> The server uses EIP-712 auth via the `authenticate` tool instead.
>
> If Claude starts running Bash to inspect local config instead of
> calling Synthetix MCP tools, the MCP server is not connected in that
> Claude Code session. Run `./scripts/setup-claude-code.sh`, confirm
> `claude mcp list`, then restart Claude Code.

### Generic Streamable HTTP client

```json
{
  "name": "synthetix-dex-mcp",
  "transport": {
    "type": "streamable_http",
    "url": "http://localhost:8096/mcp",
    "headers": {}
  }
}
```

## Verification

```bash
go test ./...
go build ./...
curl -s http://localhost:8096/health/ready
```

## Agent notes

- `system://agent-guide` is sourced from [`AGENT.md`](./AGENT.md).
- `restore_session` only restores the current MCP session; clients
  that open a new connection should call `authenticate` again.
- Private tool / resource access is revalidated against the shared
  auth cache and fails closed after delegation revocation.
- Upstream rate-limit responses are surfaced as `RATE_LIMITED`; agents
  should back off and confirm write state before retrying.

## Repo layout

```text
cmd/server/        - main package (the binary)
internal/          - server guts (auth, broker, tools, streaming, etc.)
internal/lib/      - vendored support code from the legacy V4 API service
                     (logging, auth cache, time providers, validators).
                     Marked internal/ so external consumers don't depend
                     on it; will be pruned over time.
examples/          - sample MCP client configurations
sample/node-scripts/ - terminal-only external-wallet sidecar examples
config.env.example - default env vars, copy to config.env to customise
Dockerfile         - production image build
```

The `internal/lib/` tree is intentionally untouched in this
extraction so the migration is reversible. Future cleanups will
prune dead code package by package.

## Resources

- Synthetix Exchange: [synthetix.exchange](https://synthetix.exchange)
- Synthetix Docs: [docs.synthetix.io](https://docs.synthetix.io)
- Synthetix API: [papi.synthetix.io](https://papi.synthetix.io)
- Go SDK: [github.com/synthetixio/synthetix-go](https://github.com/synthetixio/synthetix-go)
- Python SDK: [github.com/Synthetixio/synthetix-sdk](https://github.com/Synthetixio/synthetix-sdk)
- MCP Server: [github.com/Fenway-snx/synthetix-mcp](https://github.com/Fenway-snx/synthetix-mcp)

## Versioning + stability

Pre-1.0. The MCP tool surface and env-var contract may change between
minor versions; breaks ship in release notes. Once an external
operator pins a release in production, this repo will tag `v1.0.0`
and follow [semver](https://semver.org/) strictly.

## Contributing

See [`CONTRIBUTING.md`](./CONTRIBUTING.md) for the dev loop, commit
conventions, and how to run tests locally.

## Security

Security disclosures: see [`SECURITY.md`](./SECURITY.md). Please
**do not** file public issues for vulnerabilities.

## License

[MIT](./LICENSE) © Synthetix.
