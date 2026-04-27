#!/usr/bin/env node
import { readFile } from "node:fs/promises";
import { stdin as input, stdout as output } from "node:process";
import readline from "node:readline/promises";
import { Wallet } from "ethers";

const DEFAULT_MCP_URL = "http://localhost:8096/mcp";
const PROTOCOL_VERSION = "2025-06-18";

function usage() {
  console.log(`Usage:
  node authenticate-external-wallet.mjs --subaccount-id <id> [options]

Options:
  --mcp-url <url>           MCP HTTP URL. Default: ${DEFAULT_MCP_URL}
  --session-id <id>         Existing Claude MCP session ID from get_session.
  --subaccount-id <id>      Required Synthetix subaccount ID.
  --private-key-file <path> Read wallet/delegate private key from a local file.
  --private-key-env <name>  Read private key from this env var. Default: SNXMCP_EXTERNAL_WALLET_PRIVATE_KEY_HEX
  --help                   Show this help.

Security:
  Do not paste private keys or signatures into Claude/Cursor/chat.
  Prefer --private-key-file or the hidden terminal prompt.
`);
}

function parseArgs(argv) {
  const args = {
    mcpUrl: process.env.SNXMCP_MCP_URL || DEFAULT_MCP_URL,
    privateKeyEnv: "SNXMCP_EXTERNAL_WALLET_PRIVATE_KEY_HEX",
  };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--help" || arg === "-h") {
      args.help = true;
      continue;
    }
    const next = argv[i + 1];
    switch (arg) {
      case "--mcp-url":
        args.mcpUrl = requiredValue(arg, next);
        i += 1;
        break;
      case "--session-id":
        args.sessionId = requiredValue(arg, next);
        i += 1;
        break;
      case "--subaccount-id":
        args.subaccountId = requiredValue(arg, next);
        i += 1;
        break;
      case "--private-key-file":
        args.privateKeyFile = requiredValue(arg, next);
        i += 1;
        break;
      case "--private-key-env":
        args.privateKeyEnv = requiredValue(arg, next);
        i += 1;
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }
  return args;
}

function requiredValue(flag, value) {
  if (!value || value.startsWith("--")) {
    throw new Error(`${flag} requires a value`);
  }
  return value;
}

async function readPrivateKey(args) {
  if (args.privateKeyFile) {
    return normalizePrivateKey(await readFile(args.privateKeyFile, "utf8"));
  }
  if (process.env[args.privateKeyEnv]) {
    return normalizePrivateKey(process.env[args.privateKeyEnv]);
  }

  const rl = readline.createInterface({ input, output });
  const wasRaw = input.isRaw;
  if (input.isTTY) input.setRawMode(true);
  let key = "";
  output.write("Delegated wallet private key (hidden): ");
  for await (const char of input) {
    const s = char.toString("utf8");
    if (s === "\r" || s === "\n") break;
    if (s === "\u0003") {
      output.write("\n");
      process.exit(130);
    }
    if (s === "\u007f") {
      key = key.slice(0, -1);
      continue;
    }
    key += s;
  }
  if (input.isTTY) input.setRawMode(wasRaw);
  output.write("\n");
  rl.close();
  return normalizePrivateKey(key);
}

function normalizePrivateKey(raw) {
  const trimmed = String(raw).trim();
  if (!trimmed) throw new Error("private key is empty");
  return trimmed.startsWith("0x") ? trimmed : `0x${trimmed}`;
}

class MCPHTTPClient {
  constructor(endpoint, sessionId) {
    this.endpoint = endpoint;
    this.sessionId = sessionId || "";
    this.nextID = 1;
  }

  async initializeIfNeeded() {
    if (this.sessionId) return;
    const response = await this.post({
      jsonrpc: "2.0",
      id: this.nextID++,
      method: "initialize",
      params: {
        protocolVersion: PROTOCOL_VERSION,
        capabilities: {},
        clientInfo: { name: "synthetix-mcp-external-wallet-auth", version: "0.1.0" },
      },
    });
    const sessionId = response.headers.get("mcp-session-id");
    if (!sessionId) throw new Error("MCP initialize did not return Mcp-Session-Id");
    this.sessionId = sessionId;
    await this.post({
      jsonrpc: "2.0",
      method: "notifications/initialized",
      params: {},
    }, { expectBody: false });
  }

  async callTool(name, args) {
    await this.initializeIfNeeded();
    const response = await this.post({
      jsonrpc: "2.0",
      id: this.nextID++,
      method: "tools/call",
      params: { name, arguments: args },
    });
    const payload = await parseMCPResponse(response);
    if (payload.error) {
      throw new Error(`${name} failed: ${payload.error.message || JSON.stringify(payload.error)}`);
    }
    const result = payload.result;
    if (result?.isError) {
      throw new Error(`${name} returned an MCP tool error: ${extractText(result)}`);
    }
    return result;
  }

  async post(body, { expectBody = true } = {}) {
    const headers = {
      "accept": "application/json, text/event-stream",
      "content-type": "application/json",
    };
    if (this.sessionId) headers["mcp-session-id"] = this.sessionId;

    const response = await fetch(this.endpoint, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status} from MCP server: ${await response.text()}`);
    }
    if (!expectBody) return response;
    return response;
  }
}

async function parseMCPResponse(response) {
  const text = await response.text();
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

function typedDataForEthers(typedData) {
  const { domain, types, primaryType, message } = typedData;
  const cleanTypes = { ...types };
  delete cleanTypes.EIP712Domain;
  return { domain, types: cleanTypes, primaryType, message };
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) {
    usage();
    return;
  }
  if (!args.subaccountId) {
    throw new Error("--subaccount-id is required");
  }

  const privateKey = await readPrivateKey(args);
  const wallet = new Wallet(privateKey);
  const client = new MCPHTTPClient(args.mcpUrl, args.sessionId);

  const preview = extractStructured(await client.callTool("preview_auth_message", {
    subAccountId: args.subaccountId,
  }));
  const typedData = typedDataForEthers(preview.typedData);
  const signatureHex = await wallet.signTypedData(typedData.domain, typedData.types, typedData.message);

  const auth = extractStructured(await client.callTool("authenticate", {
    message: JSON.stringify(preview.typedData),
    signatureHex,
  }));

  console.log("authentication complete");
  console.log(`sessionId: ${auth.sessionId || client.sessionId}`);
  console.log(`walletAddress: ${auth.walletAddress || wallet.address}`);
  console.log(`subAccountId: ${auth.subAccountId || args.subaccountId}`);
  console.log("");
  console.log("In Claude, call get_session to confirm authMode='authenticated'.");
}

main().catch((err) => {
  console.error(`ERROR: ${err.message}`);
  process.exit(1);
});
