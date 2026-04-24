package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	sdkeip712 "github.com/synthetixio/synthetix-go/eip712"
)

// Minimum surface the preview tools need from the auth manager: the
// canonical EIP-712 domain values used during validation. Kept narrow
// so tests can stub it.
type tradePreviewDomainProvider interface {
	DomainName() string
	DomainVersion() string
	ChainID() int
}

// Inputs to fetch the EIP-712 typed-data the client signs to produce
// the `authenticate` tool's `message` argument. Kept minimal so agents
// without EIP-712 expertise can call this first.
type previewAuthMessageInput struct {
	SubAccountID FlexInt64 `json:"subAccountId" jsonschema:"Target subaccount ID. The returned message authorizes binding the current MCP session to this subaccount."`
	Timestamp    int64     `json:"timestamp,omitempty" jsonschema:"UNIX timestamp in seconds that will be embedded in the AuthMessage. Omit to use the current server time."`
}

// Describes the write action a client intends to sign. Exactly one of
// the action sub-objects must be populated; the tool rejects ambiguous
// inputs. All 64-bit IDs accept either JSON strings or numbers.
type previewTradeSignatureInput struct {
	Action        string                         `json:"action" jsonschema:"One of: placeOrders, modifyOrder, cancelOrders, cancelAllOrders, closePosition."`
	SubAccountID  FlexInt64                      `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Nonce         int64                          `json:"nonce,omitempty" jsonschema:"Unique nonce for this action. Omit to have the server generate a fresh monotonic nonce (current UTC ms)."`
	ExpiresAfter  int64                          `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Must be strictly greater than nonce. Omit to default to nonce + 24h."`
	PlaceOrder    *previewOrderInput             `json:"placeOrder,omitempty" jsonschema:"Populate for action=placeOrders. Same shape as the place_order tool's order fields (symbol, side, type, quantity, price, timeInForce, etc.)."`
	ModifyOrder   *previewTradeModifyInput       `json:"modifyOrder,omitempty" jsonschema:"Populate for action=modifyOrder."`
	CancelOrder   *previewTradeCancelInput       `json:"cancelOrder,omitempty" jsonschema:"Populate for action=cancelOrders (single order)."`
	CancelAll     *previewTradeCancelAllInput    `json:"cancelAllOrders,omitempty" jsonschema:"Populate for action=cancelAllOrders."`
	ClosePosition *previewTradeClosePosInput     `json:"closePosition,omitempty" jsonschema:"Populate for action=closePosition. The server normalizes it into a placeOrders action under the hood."`
}

type previewTradeModifyInput struct {
	VenueOrderID  string `json:"venueOrderId,omitempty" jsonschema:"Existing venue order ID to modify. Mutually exclusive with clientOrderId."`
	ClientOrderID string `json:"clientOrderId,omitempty" jsonschema:"Existing client-supplied order ID to modify. Mutually exclusive with venueOrderId."`
	Price         string `json:"price,omitempty" jsonschema:"New limit price as a decimal string."`
	Quantity      string `json:"quantity,omitempty" jsonschema:"New order quantity as a decimal string."`
	TriggerPrice  string `json:"triggerPrice,omitempty" jsonschema:"New trigger price for conditional orders."`
}

type previewTradeCancelInput struct {
	VenueOrderID  string `json:"venueOrderId,omitempty" jsonschema:"Venue order ID to cancel. Mutually exclusive with clientOrderId."`
	ClientOrderID string `json:"clientOrderId,omitempty" jsonschema:"Client-supplied order ID to cancel. Mutually exclusive with venueOrderId."`
}

type previewTradeCancelAllInput struct {
	Symbol string `json:"symbol,omitempty" jsonschema:"Restrict cancellation to this market symbol. Omit to cancel across all markets."`
}

type previewTradeClosePosInput struct {
	Symbol     string `json:"symbol" jsonschema:"Market symbol of the position to close, e.g. BTC-USDT."`
	Side       string `json:"side,omitempty" jsonschema:"Optional side of the existing position: 'long' or 'short'. When provided, the preview skips the live getPositions pre-flight and uses the caller-supplied side + quantity to build the typed data. Required in deployments without a broker-signed positions read. Must match the value close_position will be called with."`
	Quantity   string `json:"quantity,omitempty" jsonschema:"Quantity to close. Required when 'side' is provided. When omitted and 'side' is also omitted, the server reads the live position and defaults to the full open quantity, mirroring close_position."`
	Method     string `json:"method,omitempty" jsonschema:"Close method: 'market' (default) or 'limit'."`
	LimitPrice string `json:"limitPrice,omitempty" jsonschema:"Limit price when method=limit."`
}

