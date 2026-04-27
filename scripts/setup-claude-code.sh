#!/usr/bin/env bash
set -euo pipefail

NAME="${SNXMCP_CLAUDE_NAME:-synthetix-dex-mcp}"
URL="${SNXMCP_MCP_URL:-http://localhost:8096/mcp}"
HEALTH_URL="${SNXMCP_HEALTH_URL:-${URL%/mcp}/health/ready}"
ENV_FILE="${SNXMCP_ENV_FILE:-config.env}"
LOG_FILE="${SNXMCP_SETUP_LOG_FILE:-.synthetix-mcp-server.log}"
NO_START=false
DRY_RUN=false
VERIFY=false

say() {
  printf '%s\n' "$*"
}

fail() {
  say "ERROR: $*" >&2
  exit 1
}

usage() {
  cat <<EOF
Usage: $0 [verify] [--verify] [--no-start] [--dry-run] [--help]

Options:
  verify       Check health and try a non-interactive Claude ping when supported.
  --verify    Same as the verify sub-command.
  --no-start   Do not start the MCP server if it is not already reachable.
  --dry-run    Print the actions that would be taken, without changing anything.
  --help       Show this help text.

Environment:
  SNXMCP_CLAUDE_NAME       Claude Code MCP server name. Default: ${NAME}
  SNXMCP_MCP_URL           MCP URL. Default: ${URL}
  SNXMCP_HEALTH_URL        Health URL. Default: ${HEALTH_URL}
  SNXMCP_ENV_FILE          Env file to source before starting. Default: ${ENV_FILE}
  SNXMCP_SETUP_LOG_FILE    Background server log. Default: ${LOG_FILE}
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    verify|--verify)
      VERIFY=true
      ;;
    --no-start)
      NO_START=true
      ;;
    --dry-run)
      DRY_RUN=true
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
  shift
done

run() {
  if [ "${DRY_RUN}" = true ]; then
    say "DRY RUN: $*"
    return 0
  fi
  "$@"
}

health_ok() {
  curl -fsS "${HEALTH_URL}" >/dev/null 2>&1
}

source_env_file() {
  if [ -f "${ENV_FILE}" ]; then
    # shellcheck disable=SC1090
    set -a
    case "${ENV_FILE}" in
      /*|*/*) . "${ENV_FILE}" ;;
      *) . "./${ENV_FILE}" ;;
    esac
    set +a
  fi
}

server_args() {
  if [ -n "${SNXMCP_SETUP_SERVER_ARGS:-}" ]; then
    printf '%s\n' "${SNXMCP_SETUP_SERVER_ARGS}"
    return 0
  fi
  if [ -n "${SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX:-}" ] || [ -n "${SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE:-}" ]; then
    printf '%s\n' ""
    return 0
  fi
  printf '%s\n' "--no-broker"
}

session_opener() {
  cat <<EOF
You have MCP tools available from ${NAME}. Before doing anything else, call ping to verify the connection. If ping fails or you see "unknown tool", stop, do not use Bash as a fallback, and tell me to restart Claude Code and run: claude mcp list
EOF
}

claude_supports_print() {
  claude --help 2>&1 | grep -Eq -- '(^|[[:space:]])--print([,=[:space:]]|$)|(^|[[:space:]])-p([,[:space:]]|$)'
}

verify_setup() {
  say "Verification"
  say
  if [ "${DRY_RUN}" = true ]; then
    say "DRY RUN: would check MCP health: curl -fsS ${HEALTH_URL}"
  elif health_ok; then
    say "OK: MCP health endpoint is reachable."
  else
    fail "MCP health endpoint is not reachable: ${HEALTH_URL}"
  fi

  if ! command -v claude >/dev/null 2>&1; then
    say "Claude Code CLI was not found on PATH; skipping Claude ping verification."
    return 0
  fi

  if ! claude_supports_print; then
    say "Claude Code CLI does not appear to support non-interactive --print/-p mode; skipping Claude ping verification."
    say "Manual verification prompt:"
    say
    session_opener
    return 0
  fi

  prompt="$(session_opener)"
  say "Attempting non-interactive Claude ping verification..."
  if [ "${DRY_RUN}" = true ]; then
    say "DRY RUN: claude --print \"${prompt}\""
    return 0
  fi

  if output="$(claude --print "${prompt}" 2>&1)"; then
    say "${output}"
    if printf '%s\n' "${output}" | grep -Eiq 'unknown tool|not connected|no mcp|bash'; then
      fail "Claude response suggests the MCP connection is not active."
    fi
    say "OK: Claude non-interactive command completed. Confirm above that ping was called successfully."
    return 0
  fi

  say "${output}" >&2
  fail "Claude non-interactive ping failed."
}

