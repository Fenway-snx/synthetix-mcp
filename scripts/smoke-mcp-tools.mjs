#!/usr/bin/env node

const DEFAULT_MCP_URL = "http://localhost:8096/mcp";
const PROTOCOL_VERSION = "2025-06-18";

function usage() {
  console.log(`Usage: node scripts/smoke-mcp-tools.mjs [options]

Options:
  --url <url>              MCP HTTP endpoint (default: ${DEFAULT_MCP_URL})
  --symbol <symbol>        Market symbol to exercise (default: BTC-USDT)
  --wallet-address <addr>  Also test lookup_subaccount for this address
  --skip-streaming         Skip subscribe/unsubscribe smoke checks
  --timeout-ms <ms>        Per-request timeout (default: 15000)
  --list-only              Only test initialize + tools/resources/prompts listing
  --help                   Show this help

The script never places, cancels, or modifies live orders.`);
}

function parseArgs(argv) {
  const args = {
    listOnly: false,
    mcpUrl: DEFAULT_MCP_URL,
    skipStreaming: false,
    symbol: "BTC-USDT",
    timeoutMs: 15_000,
    walletAddress: "",
  };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case "--help":
      case "-h":
        args.help = true;
        break;
      case "--url":
        args.mcpUrl = requireValue(argv, ++i, arg);
        break;
      case "--symbol":
        args.symbol = requireValue(argv, ++i, arg);
        break;
      case "--wallet-address":
        args.walletAddress = requireValue(argv, ++i, arg);
        break;
      case "--skip-streaming":
        args.skipStreaming = true;
        break;
      case "--timeout-ms":
        args.timeoutMs = Number(requireValue(argv, ++i, arg));
        if (!Number.isFinite(args.timeoutMs) || args.timeoutMs <= 0) {
          throw new Error("--timeout-ms must be a positive number");
        }
        break;
      case "--list-only":
        args.listOnly = true;
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }
  return args;
}

function requireValue(argv, index, flag) {
  const value = argv[index];
  if (!value || value.startsWith("--")) {
    throw new Error(`${flag} requires a value`);
  }
  return value;
}

class MCPHTTPClient {
  constructor(endpoint, timeoutMs) {
    this.endpoint = endpoint;
    this.timeoutMs = timeoutMs;
    this.sessionId = "";
    this.nextID = 1;
  }

  async initialize() {
    const response = await this.post({
      jsonrpc: "2.0",
      id: this.nextID++,
      method: "initialize",
      params: {
        protocolVersion: PROTOCOL_VERSION,
        capabilities: {},
        clientInfo: { name: "synthetix-mcp-smoke", version: "0.1.0" },
      },
    });
    const payload = await parseMCPResponse(response);
    if (payload.error) {
      throw new Error(`initialize failed: ${payload.error.message || JSON.stringify(payload.error)}`);
    }
    const sessionId = response.headers.get("mcp-session-id");
    if (!sessionId) throw new Error("initialize did not return Mcp-Session-Id");
    this.sessionId = sessionId;
    await this.post({
      jsonrpc: "2.0",
      method: "notifications/initialized",
      params: {},
    }, { expectBody: false });
    return payload.result || {};
  }

  async call(method, params = {}) {
    const response = await this.post({
      jsonrpc: "2.0",
      id: this.nextID++,
      method,
      params,
    });
    const payload = await parseMCPResponse(response);
    if (payload.error) {
      throw new Error(`${method} failed: ${payload.error.message || JSON.stringify(payload.error)}`);
    }
    return payload.result;
  }

  async callTool(name, args = {}) {
    const result = await this.call("tools/call", { name, arguments: args });
    if (result?.isError) {
      throw new Error(`${name} returned a tool error: ${extractText(result)}`);
    }
    return extractStructured(result);
  }

  async readResource(uri) {
    return this.call("resources/read", { uri });
  }

