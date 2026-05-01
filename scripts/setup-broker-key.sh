#!/usr/bin/env bash
set -euo pipefail
trap 'stty echo 2>/dev/null || true' EXIT

SECRET_DIR="${SNXMCP_SECRET_DIR:-.secrets}"
KEY_FILE="${SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE:-${SECRET_DIR}/agent-broker-private-key}"
ENV_FILE="${SNXMCP_ENV_FILE:-config.env}"

mkdir -p "${SECRET_DIR}"
chmod 700 "${SECRET_DIR}"

cat <<'EOF'
Synthetix MCP broker key setup

Paste the delegated trading private key into this terminal prompt only.
Do not paste private keys into Claude, Cursor, ChatGPT, or any agent chat.

The key will be written to a local gitignored file with 0600 permissions.
EOF

printf 'Delegated private key: '
stty -echo
IFS= read -r key
stty echo
printf '\n'

key="$(printf '%s' "${key}" | tr -d '[:space:]')"
if [ -z "${key}" ]; then
  echo "ERROR: empty key" >&2
  exit 1
fi

case "${key}" in
  0x*) ;;
  *) key="0x${key}" ;;
esac

umask 077
printf '%s\n' "${key}" > "${KEY_FILE}"
chmod 600 "${KEY_FILE}"

touch "${ENV_FILE}"
if grep -q '^SNXMCP_AGENT_BROKER_ENABLED=' "${ENV_FILE}"; then
  tmp="$(mktemp)"
  sed 's|^SNXMCP_AGENT_BROKER_ENABLED=.*|SNXMCP_AGENT_BROKER_ENABLED=true|' "${ENV_FILE}" > "${tmp}"
  mv "${tmp}" "${ENV_FILE}"
else
  printf '\nSNXMCP_AGENT_BROKER_ENABLED=true\n' >> "${ENV_FILE}"
fi

if grep -q '^SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE=' "${ENV_FILE}"; then
  tmp="$(mktemp)"
  sed "s|^SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE=.*|SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE=${KEY_FILE}|" "${ENV_FILE}" > "${tmp}"
  mv "${tmp}" "${ENV_FILE}"
else
  printf '\nSNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE=%s\n' "${KEY_FILE}" >> "${ENV_FILE}"
fi

if grep -q '^SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX=' "${ENV_FILE}"; then
  tmp="$(mktemp)"
  sed 's|^SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX=.*|SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX=|' "${ENV_FILE}" > "${tmp}"
  mv "${tmp}" "${ENV_FILE}"
fi

cat <<EOF

Broker key saved to ${KEY_FILE}

Run broker mode with:

  set -a && source ${ENV_FILE} && set +a
  go run ./cmd/server

Or use:

  make run
EOF