start_server() {
  source_env_file
  args="$(server_args)"
  if [ -z "${args}" ]; then
    start_cmd="go run ./cmd/server"
  else
    start_cmd="go run ./cmd/server ${args}"
  fi

  say "MCP server is not reachable; starting it in the background..."
  say "Command: ${start_cmd}"
  say "Log file: ${LOG_FILE}"

  if [ "${DRY_RUN}" = true ]; then
    say "DRY RUN: ${start_cmd} > ${LOG_FILE} 2>&1 &"
    return 0
  fi

  # shellcheck disable=SC2086
  nohup sh -c "${start_cmd}" > "${LOG_FILE}" 2>&1 &
  server_pid=$!
  say "Started MCP server process ${server_pid}."

  for _ in 1 2 3 4 5 6 7 8 9 10; do
    if health_ok; then
      return 0
    fi
    sleep 1
  done

  cat >&2 <<EOF
MCP server did not become ready after 10 seconds.

Check the server log:

  ${LOG_FILE}

Common fixes:
  - If broker mode needs a key, run: ./scripts/setup-broker-key.sh
  - For read-only setup, run: $0 --no-start after starting go run ./cmd/server --no-broker
  - If port 8096 is busy, stop the old process or set SNXMCP_SERVER_ADDRESS.
EOF
  exit 1
}

print_checklist() {
  cat <<EOF

Post-registration checklist

1. Verify Claude Code knows about the server:

   claude mcp list

2. Restart Claude Code if this server was just added.

3. Type this exact prompt:

   $(session_opener)

4. Then try a simple public-data tool:

   Use the ${NAME} MCP. Call ping, then get_server_info, then list_markets, and tell me the top 3 markets by 24h volume.

Common first errors and fixes:

- Claude asks to run Bash instead of using MCP tools:
  Restart Claude Code, run 'claude mcp list', then retry the prompt above.

- Server is not reachable:
  Run 'curl -fsS ${HEALTH_URL}' and check ${LOG_FILE}.

- Broker key missing:
  Run './scripts/setup-broker-key.sh' in a terminal, never in an agent chat.
  For read-only setup, start with 'go run ./cmd/server --no-broker'.

- Address already in use:
  Another server is already on 8096. Stop it or set SNXMCP_SERVER_ADDRESS.
EOF
}

say "Synthetix MCP Claude Code setup"
say "Server name: ${NAME}"
say "MCP URL:     ${URL}"
say "Health URL:  ${HEALTH_URL}"
say "Env file:    ${ENV_FILE}"
say "Dry run:     ${DRY_RUN}"
say "Verify:      ${VERIFY}"
say

if ! command -v curl >/dev/null 2>&1; then
  fail "curl is required to check the local MCP server."
fi

if [ "${DRY_RUN}" = true ]; then
  say "DRY RUN: would check server health with curl -fsS ${HEALTH_URL}"
elif ! health_ok; then
  if [ "${NO_START}" = true ]; then
    cat >&2 <<EOF
The MCP server is not reachable and --no-start was set.

Start it first:

  go run ./cmd/server --no-broker

or set up broker mode safely:

  ./scripts/setup-broker-key.sh
  set -a && source ${ENV_FILE} && set +a
  go run ./cmd/server
EOF
    print_checklist
    exit 1
  fi
  start_server
fi

if [ "${VERIFY}" = true ]; then
  verify_setup
  exit 0
fi

if [ "${DRY_RUN}" = true ]; then
  say "DRY RUN: assuming server health for registration preview."
else
  say "OK: MCP server is reachable."
fi
say

if ! command -v claude >/dev/null 2>&1; then
  cat <<EOF
Claude Code CLI was not found on PATH.

Add this MCP server manually in Claude Code:

  name: ${NAME}
  url:  ${URL}

Then restart Claude Code and ask:

  $(session_opener)
EOF
  print_checklist
  exit 0
fi

if [ "${DRY_RUN}" = true ]; then
  say "DRY RUN: would run claude mcp list"
  say "DRY RUN: would add server if missing:"
  say "DRY RUN: claude mcp add --transport http ${NAME} ${URL}"
elif claude mcp list 2>/dev/null | grep -Fq "${NAME}"; then
  say "OK: Claude Code already has an MCP server named ${NAME}."
else
  say "Registering ${NAME} with Claude Code..."
  if run claude mcp add --transport http "${NAME}" "${URL}"; then
    say "OK: registered with Claude Code."
  else
    cat >&2 <<EOF
Automatic registration failed. Your Claude Code version may use a different
MCP add syntax.

Try one of these manually:

  claude mcp add --transport http ${NAME} ${URL}
  claude mcp add ${NAME} ${URL}

Then run:

  claude mcp list
EOF
    exit 1
  fi
fi

say
say "Current Claude Code MCP servers:"
if [ "${DRY_RUN}" = true ]; then
  say "DRY RUN: claude mcp list"
else
  claude mcp list || true
fi

print_checklist
