# Node Sidecar Scripts

These scripts are for users who intentionally run Synthetix MCP in
external-wallet mode:

```bash
go run ./cmd/server --no-broker
```

Claude cannot sign EIP-712 payloads by itself, and users should never paste
private keys, typed data, digests, or signatures into agent chat. The sidecar
runs in a normal terminal, signs locally, and calls MCP over localhost.

## Authenticate A Claude Session

1. In Claude, ask:

```text
Use the synthetix-dex-mcp MCP server and call get_session. Show me only the sessionId.
```

2. In this directory:

```bash
npm install
node authenticate-external-wallet.mjs \
  --session-id <SESSION_ID_FROM_CLAUDE> \
  --subaccount-id <SUBACCOUNT_ID> \
  --private-key-file ~/.synthetix/delegate-key
```

If you omit `--private-key-file`, the script uses
`SNXMCP_EXTERNAL_WALLET_PRIVATE_KEY_HEX` when set, otherwise it prompts for the
key with hidden terminal input.

3. In Claude, ask:

```text
Use the synthetix-dex-mcp MCP server and call get_session. Confirm authMode is authenticated.
```

## Standalone Mode

If you omit `--session-id`, the script creates and authenticates its own MCP
session. That is useful for direct scripting, but it does not authenticate
Claude's separate MCP session.
