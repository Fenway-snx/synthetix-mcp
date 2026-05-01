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
	Action                    string                                 `json:"action" jsonschema:"One of: placeOrders, modifyOrder, cancelOrders, cancelAllOrders, closePosition."`
	SubAccountID              FlexInt64                              `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Nonce                     int64                                  `json:"nonce,omitempty" jsonschema:"Unique nonce for this action. Omit to have the server generate a fresh monotonic nonce (current UTC ms)."`
	ExpiresAfter              int64                                  `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Must be strictly greater than nonce. Omit to default to nonce + 24h."`
	AddDelegatedSigner        *previewAddDelegatedSignerInput        `json:"addDelegatedSigner,omitempty" jsonschema:"Populate for action=addDelegatedSigner."`
	PlaceOrder                *previewOrderInput                     `json:"placeOrder,omitempty" jsonschema:"Populate for action=placeOrders. Same shape as the signed_place_order tool's order fields (symbol, side, type, quantity, price, timeInForce, etc.)."`
	ModifyOrder               *previewTradeModifyInput               `json:"modifyOrder,omitempty" jsonschema:"Populate for action=modifyOrder."`
	CancelOrder               *previewTradeCancelInput               `json:"cancelOrder,omitempty" jsonschema:"Populate for action=cancelOrders (single order)."`
	CancelAll                 *previewTradeCancelAllInput            `json:"cancelAllOrders,omitempty" jsonschema:"Populate for action=cancelAllOrders."`
	ClosePosition             *previewTradeClosePosInput             `json:"closePosition,omitempty" jsonschema:"Populate for action=closePosition. The server normalizes it into a placeOrders action under the hood."`
	RemoveAllDelegatedSigners *previewRemoveAllDelegatedSignersInput `json:"removeAllDelegatedSigners,omitempty" jsonschema:"Populate for action=removeAllDelegatedSigners."`
	RemoveDelegatedSigner     *previewRemoveDelegatedSignerInput     `json:"removeDelegatedSigner,omitempty" jsonschema:"Populate for action=removeDelegatedSigner."`
	ScheduleCancel            *previewScheduleCancelInput            `json:"scheduleCancel,omitempty" jsonschema:"Populate for action=scheduleCancel."`
	TransferCollateral        *previewTransferCollateralInput        `json:"transferCollateral,omitempty" jsonschema:"Populate for action=transferCollateral."`
	UpdateLeverage            *previewUpdateLeverageInput            `json:"updateLeverage,omitempty" jsonschema:"Populate for action=updateLeverage."`
	WithdrawCollateral        *previewWithdrawCollateralInput        `json:"withdrawCollateral,omitempty" jsonschema:"Populate for action=withdrawCollateral."`
}

type previewAddDelegatedSignerInput struct {
	DelegateAddress string   `json:"delegateAddress" jsonschema:"Delegate EVM wallet address."`
	Permissions     []string `json:"permissions" jsonschema:"Delegated permissions, e.g. trading."`
	ExpiresAt       int64    `json:"expiresAt,omitempty" jsonschema:"Optional UNIX timestamp in seconds. Omit or set 0 for no expiry."`
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
	Side       string `json:"side,omitempty" jsonschema:"Optional side of the existing position: 'long' or 'short'. When provided, the preview skips the live getPositions pre-flight and uses the caller-supplied side + quantity to build the typed data. Required in deployments without a broker-signed positions read. Must match the value signed_close_position will be called with."`
	Quantity   string `json:"quantity,omitempty" jsonschema:"Quantity to close. Required when 'side' is provided. When omitted and 'side' is also omitted, the server reads the live position and defaults to the full open quantity, mirroring signed_close_position."`
	Method     string `json:"method,omitempty" jsonschema:"Close method: 'market' (default) or 'limit'."`
	LimitPrice string `json:"limitPrice,omitempty" jsonschema:"Limit price when method=limit."`
}

type previewRemoveAllDelegatedSignersInput struct{}