// Everything a wallet needs to produce the signature payload for the
// matching write tool. typedData is canonical EIP-712 JSON for
// viem/ethers/Web3.py; digest is the precomputed keccak256 hash for
// hardware signers. Nonce and expiresAfter echo back (possibly
// server-populated) for verbatim reuse on the write call.
type previewSignatureOutput struct {
	Meta         responseMeta   `json:"_meta"`
	Action       string         `json:"action"`
	PrimaryType  string         `json:"primaryType"`
	Nonce        int64          `json:"nonce"`
	ExpiresAfter int64          `json:"expiresAfter"`
	SubAccountID int64          `json:"subAccountId,string"`
	TypedData    map[string]any `json:"typedData"`
	Digest       string         `json:"digest"`
	Notes        []string       `json:"notes"`
}

type previewAuthOutput struct {
	Meta         responseMeta   `json:"_meta"`
	PrimaryType  string         `json:"primaryType"`
	SubAccountID int64          `json:"subAccountId,string"`
	Timestamp    int64          `json:"timestamp"`
	Action       string         `json:"action"`
	TypedData    map[string]any `json:"typedData"`
	Digest       string         `json:"digest"`
	Notes        []string       `json:"notes"`
}

// Registers the two onboarding tools. preview_auth_message is public so
// brand-new agents can bootstrap; preview_trade_signature requires an
// authenticated session for the subaccount context that closePosition
// and similar actions need.
func RegisterSignaturePreviewTools(
	server *mcp.Server,
	deps *ToolDeps,
	domain tradePreviewDomainProvider,
	tradeReads *TradeReadClient,
) {
	addPublicTool(server, deps, &mcp.Tool{
		Name: "preview_auth_message",
		Description: "Return the exact EIP-712 typed-data object a client should sign to produce the `message` argument of the `authenticate` tool. Agents should use this instead of hand-crafting EIP-712 payloads: sign `typedData` with any standard EIP-712 signer (viem.signTypedData, eth_signTypedData_v4, ethers Wallet._signTypedData, Web3.py sign_typed_data), then pass the serialized JSON back to `authenticate` along with the 0x-prefixed 65-byte signature hex. AGENT POLICY: only call this when you (the agent) hold a private key locally. If get_server_info.agentBroker.enabled=true, prefer the quick_* tools — the broker signs server-side and skips this whole flow. Never dump the returned typedData into chat and ask a human to sign it.",
	}, func(ctx context.Context, tc ToolContext, input previewAuthMessageInput) (*mcp.CallToolResult, previewAuthOutput, error) {
		subAccountID := input.SubAccountID.Int64()
		if subAccountID <= 0 {
			return toolErrorResponse[previewAuthOutput](fmt.Errorf("subAccountId is required and must be positive"))
		}
		ts := input.Timestamp
		if ts <= 0 {
			ts = nowUnixSeconds()
		}
		typedData := sdkeip712.BuildAuthMessageWithDomain(
			uint64(subAccountID),
			ts,
			sdkeip712.ActionWebSocketAuth,
			resolvePreviewDomain(domain),
		)
		serialized, err := typedDataToMap(typedData)
		if err != nil {
			return toolErrorResponse[previewAuthOutput](err)
		}
		digest, err := eip712DigestHex(typedData)
		if err != nil {
			return toolErrorResponse[previewAuthOutput](err)
		}
		return nil, previewAuthOutput{
			Meta:         newResponseMeta(authModeFromContext(tc)),
			PrimaryType:  "AuthMessage",
			SubAccountID: subAccountID,
			Timestamp:    ts,
			Action:       sdkeip712.ActionWebSocketAuth,
			TypedData:    serialized,
			Digest:       digest,
			Notes: []string{
				"Sign typedData with any EIP-712-capable signer; serialize the same typedData object to JSON and pass it as `authenticate.message`.",
				"digest is the keccak256 EIP-712 hash (for hardware/offline signers); normal EIP-712 signers can ignore it.",
				"timestamp is in seconds since epoch; the server enforces a bounded skew window on the AuthMessage.",
			},
		}, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name: "preview_trade_signature",
		Description: "Return the canonical EIP-712 typed-data object a client should sign to authorize one trade action (placeOrders, modifyOrder, cancelOrders, cancelAllOrders, or closePosition). The returned nonce and expiresAfter MUST be passed back verbatim to the corresponding write tool along with the produced signature. This is the recommended way to obtain trade signatures — constructing the typed-data by hand is error-prone because field order, type coercions, and normalization (e.g. placeOrders order field set) must match the server byte-for-byte. AGENT POLICY: only call this when you (the agent) hold a private key locally. If get_server_info.agentBroker.enabled=true, call quick_place_order / quick_close_position / quick_cancel_order / quick_cancel_all_orders instead — the broker generates and signs the payload internally in one round trip. Never paste the returned typedData into chat for a human to sign.",
	}, func(in previewTradeSignatureInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input previewTradeSignatureInput) (*mcp.CallToolResult, previewSignatureOutput, error) {
			subAccountID := input.SubAccountID.Int64()
			if subAccountID == 0 {
				subAccountID = tc.State.SubAccountID
			}
			if subAccountID <= 0 {
				return toolErrorResponse[previewSignatureOutput](fmt.Errorf("subAccountId unavailable: authenticate first or pass subAccountId explicitly"))
			}

			nonce := input.Nonce
			if nonce <= 0 {
				nonce = nowUnixMillis()
			}
			expiresAfter := input.ExpiresAfter
			if expiresAfter <= 0 {
				expiresAfter = nonce + int64(24*60*60*1000)
			}
			if expiresAfter <= nonce {
				return toolErrorResponse[previewSignatureOutput](fmt.Errorf("expiresAfter must be strictly greater than nonce"))
			}

			payload, requestAction, err := buildTradePreviewPayload(ctx, tradeReads, tc, subAccountID, input)
			if err != nil {
				return toolErrorResponse[previewSignatureOutput](err)
			}

			domainName, domainVersion, chainID := resolvePreviewDomainTriple(domain)

			// CreateTradeTypedData is the validation-payload-aware
			// bridge between lib/api/validation types and EIP-712.
			// The pure-crypto path (digest, serialize, AuthMessage)
			// has been migrated to sdk/eip712; trade typed-data
			// construction stays here because it introspects rich
			// Validated*Action structs from lib/api/validation that
			// the SDK deliberately doesn't depend on. The
			// sdkparity_test package pins byte-for-byte digest
			// agreement between this path and the SDK builders.
			typedData, err := snx_lib_auth.CreateTradeTypedData(
				snx_lib_auth.SubAccountId(strconv.FormatInt(subAccountID, 10)),
				snx_lib_auth.Nonce(nonce),
				expiresAfter,
				requestAction,
				payload,
				domainName,
				domainVersion,
				chainID,
			)
			if err != nil {
				return toolErrorResponse[previewSignatureOutput](err)
			}

			serialized, err := typedDataToMap(typedData)
			if err != nil {
				return toolErrorResponse[previewSignatureOutput](err)
			}
			digest, err := eip712DigestHex(typedData)
			if err != nil {
				return toolErrorResponse[previewSignatureOutput](err)
			}

			return nil, previewSignatureOutput{
				Meta:         newResponseMeta(authModeFromContext(tc)),
				Action:       string(requestAction),
				PrimaryType:  typedData.PrimaryType,
				Nonce:        nonce,
				ExpiresAfter: expiresAfter,
				SubAccountID: subAccountID,
				TypedData:    serialized,
				Digest:       digest,
				Notes: []string{
					"Sign typedData with any EIP-712-capable signer (viem.signTypedData, ethers Wallet._signTypedData, eth_signTypedData_v4, Web3.py sign_typed_data).",
					"Pass the same `nonce` and `expiresAfter` values (as numbers) to the corresponding write tool (place_order, modify_order, cancel_order, cancel_all_orders, or close_position).",
					"Split the produced 65-byte signature into {r, s, v} and pass it as the `signature` argument. v must be 27 or 28.",
					"digest is the precomputed keccak256 EIP-712 hash (useful for hardware/offline signers).",
					"For action=closePosition, the server reads the live position to derive the counter-side (BUY for short, SELL for long) and the default close quantity. If the position changes between this preview and the close_position call, re-preview to avoid INVALID_SIGNATURE.",
				},
			}, nil
		})
}