  async post(body, { expectBody = true } = {}) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs);
    try {
      const headers = {
        accept: "application/json, text/event-stream",
        "content-type": "application/json",
      };
      if (this.sessionId) headers["mcp-session-id"] = this.sessionId;
      const response = await fetch(this.endpoint, {
        method: "POST",
        headers,
        body: JSON.stringify(body),
        signal: controller.signal,
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status} from MCP server: ${await response.text()}`);
      }
      if (!expectBody) return response;
      return response;
    } finally {
      clearTimeout(timeout);
    }
  }
}

async function parseMCPResponse(response) {
  const text = await response.text();
  if (!text.trim()) return {};
  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("text/event-stream")) {
    const data = text
      .split(/\r?\n/)
      .filter((line) => line.startsWith("data:"))
      .map((line) => line.slice("data:".length).trim())
      .join("\n");
    if (!data) throw new Error(`empty SSE response: ${text}`);
    return JSON.parse(data);
  }
  return JSON.parse(text);
}

function extractStructured(result) {
  if (result?.structuredContent) return result.structuredContent;
  const text = extractText(result);
  if (!text) return {};
  return JSON.parse(text);
}

function extractText(result) {
  return (result?.content || [])
    .filter((item) => item.type === "text")
    .map((item) => item.text)
    .join("\n");
}

function extractResourceText(result) {
  const contents = result?.contents || [];
  return contents
    .map((item) => item.text || item.blob || "")
    .filter(Boolean)
    .join("\n");
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function hasAnyKey(value, keys) {
  return value && keys.some((key) => Object.prototype.hasOwnProperty.call(value, key));
}

function logStep(message) {
  console.log(`- ${message}`);
}

function logOK(message) {
  console.log(`  ok: ${message}`);
}

async function runCheck(name, fn) {
  logStep(name);
  const started = Date.now();
  const detail = await fn();
  const elapsed = Date.now() - started;
  logOK(`${detail || "passed"} (${elapsed}ms)`);
}

function deriveHealthURL(mcpUrl) {
  const url = new URL(mcpUrl);
  url.pathname = "/health/ready";
  url.search = "";
  url.hash = "";
  return url.toString();
}

async function checkHealth(mcpUrl, timeoutMs) {
  const healthURL = deriveHealthURL(mcpUrl);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(healthURL, { signal: controller.signal });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
  } finally {
    clearTimeout(timeout);
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) {
    usage();
    return;
  }

  console.log("Synthetix MCP comprehensive smoke test");
  console.log(`endpoint: ${args.mcpUrl}`);
  console.log(`symbol:   ${args.symbol}`);
  console.log("");

  await runCheck("health endpoint", async () => {
    await checkHealth(args.mcpUrl, args.timeoutMs);
    return "ready";
  });

  const client = new MCPHTTPClient(args.mcpUrl, args.timeoutMs);
  await runCheck("initialize MCP session", async () => {
    await client.initialize();
    return `session ${client.sessionId}`;
  });

  let toolNames = new Set();
  let resourceURIs = new Set();
  let promptNames = new Set();

  await runCheck("list tools/resources/prompts", async () => {
    const [tools, resources, prompts] = await Promise.all([
      client.call("tools/list"),
      client.call("resources/list"),
      client.call("prompts/list"),
    ]);
    toolNames = new Set((tools.tools || []).map((tool) => tool.name));
    resourceURIs = new Set((resources.resources || []).map((resource) => resource.uri));
    promptNames = new Set((prompts.prompts || []).map((prompt) => prompt.name));

    for (const name of [
      "ping",
      "get_system_health",
      "get_auth_status",
      "get_server_info",
      "get_context",
      "list_markets",
      "get_market_summary",
      "get_orderbook",
      "get_recent_trades",
      "get_funding_rate",
      "get_candles",
      "get_session",
      "subscribe",
      "unsubscribe",
    ]) {
      assert(toolNames.has(name), `missing expected tool: ${name}`);
    }
    for (const uri of [
      "system://agent-guide",
      "system://server-info",
      "system://routing-guide",
      "system://status",
      "account://risk-limits",
    ]) {
      assert(resourceURIs.has(uri), `missing expected resource: ${uri}`);
    }
    for (const name of ["quickstart", "startup-validation", "pre-trade-checklist"]) {
      assert(promptNames.has(name), `missing expected prompt: ${name}`);
    }
    return `${toolNames.size} tools, ${resourceURIs.size} resources, ${promptNames.size} prompts`;
  });

  if (args.listOnly) {
    console.log("");
    console.log("Smoke test passed.");
    return;
  }

  let serverInfo = {};
  await runCheck("call ping/get_system_health/get_auth_status/get_server_info", async () => {
    const [ping, systemHealth, authStatus, info] = await Promise.all([
      client.callTool("ping"),
      client.callTool("get_system_health"),
      client.callTool("get_auth_status"),
      client.callTool("get_server_info"),
    ]);
    assert(ping.ok === true, "ping did not return ok=true");
    assert(hasAnyKey(systemHealth, ["status", "rest", "websocket", "auth"]), "system health payload looked empty");
    assert(hasAnyKey(authStatus, ["authMode", "authenticated", "status"]), "auth status payload looked empty");
    assert(info.serverName || info.version || info.agentBroker, "server info payload looked empty");
    serverInfo = info;
    return `broker enabled=${Boolean(info.agentBroker?.enabled)}`;
  });

  await runCheck("call get_context/get_session", async () => {
    const [context, session] = await Promise.all([
      client.callTool("get_context"),
      client.callTool("get_session"),
    ]);
    assert(context.server || context.capabilities || context._meta, "context payload looked empty");
    assert(session.sessionId === client.sessionId, "get_session did not echo the MCP session ID");
    return `authMode=${session.authMode || "unknown"}`;
  });

  await runCheck("read core resources", async () => {
    for (const uri of [
      "system://routing-guide",
      "system://server-info",
      "system://status",
      "account://risk-limits",
    ]) {
      const result = await client.readResource(uri);
      const text = extractResourceText(result);
      assert(text.length > 0, `${uri} returned empty content`);
    }
    const guide = await client.readResource("system://agent-guide");
    assert(extractResourceText(guide).includes("Synthetix"), "agent guide content looked wrong");
    return "resources readable";
  });

  await runCheck("call list_markets", async () => {
    const result = await client.callTool("list_markets", { status: "open" });
    assert(Array.isArray(result.markets), "list_markets did not return markets array");
    assert(result.markets.length > 0, "list_markets returned no markets");
    return `${result.markets.length} open markets`;
  });

  await runCheck(`call market data tools for ${args.symbol}`, async () => {
    const [summary, orderbook, trades, funding, candles] = await Promise.all([
      client.callTool("get_market_summary", { symbol: args.symbol }),
      client.callTool("get_orderbook", { symbol: args.symbol, limit: 5 }),
      client.callTool("get_recent_trades", { symbol: args.symbol, limit: 5 }),
      client.callTool("get_funding_rate", { symbol: args.symbol }),
      client.callTool("get_candles", { symbol: args.symbol, timeframe: "1m", limit: 5 }),
    ]);
    assert(summary.market || summary.prices || summary.summary, "market summary payload looked empty");
    assert(Array.isArray(orderbook.bids) && Array.isArray(orderbook.asks), "orderbook did not return bid/ask arrays");
    assert(Array.isArray(trades.trades), "recent trades did not return trades array");
    assert(
      funding.fundingRateEntry || funding.fundingRate || funding.estimatedFundingRate !== undefined,
      "funding rate payload looked empty",
    );
    assert(Array.isArray(candles.candles), "candles did not return candles array");
    return "summary, orderbook, trades, funding, candles";
  });

  await runCheck(`read market specs for ${args.symbol}`, async () => {
    const result = await client.readResource(`market://specs/${encodeURIComponent(args.symbol)}`);
    const text = extractResourceText(result);
    assert(text.includes(args.symbol), "market specs did not include requested symbol");
    return "market specs readable";
  });

  if (args.walletAddress) {
    await runCheck("call lookup_subaccount", async () => {
      const result = await client.callTool("lookup_subaccount", {
        walletAddress: args.walletAddress,
        includeDelegations: true,
      });
      assert(hasAnyKey(result, ["ownedSubAccountIds", "delegatedSubAccountIds", "subAccountIds"]), "lookup payload looked empty");
      return "lookup_subaccount returned";
    });
  }

  if (!args.skipStreaming) {
    await runCheck("subscribe/unsubscribe smoke", async () => {
      const sub = await client.callTool("subscribe", {
        subscriptions: [
          { channel: "trades", params: { symbol: args.symbol } },
          { channel: "orderbook", params: { symbol: args.symbol } },
        ],
      });
      assert(Array.isArray(sub.activeSubscriptions), "subscribe did not return activeSubscriptions");
      const unsub = await client.callTool("unsubscribe", {
        channels: ["trades", "orderbook"],
        symbol: args.symbol,
      });
      assert(Array.isArray(unsub.removedSubscriptions), "unsubscribe did not return removedSubscriptions");
      return `${sub.activeSubscriptions.length} active, ${unsub.removedSubscriptions.length} removed`;
    });
  }

  await runCheck("validate broker/external-wallet routing", async () => {
    const brokerEnabled = Boolean(serverInfo.agentBroker?.enabled);
    if (brokerEnabled) {
      for (const name of ["place_order", "cancel_order", "cancel_all_orders", "close_position"]) {
        assert(toolNames.has(name), `broker mode missing canonical tool: ${name}`);
      }
      for (const name of [
        "signed_place_order",
        "signed_cancel_order",
        "signed_cancel_all_orders",
        "signed_close_position",
        "preview_trade_signature",
      ]) {
        assert(!toolNames.has(name), `broker mode should hide external-wallet tool: ${name}`);
      }
      return "broker tool routing is canonical";
    }

    for (const name of ["preview_trade_signature", "signed_place_order", "signed_cancel_order"]) {
      assert(toolNames.has(name), `external-wallet mode missing signed tool: ${name}`);
    }
    assert(!toolNames.has("place_order"), "external-wallet mode should not expose place_order");
    return "external-wallet routing is explicit";
  });

  console.log("");
  console.log("Smoke test passed.");
}

main().catch((err) => {
  console.error("");
  console.error(`Smoke test failed: ${err.message}`);
  process.exit(1);
});
