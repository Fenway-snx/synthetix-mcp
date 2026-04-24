# synthetix-mcp

Streamable-HTTP MCP server for the public Synthetix V4 off-chain
trading API. Lets AI agents (Claude, Cursor, ChatGPT, anything that
speaks MCP) read market data, manage subaccounts, sign orders, and
stream live updates from `api.synthetix.io`.

- Built on top of [`github.com/synthetixio/synthetix-go`](https://github.com/synthetixio/synthetix-go)
- EIP-712 signing in-process via the `authenticate` MCP tool — no
  OAuth, no key servers, the broker holds an in-memory delegate key
  scoped to a single MCP session
- Loopback-only: ships with localhost binding and HTTP-source guards
  by default
- Tool surface includes `list_markets`, `get_account`, `place_orders`,
  `cancel_orders`, `subscribe_*`, plus the `quick_*` family for
  guarded one-shot trades

> Companion repo: the [`synthetix-go`](https://github.com/synthetixio/synthetix-go)
> SDK is the underlying client this server is built on. If you want to
> drive the API from a Go program directly (no LLM in the loop), use
> the SDK alone.

## Local endpoint

```text
http://localhost:8096/mcp           # MCP transport
http://localhost:8096/health        # liveness
http://localhost:8096/health/ready  # readiness
http://localhost:9089/metrics       # Prometheus
```

## Run it

### From source

```bash
git clone https://github.com/Fenway-snx/synthetix-mcp.git
cd synthetix-mcp
cp config.env.example config.env
set -a && source config.env && set +a
export SNXMCP_API_BASE_URL=https://api.synthetix.io
go run ./cmd/server
```

### Docker

```bash
docker build -t synthetix-mcp:dev .
docker run --rm -p 8096:8096 -p 9089:9089 \
  -e SNXMCP_API_BASE_URL=https://api.synthetix.io \
  synthetix-mcp:dev
```

## Configuration

All env vars use the `SNXMCP_` prefix. See [`config.env.example`](./config.env.example)
for the full list and defaults. Most useful in practice:

| Var | Default | Meaning |
| --- | --- | --- |
| `SNXMCP_API_BASE_URL` | `https://api.synthetix.io` | Public REST + WS endpoint to talk to. |
| `SNXMCP_LISTEN_ADDR` | `127.0.0.1:8096` | MCP transport bind. **Keep loopback** unless you put a TLS proxy in front. |
| `SNXMCP_METRICS_ADDR` | `127.0.0.1:9089` | Prometheus bind. |
| `SNXMCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |

`mcp-service` is standalone: **no Redis, no NATS, no internal gRPC**.
Sessions, nonces, and rate limits are kept in memory. All runtime
data comes from the public Synthetix REST + WebSocket API.

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
(e.g. `claude mcp remove synthetix-offchain && claude mcp add ...`)
and reconnect. Then authenticate via the `authenticate` tool using
`preview_auth_message`, or run the `quickstart` prompt for a guided
first-trade walk-through.

## Example client configs

### Cursor (`.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "synthetix-offchain": {
      "url": "http://localhost:8096/mcp"
    }
  }
}
```

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "synthetix-offchain": {
      "url": "http://localhost:8096/mcp"
    }
  }
}
```

### Claude Code (`.claude.json` or project settings)

```json
{
  "mcpServers": {
    "synthetix-offchain": {
      "url": "http://localhost:8096/mcp"
    }
  }
}
```

> **Note:** Claude Code may show a transient OAuth error on first
> connection. This is expected — the client probes for OAuth
> endpoints, receives a 404, and falls back to a direct connection.
> The server uses EIP-712 auth via the `authenticate` tool instead.

### Generic Streamable HTTP client

```json
{
  "name": "synthetix-offchain",
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
- Tool calls are rate limited per IP; authenticated tool calls are
  additionally rate limited per subaccount.

## Repo layout

```text
cmd/server/        - main package (the binary)
internal/          - server guts (auth, broker, tools, streaming, etc.)
internal/lib/      - vendored support code from the v4-offchain monorepo
                     (logging, auth cache, time providers, validators).
                     Marked internal/ so external consumers don't depend
                     on it; will be pruned over time.
examples/          - sample MCP client configurations
config.env.example - default env vars, copy to config.env to customise
Dockerfile         - production image build
```

The `internal/lib/` tree is intentionally untouched in this
extraction so the migration is reversible. Future cleanups will
prune dead code package by package.

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
