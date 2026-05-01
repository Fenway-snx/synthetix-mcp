# Contributing to synthetix-mcp

Thanks for picking up a PR. The MCP server's job is to be a thin,
agent-friendly wrapper over the [synthetix-go](https://github.com/synthetixio/synthetix-go)
SDK. Anything that's not MCP-shaped (transport, tool dispatch,
guardrails, the broker key lifecycle) probably belongs upstream in
the SDK rather than here.

## Dev loop

```bash
git clone https://github.com/Fenway-snx/synthetix-mcp.git
cd synthetix-mcp

# While the SDK is private, point the replace at a local sibling clone:
git clone https://github.com/Fenway-snx/synthetix-go-sdk.git ../synthetix-go
# then edit go.mod's `replace` line to ../synthetix-go (already set
# during initial extraction).

go mod download
go build ./...
go test ./...
```

The test suite is mostly hermetic — REST and WebSocket interactions
are served by `httptest.Server` fakes. A few integration tests in
`internal/sdkparity` cross-check the MCP-side EIP-712 builders
against the lib/auth implementations to catch divergence; they don't
need network either.

## Before you open a PR

- `go build ./...`
- `go vet ./...`
- `go test ./...` (race detector adds 5–10s; use `-race` if you've
  touched the streaming / broker paths)
- `gofmt -s -w .`

If your change adds a new MCP tool, add at least:
- a happy-path test that exercises the JSON-Schema input validation
- a guardrail test (auth required? permissioned? side-effect free?)
- a one-line entry in `internal/tools/registry.go`'s ordering so it
  shows up consistently in tool listings

## Commit style

- Imperative subject line, ≤72 chars (`broker: drop unused
  legacy-signer fallback`).
- Body explains *why* — what user-visible behaviour changes, what
  trade-offs you considered. The diff already explains *what*.
- Prefix the subject with the package or surface you're touching when
  it scopes cleanly (`broker:`, `tools:`, `streaming:`, `auth:`,
  `cmd/server:`).
- One logical change per commit. Mechanical churn (gofmt, import
  reorders) goes in its own commit.

## Public surface discipline

The MCP tool / resource / prompt names and JSON-Schema input shapes
are a contract with every connected agent. Treat them like a public
API:

- Renames need a deprecation alias for at least one minor release.
- New required input fields on existing tools are a breaking change;
  add them as optional with a sensible default first.
- Output JSON shapes must match what the agent guide
  (`AGENT.md` / `system://agent-guide`) advertises. If the shape
  needs to change, update the guide in the same commit.

## Logging

The server uses the SDK's `logger.Logger` interface plus the
zerolog-backed implementation in `internal/lib/logging/zerolog`. Don't
import a concrete logger directly in tools or broker code — take a
`logger.Logger` and let the entrypoint wire the implementation.

## internal/lib/

The `internal/lib/` tree is vendored from the legacy Synthetix V4 API
service at extraction time. Treat it as a frozen snapshot:

- Don't edit it for cosmetic reasons.
- Bug fixes are fine but should be mirrored upstream.
- Long-term, packages here will be pruned (a lot of `internal/lib/`
  is dead code in the MCP server's actual runtime path).

## Reporting bugs

Open an issue with:
- a reproducer (env vars + a minimal MCP client trace)
- the exact `synthetix-mcp` git SHA you ran
- the SDK version (`go list -m github.com/synthetixio/synthetix-go`)

If the bug is security-sensitive, follow [`SECURITY.md`](./SECURITY.md)
instead.