type previewRemoveDelegatedSignerInput struct {
	DelegateAddress string `json:"delegateAddress" jsonschema:"Delegate EVM wallet address to remove."`
}

type previewScheduleCancelInput struct {
	TimeoutSeconds int64 `json:"timeoutSeconds" jsonschema:"Seconds before open orders are cancelled if not refreshed. Pass 0 to clear."`
}

type previewTransferCollateralInput struct {
	ToSubAccountID FlexInt64 `json:"toSubAccountId" jsonschema:"Destination subaccount ID."`
	Symbol         string    `json:"symbol" jsonschema:"Collateral symbol, e.g. USDC."`
	Amount         string    `json:"amount" jsonschema:"Amount to transfer as a decimal string."`
}

type previewUpdateLeverageInput struct {
	Symbol   string `json:"symbol" jsonschema:"Market symbol, e.g. BTC-USDT."`
	Leverage string `json:"leverage" jsonschema:"Target leverage as a positive decimal string."`
}

type previewWithdrawCollateralInput struct {
	Symbol      string `json:"symbol" jsonschema:"Collateral symbol, e.g. USDC."`
	Amount      string `json:"amount" jsonschema:"Amount to withdraw as a decimal string."`
	Destination string `json:"destination" jsonschema:"Destination EVM address."`
}

// Signature payload material for the matching write tool.
// Nonce and expiry echo back for verbatim reuse.
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

// Registers preview helpers for auth bootstrap and trade signing.
func RegisterSignaturePreviewTools(
	server *mcp.Server,
	deps *ToolDeps,
	domain tradePreviewDomainProvider,
	tradeReads *TradeReadClient,
	registerTradePreview bool,
) {
	addPublicTool(server, deps, &mcp.Tool{
		Name:        "preview_auth_message",
		Description: "Return the exact EIP-712 typed-data object a local sidecar signer should sign to produce the `message` argument of the `authenticate` tool. Agents should use this instead of hand-crafting EIP-712 payloads, but Claude cannot sign the payload by itself. In external-wallet mode, ask the operator to run sample/node-scripts/authenticate-external-wallet.mjs against this MCP session ID. If get_server_info.agentBroker.enabled=true, use canonical broker tools instead. Never dump returned typedData into chat and ask a human to sign it.",
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

	if !registerTradePreview {
		return
	}

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "preview_trade_signature",
		Description: "Return the canonical EIP-712 typed-data object a client should sign to authorize one signed_* trade action. The returned nonce and expiresAfter MUST be passed back verbatim to the corresponding signed_* write tool along with the produced signature. This is the recommended way to obtain external-wallet trade signatures because field order, type coercions, and normalization must match the server byte-for-byte. AGENT POLICY: only call this when you (the agent) hold a private key locally. If get_server_info.agentBroker.enabled=true, call the canonical broker tools instead; the self-hosted broker generates and signs the payload internally.",
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

			// Trade typed-data construction stays here because it depends on
			// rich validation payloads that the SDK must not import.
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
					"Pass the same `nonce` and `expiresAfter` values (as numbers) to the corresponding signed_* write tool (signed_place_order, signed_modify_order, signed_cancel_order, signed_cancel_all_orders, signed_close_position, or the matching signed lifecycle tool).",
					"Split the produced 65-byte signature into {r, s, v} and pass it as the `signature` argument. v must be 27 or 28.",
					"digest is the precomputed keccak256 EIP-712 hash (useful for hardware/offline signers).",
					"For action=closePosition, the server reads the live position to derive the counter-side (BUY for short, SELL for long) and the default close quantity. If the position changes between this preview and the signed_close_position call, re-preview to avoid INVALID_SIGNATURE.",
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

// Normalises the signing domain, falling back to canonical values.
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

// Returns the signing domain needed for authentication previews.
func resolvePreviewDomain(d tradePreviewDomainProvider) apitypes.TypedDataDomain {
	name, version, chainID := resolvePreviewDomainTriple(d)
	return sdkeip712.Domain(name, version, chainID)
}