// Returns the auth-mode string for response metadata without forcing
// callers to reach into session internals.
func authModeFromContext(tc ToolContext) string {
	if tc.State != nil && tc.State.AuthMode != "" {
		return string(tc.State.AuthMode)
	}
	return "public"
}

// resolvePreviewDomainTriple normalises the auth-manager domain
// triple, falling back to the canonical Synthetix values when an
// override is empty. Mirrors the same defaulting CreateTradeTypedData
// applies internally.
func resolvePreviewDomainTriple(d tradePreviewDomainProvider) (string, string, int) {
	name := d.DomainName()
	if name == "" {
		name = sdkeip712.DefaultDomainName
	}
	version := d.DomainVersion()
	if version == "" {
		version = sdkeip712.DefaultDomainVersion
	}
	chainID := d.ChainID()
	if chainID == 0 {
		chainID = sdkeip712.DefaultChainID
	}
	return name, version, chainID
}

// resolvePreviewDomain returns the apitypes.TypedDataDomain the
// AuthMessage builder needs, sourced from the same auth-manager
// triple resolvePreviewDomainTriple uses.
func resolvePreviewDomain(d tradePreviewDomainProvider) apitypes.TypedDataDomain {
	name, version, chainID := resolvePreviewDomainTriple(d)
	return sdkeip712.Domain(name, version, chainID)
}
