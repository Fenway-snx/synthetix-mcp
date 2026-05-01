package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	"github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/synthetixio/synthetix-go/synthetix"
	backend_types "github.com/synthetixio/synthetix-go/types"
)

type tradeActionAuthenticator interface {
	ValidateTradeAction(
		sessionWalletAddress string,
		sessionSubAccountID int64,
		nonce int64,
		expiresAfter int64,
		action snx_lib_api_types.RequestAction,
		payload any,
		signature snx_lib_auth.TradeSignature,
	) error
}

type marketConfigReadClient interface {
	GetMarket(ctx context.Context, symbol string) (*backend_types.MarketResponse, error)
	GetMarketPrices(ctx context.Context) (map[string]backend_types.MarketPriceResponse, error)
}

type tradeSignatureInput struct {
	R string `json:"r" jsonschema:"Hex-encoded r component of the EIP-712 signature (0x-prefixed 32-byte value). Obtain the full signature by signing the typedData returned by preview_trade_signature, then split into {r, s, v}."`
	S string `json:"s" jsonschema:"Hex-encoded s component of the EIP-712 signature (0x-prefixed 32-byte value)."`
	V uint8  `json:"v" jsonschema:"Recovery byte of the EIP-712 signature (27 or 28). Wallets that return {0,1} should add 27 before sending."`
}

type previewOrderInput struct {
	SubAccountID  FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Symbol        string    `json:"symbol" jsonschema:"Market symbol, e.g. BTC-USDT. Use list_markets to discover available symbols."`
	Side          string    `json:"side" jsonschema:"Order side: BUY or SELL. Aliases long/short are accepted and normalized to BUY/SELL."`
	Type          string    `json:"type" jsonschema:"Order type: LIMIT, MARKET, STOP, STOP_MARKET, TAKE_PROFIT, or TAKE_PROFIT_MARKET."`
	Quantity      string    `json:"quantity" jsonschema:"Order quantity as a decimal string, e.g. '0.01'. Must meet market minimum trade amount."`
	Price         string    `json:"price,omitempty" jsonschema:"Limit price as a decimal string. Required for LIMIT, STOP, and TAKE_PROFIT orders. Must align with market tick size."`
	TriggerPrice  string    `json:"triggerPrice,omitempty" jsonschema:"Trigger price for conditional orders (STOP, STOP_MARKET, TAKE_PROFIT, TAKE_PROFIT_MARKET)."`
	TimeInForce   string    `json:"timeInForce,omitempty" jsonschema:"Time-in-force policy: GTC (default for limit), IOC, or FOK."`
	ReduceOnly    bool      `json:"reduceOnly,omitempty" jsonschema:"If true, the order can only reduce an existing position, never increase it."`
	PostOnly      bool      `json:"postOnly,omitempty" jsonschema:"If true, the limit order is rejected if it would immediately match (maker only)."`
	ClientOrderID string    `json:"clientOrderId,omitempty" jsonschema:"Optional client-supplied order ID for idempotency and client-side tracking."`
}

type signedPlaceOrderInput struct {
	previewOrderInput
	ExpiresAfter int64               `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Omit for no expiry."`
	Nonce        int64               `json:"nonce" jsonschema:"Unique nonce for this action. Must not be reused across calls."`
	Signature    tradeSignatureInput `json:"signature" jsonschema:"EIP-712 signature for this place-order action."`
}

type modifyOrderInput struct {
	SubAccountID  FlexInt64           `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	VenueOrderID  string              `json:"venueOrderId,omitempty" jsonschema:"Exchange-assigned order ID. Provide this OR clientOrderId, not both."`
	ClientOrderID string              `json:"clientOrderId,omitempty" jsonschema:"Client-supplied order ID. Provide this OR venueOrderId, not both."`
	Quantity      string              `json:"quantity,omitempty" jsonschema:"New order quantity as a decimal string. Omit to keep current quantity."`
	Price         string              `json:"price,omitempty" jsonschema:"New limit price as a decimal string. Omit to keep current price."`
	TriggerPrice  string              `json:"triggerPrice,omitempty" jsonschema:"New trigger price for conditional orders. Omit to keep current trigger."`
	ExpiresAfter  int64               `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Omit for no expiry."`
	Nonce         int64               `json:"nonce" jsonschema:"Unique nonce for this action. Must not be reused across calls."`
	Signature     tradeSignatureInput `json:"signature" jsonschema:"EIP-712 signature for this modify action."`
}

type cancelOrderInput struct {
	SubAccountID  FlexInt64           `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	VenueOrderID  string              `json:"venueOrderId,omitempty" jsonschema:"Exchange-assigned order ID. Provide this OR clientOrderId, not both."`
	ClientOrderID string              `json:"clientOrderId,omitempty" jsonschema:"Client-supplied order ID. Provide this OR venueOrderId, not both."`
	Nonce         int64               `json:"nonce" jsonschema:"Unique nonce for this action. Must not be reused across calls."`
	ExpiresAfter  int64               `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Omit for no expiry; when set, must be strictly greater than nonce and must match the value signed in the EIP-712 payload."`
	Signature     tradeSignatureInput `json:"signature" jsonschema:"EIP-712 signature for this cancel action."`
}

type cancelAllOrdersInput struct {
	SubAccountID FlexInt64           `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Symbol       string              `json:"symbol,omitempty" jsonschema:"Cancel only orders for this market symbol, e.g. BTC-USDT. Omit to cancel all orders across all markets."`
	Nonce        int64               `json:"nonce" jsonschema:"Unique nonce for this action. Must not be reused across calls."`
	ExpiresAfter int64               `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Omit for no expiry; when set, must be strictly greater than nonce and must match the value signed in the EIP-712 payload."`
	Signature    tradeSignatureInput `json:"signature" jsonschema:"EIP-712 signature for this cancel-all action."`
}

type closePositionInput struct {
	SubAccountID FlexInt64           `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the session's authenticated subaccount."`
	Symbol       string              `json:"symbol" jsonschema:"Market symbol of the position to close, e.g. BTC-USDT."`
	Side         string              `json:"side,omitempty" jsonschema:"Optional side of the existing position: long or short. BUY/SELL aliases are accepted. When provided, close_position skips the getPositions pre-flight and uses the caller-supplied side + quantity directly. Required when the agent broker is disabled. When omitted, close_position fetches positions via the broker-signed REST read and infers side."`
	Quantity     string              `json:"quantity,omitempty" jsonschema:"Quantity to close as a decimal string. Required when 'side' is provided (the server cannot infer full-position size without the getPositions pre-flight). When omitted and 'side' is also omitted, close_position closes the full inferred position."`
	Method       string              `json:"method,omitempty" jsonschema:"Close method: 'market' (default) or 'limit'. Use limit with limitPrice for price-sensitive closes."`
	LimitPrice   string              `json:"limitPrice,omitempty" jsonschema:"Limit price for method=limit closes. Ignored for market closes."`
	ExpiresAfter int64               `json:"expiresAfter,omitempty" jsonschema:"Signature expiry as UTC milliseconds since epoch. Omit for no expiry."`
	Nonce        int64               `json:"nonce" jsonschema:"Unique nonce for this action. Must not be reused across calls."`
	Signature    tradeSignatureInput `json:"signature" jsonschema:"EIP-712 signature for this close-position action."`
}

type previewOrderOutput struct {
	Meta             responseMeta          `json:"_meta"`
	CanSubmit        bool                  `json:"canSubmit"`
	NormalizedOrder  normalizedOrderOutput `json:"normalizedOrder"`
	Notes            []string              `json:"notes"`
	ValidationErrors []string              `json:"validationErrors"`
}

type normalizedOrderOutput struct {
	ClientOrderID string `json:"clientOrderId,omitempty"`
	PostOnly      bool   `json:"postOnly"`
	Price         string `json:"price,omitempty"`
	Quantity      string `json:"quantity"`
	ReduceOnly    bool   `json:"reduceOnly"`
	Side          string `json:"side"`
	Symbol        string `json:"symbol"`
	TimeInForce   string `json:"timeInForce,omitempty"`
	TriggerPrice  string `json:"triggerPrice,omitempty"`
	Type          string `json:"type"`
}

// Buckets the upstream OrderStatus enum so callers don't have to
// memorise it:
//   - ACCEPTED:             ACCEPTED, FILLED, PARTIALLY_FILLED, MODIFIED, PENDING
//   - REJECTED:             REJECTED, EXPIRED, CANCELLED(!isSuccess)
//   - PENDING_CONFIRMATION: UNKNOWN/empty (sync ack arrived before
//     match engine applied) — caller must poll,
//     NOT retry, to avoid double-fills.
const (
	OrderPhaseAccepted            = "ACCEPTED"
	OrderPhasePendingConfirmation = "PENDING_CONFIRMATION"
	OrderPhaseRejected            = "REJECTED"
)

type placeOrderOutput struct {
	Meta        responseMeta  `json:"_meta"`
	Accepted    bool          `json:"accepted"`
	AvgPrice    string        `json:"avgPrice"`
	CumQty      string        `json:"cumQty"`
	ErrorCode   string        `json:"errorCode,omitempty"`
	ErrorDetail *errorDetail  `json:"errorDetail,omitempty"`
	FollowUp    []string      `json:"followUp"`
	IsSuccess   bool          `json:"isSuccess"`
	Message     string        `json:"message"`
	OrderID     orderIDOutput `json:"orderId"`
	OrigQty     string        `json:"origQty"`
	Phase       string        `json:"phase"`
	Status      string        `json:"status"`
	Symbol      string        `json:"symbol"`
}

type modifyOrderOutput struct {
	Meta              responseMeta  `json:"_meta"`
	AveragePrice      string        `json:"averagePrice,omitempty"`
	CumulativeFillQty string        `json:"cumulativeFillQty,omitempty"`
	ErrorCode         string        `json:"errorCode,omitempty"`
	ErrorDetail       *errorDetail  `json:"errorDetail,omitempty"`
	ErrorMessage      string        `json:"errorMessage,omitempty"`
	OrderID           orderIDOutput `json:"orderId"`
	Price             string        `json:"price,omitempty"`
	Quantity          string        `json:"quantity,omitempty"`
	Status            string        `json:"status"`
	TriggerPrice      string        `json:"triggerPrice,omitempty"`
}

type cancelOrderItemOutput struct {
	ErrorCode    string        `json:"errorCode,omitempty"`
	ErrorDetail  *errorDetail  `json:"errorDetail,omitempty"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
	OrderID      orderIDOutput `json:"orderId"`
	Symbol       string        `json:"symbol"`
}

type cancelOrderOutput struct {
	Meta   responseMeta            `json:"_meta"`
	Orders []cancelOrderItemOutput `json:"orders"`
}

type closePositionOutput struct {
	placeOrderOutput
	ClosedQuantity            string `json:"closedQuantity"`
	RemainingPositionQuantity string `json:"remainingPositionQuantity"`
}

type closeAllPositionItemOutput struct {
	ClosedQuantity string           `json:"closedQuantity"`
	Order          placeOrderOutput `json:"order"`
	Side           string           `json:"side"`
	Symbol         string           `json:"symbol"`
}

type closeAllPositionsOutput struct {
	Meta      responseMeta                 `json:"_meta"`
	Positions []closeAllPositionItemOutput `json:"positions"`
}

type errorDetail struct {
	Remediation []string `json:"remediation"`
	Retryable   bool     `json:"retryable"`
}

func RegisterTradingTools(
	server *mcp.Server,
	deps *ToolDeps,
	marketConfigClient marketConfigReadClient,
	tradeReads *TradeReadClient,
	snapshotManager *risksnapshot.Manager,
	authenticator tradeActionAuthenticator,
	registerSignedTools bool,
) {
	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "preview_order",
		Description: "Dry-run an order to validate field normalization, market constraints (tick size, min trade amount, notional value), and order type compatibility without submitting. Returns canSubmit:true if the order shape passes all checks. Does not guarantee margin acceptance or matching.",
	}, func(in previewOrderInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input previewOrderInput) (*mcp.CallToolResult, previewOrderOutput, error) {
			validated, normalized, err := buildValidatedPlaceOrder(input)
			if err != nil {
				return nil, previewOrderOutput{
					Meta:            newResponseMeta(string(session.AuthModeAuthenticated)),
					CanSubmit:       false,
					NormalizedOrder: normalized,
					Notes: []string{
						"preview_order is non-authoritative and does not guarantee margin acceptance, rate-limit acceptance, or final backend acceptance.",
					},
					ValidationErrors: []string{err.Error()},
				}, nil
			}
			validationErrors := validatePreviewAgainstMarket(ctx, marketConfigClient, normalized)
			validationErrors = append(validationErrors, guardrailPreviewErrors(ctx, tc.SessionID, tc.State, snapshotManager, marketConfigClient, normalized)...)
			if len(validationErrors) > 0 {
				return nil, previewOrderOutput{
					Meta:            newResponseMeta(string(session.AuthModeAuthenticated)),
					CanSubmit:       false,
					NormalizedOrder: normalized,
					Notes: []string{
						"preview_order is non-authoritative and does not guarantee margin acceptance, rate-limit acceptance, or final backend acceptance.",
						"preview_order validates shared request shape plus market existence, open status, minimum trade size, tick-size alignment, and basic notional constraints.",
					},
					ValidationErrors: validationErrors,
				}, nil
			}

			return nil, previewOrderOutput{
				Meta:            newResponseMeta(string(session.AuthModeAuthenticated)),
				CanSubmit:       true,
				NormalizedOrder: normalized,
				Notes: []string{
					fmt.Sprintf("preview validated for authenticated subaccount %d", tc.State.SubAccountID),
					"preview_order is non-authoritative and does not guarantee margin acceptance, rate-limit acceptance, or final backend acceptance.",
					"preview_order validates shared request shape plus market existence, open status, minimum trade size, tick-size alignment, and basic notional constraints.",
					fmt.Sprintf("normalized client payload contains %d order(s)", len(validated.Payload.Orders)),
				},
				ValidationErrors: []string{},
			}, nil
		})

	if !registerSignedTools {
		return
	}

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_place_order",
		Description: "Advanced external-wallet path: submit one order with a caller-provided EIP-712 signature. Use only when the self-hosted broker is disabled and the agent holds a private key locally. Recommended flow: preview_order, preview_trade_signature action='placeOrders', sign typedData locally, then call signed_place_order with the echoed nonce/expiresAfter and split {r, s, v}.",
	}, func(in signedPlaceOrderInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input signedPlaceOrderInput) (*mcp.CallToolResult, placeOrderOutput, error) {
			validated, normalized, err := buildValidatedPlaceOrder(input.previewOrderInput)
			if err != nil {
				return toolErrorResponse[placeOrderOutput](err)
			}
			if err := enforcePlaceOrderGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, marketConfigClient, normalized); err != nil {
				return guardrailRejectionResponse[placeOrderOutput](err, normalized)
			}
			if err := authenticator.ValidateTradeAction(
				tc.State.WalletAddress,
				tc.State.SubAccountID,
				input.Nonce,
				input.ExpiresAfter,
				snx_lib_api_types.RequestAction("placeOrders"),
				validated,
				mapSignature(input.Signature),
			); err != nil {
				return toolErrorResponse[placeOrderOutput](err)
			}

			resp, err := tradeReads.PlaceOrdersWithSignature(ctx, tc, validated.Payload, SignedWrite{
				WalletAddress: tc.State.WalletAddress,
				Nonce:         input.Nonce,
				ExpiresAfter:  input.ExpiresAfter,
				Signature:     mapSignature(input.Signature),
			})
			if err != nil {
				return toolErrorResponse[placeOrderOutput](fmt.Errorf("place order: %w", err))
			}
			if resp == nil || len(resp.Statuses) == 0 {
				return toolErrorResponse[placeOrderOutput](fmt.Errorf("place order: empty response"))
			}

			result := mapPlaceOrderResultREST(resp.Statuses[0], normalized.Symbol, normalized.Quantity)
			applyPlacedOrderSnapshot(tc.SessionID, snapshotManager, normalized, result)
			return nil, result, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_modify_order",
		Description: "Advanced external-wallet path: modify one open order with a caller-provided EIP-712 signature. Use only when the self-hosted broker is disabled and the agent holds a private key locally. Identify the order by venueOrderId or clientOrderId, then use preview_trade_signature action='modifyOrder' before calling signed_modify_order.",
	}, func(in modifyOrderInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input modifyOrderInput) (*mcp.CallToolResult, modifyOrderOutput, error) {
			actionPayload, venueOrderID, clientOrderID, err := buildModifyPayload(input)
			if err != nil {
				return toolErrorResponse[modifyOrderOutput](err)
			}
			existingOrder, err := enforceModifyOrderGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, marketConfigClient, input, venueOrderID, clientOrderID)
			if err != nil {
				return toolErrorResponse[modifyOrderOutput](err)
			}
			if err := authenticator.ValidateTradeAction(
				tc.State.WalletAddress,
				tc.State.SubAccountID,
				input.Nonce,
				input.ExpiresAfter,
				snx_lib_api_types.RequestAction("modifyOrder"),
				actionPayload,
				mapSignature(input.Signature),
			); err != nil {
				return toolErrorResponse[modifyOrderOutput](err)
			}

			envelopePayload, err := modifyOrderEnvelopePayload(actionPayload)
			if err != nil {
				return toolErrorResponse[modifyOrderOutput](err)
			}
			_ = venueOrderID
			_ = clientOrderID

			resp, err := tradeReads.ModifyOrderWithSignature(ctx, tc, envelopePayload, SignedWrite{
				WalletAddress: tc.State.WalletAddress,
				Nonce:         input.Nonce,
				ExpiresAfter:  input.ExpiresAfter,
				Signature:     mapSignature(input.Signature),
			})
			if err != nil {
				return toolErrorResponse[modifyOrderOutput](fmt.Errorf("modify order: %w", err))
			}
			if resp == nil {
				return toolErrorResponse[modifyOrderOutput](fmt.Errorf("modify order: empty response"))
			}
			result := mapModifyOrderResultREST(*resp)
			applyModifiedOrderSnapshot(tc.SessionID, snapshotManager, existingOrder, input, result)
			return nil, result, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_cancel_order",
		Description: "Advanced external-wallet path: cancel one open order with a caller-provided EIP-712 signature. Use only when the self-hosted broker is disabled and the agent holds a private key locally. Use preview_trade_signature action='cancelOrders' before calling signed_cancel_order.",
	}, func(in cancelOrderInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input cancelOrderInput) (*mcp.CallToolResult, cancelOrderOutput, error) {
			actionPayload, venueOrderIDs, clientOrderIDs, err := buildCancelPayload(input)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			var canonicalVenueOrderID string
			if len(venueOrderIDs) > 0 {
				canonicalVenueOrderID = strconv.FormatUint(venueOrderIDs[0], 10)
			}
			var canonicalClientOrderID string
			if len(clientOrderIDs) > 0 {
				canonicalClientOrderID = clientOrderIDs[0]
			}
			orderContext, err := enforceCancelOrderGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, input, canonicalVenueOrderID, canonicalClientOrderID)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			if err := authenticator.ValidateTradeAction(
				tc.State.WalletAddress,
				tc.State.SubAccountID,
				input.Nonce,
				input.ExpiresAfter,
				snx_lib_api_types.RequestAction("cancelOrders"),
				actionPayload,
				mapSignature(input.Signature),
			); err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}

			envelopePayload, err := cancelOrdersEnvelopePayload(actionPayload)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}

			resp, err := tradeReads.CancelOrdersWithSignature(ctx, tc, envelopePayload, SignedWrite{
				WalletAddress: tc.State.WalletAddress,
				Nonce:         input.Nonce,
				ExpiresAfter:  input.ExpiresAfter,
				Signature:     mapSignature(input.Signature),
			})
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](fmt.Errorf("cancel order: %w", err))
			}
			if resp == nil {
				return toolErrorResponse[cancelOrderOutput](fmt.Errorf("cancel order: empty response"))
			}
			result := mapCancelOrderResultREST(resp.Statuses)
			applyCancelledOrdersSnapshot(tc.SessionID, snapshotManager, result.Orders, orderContext)
			return nil, result, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_cancel_all_orders",
		Description: "Advanced external-wallet path: cancel all open orders with a caller-provided EIP-712 signature. Use only when the self-hosted broker is disabled and the agent holds a private key locally. Always call get_open_orders first, then preview_trade_signature action='cancelAllOrders' before calling signed_cancel_all_orders.",
	}, func(in cancelAllOrdersInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input cancelAllOrdersInput) (*mcp.CallToolResult, cancelOrderOutput, error) {
			actionPayload, symbols, err := buildCancelAllPayload(input)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			if err := enforceCancelAllGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, input); err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			if err := authenticator.ValidateTradeAction(
				tc.State.WalletAddress,
				tc.State.SubAccountID,
				input.Nonce,
				input.ExpiresAfter,
				snx_lib_api_types.RequestAction("cancelAllOrders"),
				actionPayload,
				mapSignature(input.Signature),
			); err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}

			envelopePayload, err := cancelAllEnvelopePayload(actionPayload)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			resp, err := tradeReads.CancelAllOrdersWithSignature(ctx, tc, envelopePayload, SignedWrite{
				WalletAddress: tc.State.WalletAddress,
				Nonce:         input.Nonce,
				ExpiresAfter:  input.ExpiresAfter,
				Signature:     mapSignature(input.Signature),
			})
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](fmt.Errorf("cancel all orders: %w", err))
			}
			var items backend_types.CancelAllOrdersResponse
			if resp != nil {
				items = *resp
			}
			result := mapCancelAllOrdersResultREST(items)
			applyCancelAllSnapshot(tc.SessionID, snapshotManager, symbols, result.Orders)
			return nil, result, nil
		})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "signed_close_position",
		Description: "Advanced external-wallet path: close one position with a caller-provided EIP-712 signature. Use only when the self-hosted broker is disabled and the agent holds a private key locally. When broker reads are unavailable, pass explicit side and quantity. Use preview_trade_signature action='closePosition' before calling signed_close_position.",
	}, func(in closePositionInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input closePositionInput) (*mcp.CallToolResult, closePositionOutput, error) {
			positionSide, currentQuantity, err := resolveClosablePositionOrExplicit(ctx, tradeReads, tc, input.Symbol, input.Side, input.Quantity)
			if err != nil {
				return toolErrorResponse[closePositionOutput](err)
			}

			closeQuantity := currentQuantity
			if input.Side == "" && input.Quantity != "" {
				closeQuantity, err = decimal.NewFromString(input.Quantity)
				if err != nil {
					return toolErrorResponse[closePositionOutput](fmt.Errorf("invalid close quantity: %w", err))
				}
				if closeQuantity.GreaterThan(currentQuantity) {
					return toolErrorResponse[closePositionOutput](fmt.Errorf("close quantity exceeds current position quantity"))
				}
			}

			closeMethod := strings.ToLower(strings.TrimSpace(input.Method))
			if closeMethod == "" {
				closeMethod = "market"
			}
			orderType := "MARKET"
			timeInForce := ""
			price := ""
			if closeMethod == "limit" {
				orderType = synthetix.OrderTypeLimit
				timeInForce = synthetix.TimeInForceGTC
				price = input.LimitPrice
			} else if closeMethod != "market" {
				return toolErrorResponse[closePositionOutput](fmt.Errorf("method must be market or limit"))
			}

			side := synthetix.SideSell
			if strings.EqualFold(positionSide, "short") {
				side = synthetix.SideBuy
			}

			validated, normalized, err := buildValidatedPlaceOrder(previewOrderInput{
				SubAccountID: FlexInt64(tc.State.SubAccountID),
				Symbol:       input.Symbol,
				Side:         side,
				Type:         orderType,
				Quantity:     closeQuantity.String(),
				Price:        price,
				TimeInForce:  timeInForce,
				ReduceOnly:   true,
			})
			if err != nil {
				return toolErrorResponse[closePositionOutput](err)
			}
			if err := enforcePlaceOrderGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, marketConfigClient, normalized); err != nil {
				return toolErrorResponse[closePositionOutput](err)
			}
			if err := authenticator.ValidateTradeAction(
				tc.State.WalletAddress,
				tc.State.SubAccountID,
				input.Nonce,
				input.ExpiresAfter,
				snx_lib_api_types.RequestAction("placeOrders"),
				validated,
				mapSignature(input.Signature),
			); err != nil {
				return toolErrorResponse[closePositionOutput](err)
			}

			resp, err := tradeReads.PlaceOrdersWithSignature(ctx, tc, validated.Payload, SignedWrite{
				WalletAddress: tc.State.WalletAddress,
				Nonce:         input.Nonce,
				ExpiresAfter:  input.ExpiresAfter,
				Signature:     mapSignature(input.Signature),
			})
			if err != nil {
				return toolErrorResponse[closePositionOutput](fmt.Errorf("close position: %w", err))
			}
			if resp == nil || len(resp.Statuses) == 0 {
				return toolErrorResponse[closePositionOutput](fmt.Errorf("close position: empty response"))
			}

			result := mapPlaceOrderResultREST(resp.Statuses[0], normalized.Symbol, normalized.Quantity)
			applyPlacedOrderSnapshot(tc.SessionID, snapshotManager, normalized, result)
			remaining := currentQuantity.Sub(closeQuantity)
			if remaining.IsNegative() {
				remaining = decimal.Zero
			}
			return nil, closePositionOutput{
				placeOrderOutput:          result,
				ClosedQuantity:            closeQuantity.String(),
				RemainingPositionQuantity: remaining.String(),
			}, nil
		})
}

func buildValidatedPlaceOrder(input previewOrderInput) (*validation.ValidatedPlaceOrdersAction, normalizedOrderOutput, error) {
	apiOrder, normalized, err := normalizePlaceOrder(input)
	if err != nil {
		return nil, normalized, err
	}

	payload := &validation.PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   []snx_lib_api_json.PlaceOrderRequest{apiOrder},
		Grouping: validation.GroupingValues_na,
		Source:   "mcp",
	}
	validated, err := validation.NewValidatedPlaceOrdersAction(payload)
	if err != nil {
		return nil, normalized, err
	}
	return validated, normalized, nil
}

func buildValidatedPlaceOrders(inputs []previewOrderInput) (*validation.ValidatedPlaceOrdersAction, []normalizedOrderOutput, error) {
	orders := make([]snx_lib_api_json.PlaceOrderRequest, 0, len(inputs))
	normalized := make([]normalizedOrderOutput, 0, len(inputs))
	for _, input := range inputs {
		apiOrder, out, err := normalizePlaceOrder(input)
		if err != nil {
			return nil, nil, err
		}
		orders = append(orders, apiOrder)
		normalized = append(normalized, out)
	}
	payload := &validation.PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Orders:   orders,
		Grouping: validation.GroupingValues_na,
		Source:   "mcp",
	}
	validated, err := validation.NewValidatedPlaceOrdersAction(payload)
	if err != nil {
		return nil, nil, err
	}
	return validated, normalized, nil
}

func validatePreviewAgainstMarket(
	ctx context.Context,
	marketConfigClient marketConfigReadClient,
	normalized normalizedOrderOutput,
) []string {
	if marketConfigClient == nil {
		return []string{"market configuration validation is unavailable"}
	}

	market, err := marketConfigClient.GetMarket(ctx, normalized.Symbol)
	if err != nil {
		return []string{"market configuration lookup failed for the requested symbol"}
	}
	if market == nil || market.Symbol == "" {
		return []string{"symbol is not available for trading"}
	}

	validationErrors := make([]string, 0)
	if !market.IsOpen {
		validationErrors = append(validationErrors, "symbol is not currently open for trading")
	}

	quantity, err := decimal.NewFromString(normalized.Quantity)
	if err != nil {
		return []string{"quantity is not a valid decimal value"}
	}
	if market.MinOrderSize != "" {
		minTrade, err := decimal.NewFromString(market.MinOrderSize)
		if err == nil && quantity.LessThan(minTrade) {
			validationErrors = append(validationErrors, fmt.Sprintf("quantity must be at least %s", minTrade.String()))
		}
	}

	if normalized.Price != "" {
		if err := validatePriceIncrement(normalized.Price, market.PriceIncrement); err != nil {
			validationErrors = append(validationErrors, err.Error())
		}

		if market.MinNotionalValue != "" {
			price, priceErr := decimal.NewFromString(normalized.Price)
			minNotional, notionalErr := decimal.NewFromString(market.MinNotionalValue)
			if priceErr == nil && notionalErr == nil && quantity.Mul(price).LessThan(minNotional) {
				validationErrors = append(validationErrors, fmt.Sprintf("notional value must be at least %s", minNotional.String()))
			}
		}
	}

	if normalized.TriggerPrice != "" {
		if err := validatePriceIncrement(normalized.TriggerPrice, market.PriceIncrement); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("trigger price %s", err.Error()))
		}
	}

	return validationErrors
}

func validatePriceIncrement(rawValue string, rawTickSize string) error {
	if rawValue == "" || rawTickSize == "" {
		return nil
	}

	value, err := decimal.NewFromString(rawValue)
	if err != nil {
		return fmt.Errorf("must be a valid decimal value")
	}
	tickSize, err := decimal.NewFromString(rawTickSize)
	if err != nil || !tickSize.GreaterThan(decimal.Zero) {
		return nil
	}

	if !value.Mod(tickSize).IsZero() {
		return fmt.Errorf("must align to tick size %s", tickSize.String())
	}
	return nil
}

func normalizePlaceOrder(input previewOrderInput) (snx_lib_api_json.PlaceOrderRequest, normalizedOrderOutput, error) {
	side, err := normalizeOrderSide(input.Side)
	if err != nil {
		return snx_lib_api_json.PlaceOrderRequest{}, normalizedOrderOutput{}, err
	}
	symbol := normalizeSymbol(input.Symbol)

	orderType, triggerMarket, normalizedType, defaultTIF, err := mapMCPOrderType(strings.ToUpper(strings.TrimSpace(input.Type)), strings.ToUpper(strings.TrimSpace(input.TimeInForce)))
	if err != nil {
		return snx_lib_api_json.PlaceOrderRequest{}, normalizedOrderOutput{}, err
	}

	normalized := normalizedOrderOutput{
		ClientOrderID: input.ClientOrderID,
		PostOnly:      input.PostOnly,
		Price:         input.Price,
		Quantity:      input.Quantity,
		ReduceOnly:    input.ReduceOnly,
		Side:          side,
		Symbol:        symbol,
		TimeInForce:   defaultTIF,
		TriggerPrice:  input.TriggerPrice,
		Type:          normalizedType,
	}

	return snx_lib_api_json.PlaceOrderRequest{
		Symbol:          snx_lib_api_json.Symbol(symbol),
		Side:            strings.ToLower(side),
		OrderType:       orderType,
		Price:           input.Price,
		TriggerPrice:    input.TriggerPrice,
		Quantity:        snx_lib_api_json.Quantity(input.Quantity),
		ClientOrderId:   snx_lib_api_json.ClientOrderId(input.ClientOrderID),
		ReduceOnly:      input.ReduceOnly,
		IsTriggerMarket: triggerMarket,
		PostOnly:        input.PostOnly,
		ClosePosition:   false,
	}, normalized, nil
}

func mapMCPOrderType(orderType string, tif string) (string, bool, string, string, error) {
	switch orderType {
	case synthetix.OrderTypeLimit:
		if tif == "" || tif == synthetix.TimeInForceGTC {
			return "limitGtc", false, synthetix.OrderTypeLimit, synthetix.TimeInForceGTC, nil
		}
		if tif == synthetix.TimeInForceIOC {
			return "limitIoc", false, synthetix.OrderTypeLimit, synthetix.TimeInForceIOC, nil
		}
		return "", false, "", "", fmt.Errorf("LIMIT orders currently support GTC or IOC in MCP")
	case "MARKET":
		if tif != "" {
			return "", false, "", "", fmt.Errorf("MARKET orders do not accept timeInForce in MCP")
		}
		return "market", false, "MARKET", "", nil
	case "STOP":
		if tif != "" {
			return "", false, "", "", fmt.Errorf("STOP orders do not accept timeInForce in MCP")
		}
		return "triggerSl", false, "STOP", "", nil
	case "STOP_MARKET":
		if tif != "" {
			return "", false, "", "", fmt.Errorf("STOP_MARKET orders do not accept timeInForce in MCP")
		}
		return "triggerSl", true, "STOP_MARKET", "", nil
	case "TAKE_PROFIT":
		if tif != "" {
			return "", false, "", "", fmt.Errorf("TAKE_PROFIT orders do not accept timeInForce in MCP")
		}
		return "triggerTp", false, "TAKE_PROFIT", "", nil
	case "TAKE_PROFIT_MARKET":
		if tif != "" {
			return "", false, "", "", fmt.Errorf("TAKE_PROFIT_MARKET orders do not accept timeInForce in MCP")
		}
		return "triggerTp", true, "TAKE_PROFIT_MARKET", "", nil
	default:
		return "", false, "", "", fmt.Errorf("unsupported order type %q", orderType)
	}
}

func buildModifyPayload(input modifyOrderInput) (any, uint64, string, error) {
	if input.VenueOrderID != "" && input.ClientOrderID != "" {
		return nil, 0, "", fmt.Errorf("modify_order accepts either venueOrderId or clientOrderId, not both")
	}
	if input.VenueOrderID == "" && input.ClientOrderID == "" {
		return nil, 0, "", fmt.Errorf("modify_order requires venueOrderId or clientOrderId")
	}

	if input.ClientOrderID != "" {
		payload := &validation.ModifyOrderByCloidActionPayload{
			Action:        "modifyOrder",
			ClientOrderId: snx_lib_api_types.ClientOrderId(input.ClientOrderID),
			Price:         stringPtrIfNonEmpty(input.Price),
			Quantity:      quantityPtrIfNonEmpty(input.Quantity),
			TriggerPrice:  stringPtrIfNonEmpty(input.TriggerPrice),
		}
		validated, err := validation.NewValidatedModifyOrderByCloidAction(payload)
		if err != nil {
			return nil, 0, "", err
		}
		return validated, 0, string(validated.ClientOrderId), nil
	}

	payload := &validation.ModifyOrderActionPayload{
		Action:       "modifyOrder",
		VenueOrderId: snx_lib_api_types.VenueOrderId(input.VenueOrderID),
		Price:        stringPtrIfNonEmpty(input.Price),
		Quantity:     quantityPtrIfNonEmpty(input.Quantity),
		TriggerPrice: stringPtrIfNonEmpty(input.TriggerPrice),
	}
	validated, err := validation.NewValidatedModifyOrderAction(payload)
	if err != nil {
		return nil, 0, "", err
	}
	return validated, snx_lib_api_types.VenueOrderIdToUintUnvalidated(validated.VenueOrderId), "", nil
}

func buildCancelPayload(input cancelOrderInput) (any, []uint64, []string, error) {
	if input.VenueOrderID != "" && input.ClientOrderID != "" {
		return nil, nil, nil, fmt.Errorf("cancel_order accepts either venueOrderId or clientOrderId, not both")
	}
	if input.VenueOrderID == "" && input.ClientOrderID == "" {
		return nil, nil, nil, fmt.Errorf("cancel_order requires venueOrderId or clientOrderId")
	}

	if input.ClientOrderID != "" {
		payload := &validation.CancelOrdersByCloidActionPayload{
			Action:         "cancelOrders",
			ClientOrderIds: []snx_lib_api_types.ClientOrderId{snx_lib_api_types.ClientOrderId(input.ClientOrderID)},
		}
		validated, err := validation.NewValidatedCancelOrdersByCloidAction(payload)
		if err != nil {
			return nil, nil, nil, err
		}
		return validated, nil, []string{snx_lib_api_types.ClientOrderIdToStringUnvalidated(validated.ClientOrderIds[0])}, nil
	}

	payload := &validation.CancelOrdersActionPayload{
		Action:        "cancelOrders",
		VenueOrderIds: []snx_lib_api_types.VenueOrderId{snx_lib_api_types.VenueOrderId(input.VenueOrderID)},
	}
	validated, err := validation.NewValidatedCancelOrdersAction(payload)
	if err != nil {
		return nil, nil, nil, err
	}
	return validated, []uint64{snx_lib_api_types.VenueOrderIdToUintUnvalidated(validated.VenueOrderIds[0])}, nil, nil
}

func buildCancelAllPayload(input cancelAllOrdersInput) (any, []string, error) {
	symbols := []validation.Symbol{"*"}
	if input.Symbol != "" {
		symbols = []validation.Symbol{validation.Symbol(input.Symbol)}
	}
	payload := &validation.CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: symbols,
	}
	validated, err := validation.NewValidatedCancelAllOrdersAction(payload)
	if err != nil {
		return nil, nil, err
	}
	outSymbols := make([]string, 0, len(validated.Symbols))
	for _, s := range validated.Symbols {
		outSymbols = append(outSymbols, string(s))
	}
	return validated, outSymbols, nil
}

// Prefers caller-supplied side + quantity when present (the ext-
// wallet escape hatch for deployments without a broker-signed
// positions read) and falls back to the shim read otherwise. The
// explicit branch validates side is long/short and that quantity
// is a positive decimal.
func resolveClosablePositionOrExplicit(
	ctx context.Context,
	reads *TradeReadClient,
	tc ToolContext,
	symbol string,
	explicitSide string,
	explicitQuantity string,
) (string, decimal.Decimal, error) {
	side := strings.ToLower(strings.TrimSpace(explicitSide))
	if side != "" {
		normalizedSide, err := normalizePositionSide(side)
		if err != nil {
			return "", decimal.Zero, err
		}
		if strings.TrimSpace(explicitQuantity) == "" {
			return "", decimal.Zero, fmt.Errorf("quantity is required when side is provided (no getPositions pre-flight runs)")
		}
		qty, err := decimal.NewFromString(strings.TrimSpace(explicitQuantity))
		if err != nil {
			return "", decimal.Zero, fmt.Errorf("invalid close quantity: %w", err)
		}
		if !qty.GreaterThan(decimal.Zero) {
			return "", decimal.Zero, fmt.Errorf("close quantity must be positive")
		}
		return normalizedSide, qty, nil
	}
	return resolveClosablePosition(ctx, reads, tc, symbol)
}

func normalizeOrderSide(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "buy", "long":
		return synthetix.SideBuy, nil
	case "sell", "short":
		return synthetix.SideSell, nil
	default:
		return "", fmt.Errorf("side must be BUY or SELL; aliases long and short are accepted")
	}
}

func normalizePositionSide(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "buy", "long":
		return "long", nil
	case "sell", "short":
		return "short", nil
	default:
		return "", fmt.Errorf("side must be long or short; BUY and SELL aliases are accepted")
	}
}

func normalizeSymbol(raw string) string {
	symbol := strings.ToUpper(strings.TrimSpace(raw))
	if symbol == "" || strings.Contains(symbol, "-") {
		return symbol
	}
	return symbol + "-USDT"
}

// Uses the signed-read shim to pull positions for the current
// subaccount and client-side filters by symbol, aggregating into
// the long vs short decision close_position needs. The REST
// getPositions surface is broker-bound (reads sign with the broker
// key), so ext-wallet sessions without a broker have no way to
// reach this path and must instead pass explicit side + quantity.
func resolveClosablePosition(
	ctx context.Context,
	reads *TradeReadClient,
	tc ToolContext,
	symbol string,
) (string, decimal.Decimal, error) {
	if reads == nil {
		return "", decimal.Zero, fmt.Errorf("%w: close_position pre-flight needs a broker-signed positions read", ErrReadUnavailable)
	}
	positions, err := reads.GetPositions(ctx, tc)
	if err != nil {
		return "", decimal.Zero, fmt.Errorf("get positions for close: %w", err)
	}

	var longQty, shortQty decimal.Decimal
	normSym := strings.ToUpper(strings.TrimSpace(symbol))
	for _, p := range positions {
		if !strings.EqualFold(strings.TrimSpace(p.Symbol), normSym) {
			continue
		}
		qty, qerr := decimal.NewFromString(strings.TrimSpace(p.Quantity))
		if qerr != nil {
			// A malformed row shouldn't mask a real exposure reading
			// elsewhere in the response; upstream has been observed
			// to emit zero-quantity rows during tier rebuilds.
			continue
		}
		if qty.IsZero() {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(p.Side)) {
		case "long":
			longQty = longQty.Add(qty.Abs())
		case "short":
			shortQty = shortQty.Add(qty.Abs())
		}
	}

	if longQty.GreaterThan(decimal.Zero) && shortQty.GreaterThan(decimal.Zero) {
		return "", decimal.Zero, fmt.Errorf("cannot close symbol with both long and short exposure")
	}
	if longQty.GreaterThan(decimal.Zero) {
		return "long", longQty, nil
	}
	if shortQty.GreaterThan(decimal.Zero) {
		return "short", shortQty, nil
	}
	return "", decimal.Zero, fmt.Errorf("no open position found for symbol %s", symbol)
}

func stringPtrIfNonEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func quantityPtrIfNonEmpty(value string) *validation.Quantity {
	if value == "" {
		return nil
	}
	q := validation.Quantity(value)
	return &q
}

func mapSignature(signature tradeSignatureInput) snx_lib_auth.TradeSignature {
	return snx_lib_auth.TradeSignature{
		R: signature.R,
		S: signature.S,
		V: signature.V,
	}
}

// Folds OrderStatus + isSuccess/errorCode into a phase + follow-up.
// isSuccess alone is insufficient: a sync ack racing the match-engine
// apply returns isSuccess=true with status=UNKNOWN, so UNKNOWN must
// land in PENDING_CONFIRMATION to block retries.
func classifyPlaceOrderPhase(status string, isSuccess bool, errorCode string) (string, []string) {
	normalised := strings.ToUpper(strings.TrimSpace(status))
	if errorCode != "" || (!isSuccess && normalised != "" && normalised != "UNKNOWN") {
		return OrderPhaseRejected, []string{
			"Inspect errorCode / errorDetail to determine why the matching engine rejected this order.",
			"Adjust the order parameters before retrying.",
		}
	}
	switch normalised {
	case "ACCEPTED", "FILLED", "PARTIALLY_FILLED", "MODIFIED", "PENDING":
		return OrderPhaseAccepted, []string{
			"Use get_open_orders or get_order_history to retrieve the final fill state.",
		}
	case "REJECTED", "EXPIRED":
		return OrderPhaseRejected, []string{
			"Inspect errorCode / errorDetail to determine why the matching engine rejected this order.",
		}
	case "CANCELLED":
		if isSuccess {
			return OrderPhaseAccepted, []string{"The order was cancelled before it could rest on the book."}
		}
		return OrderPhaseRejected, []string{
			"Inspect errorCode / errorDetail to determine why the order was cancelled.",
		}
	default:
		return OrderPhasePendingConfirmation, []string{
			"The order submission is still awaiting final confirmation.",
			"Poll get_open_orders or get_order_history with the returned clientOrderId to confirm the final state.",
			"Do NOT retry place_order with the same clientOrderId until you have confirmed it is not live.",
		}
	}
}

func errorDetailForCode(code string) *errorDetail {
	switch code {
	case "":
		return nil
	case "IDEMPOTENCY_CONFLICT":
		return &errorDetail{
			Remediation: []string{
				"Treat the existing clientOrderId as already used for this order-scoped write.",
				"Fetch open or historical orders for the subaccount before retrying with the same clientOrderId.",
				"Generate a new clientOrderId if you intend to submit a materially new order.",
			},
			Retryable: false,
		}
	case "INSUFFICIENT_MARGIN":
		return &errorDetail{
			Remediation: []string{
				"Reduce quantity or improve price to lower required margin.",
				"Close or reduce other positions before retrying.",
				"Recheck account summary and free collateral before resubmitting.",
			},
			Retryable: false,
		}
	case "MARKET_CLOSED", "WICK_INSURANCE_ACTIVE":
		return &errorDetail{
			Remediation: []string{
				"Wait for the market protection window to clear before retrying.",
				"Refresh market status and summary before sending another write.",
			},
			Retryable: true,
		}
	case "OPERATION_TIMEOUT":
		return &errorDetail{
			Remediation: []string{
				"Query open orders or order history before retrying so you do not duplicate an accepted write.",
				"If the prior attempt did not land, retry with the same clientOrderId only for idempotent CLOID-based flows.",
			},
			Retryable: true,
		}
	case "ORDER_NOT_FOUND", "POSITION_NOT_FOUND":
		return &errorDetail{
			Remediation: []string{
				"Refresh account state before retrying this action.",
				"Verify the referenced order or position still exists on the authenticated subaccount.",
			},
			Retryable: false,
		}
	case "POST_ONLY_WOULD_TRADE", "PRICE_OUT_OF_BOUNDS", "QUANTITY_TOO_SMALL", "QUANTITY_BELOW_FILLED", "REDUCE_ONLY_NO_POSITION", "REDUCE_ONLY_SAME_SIDE", "REDUCE_ONLY_WOULD_INCREASE":
		return &errorDetail{
			Remediation: []string{
				"Adjust the order parameters and resubmit a new request.",
				"Use preview_order or current account state to validate the revised shape first.",
			},
			Retryable: false,
		}
	case "NO_LIQUIDITY", "IOC_NOT_FILLED", "FOK_NOT_FILLED":
		return &errorDetail{
			Remediation: []string{
				"Inspect the orderbook and recent trades before retrying.",
				"Consider a smaller size, wider price, or a non-immediate execution style.",
			},
			Retryable: false,
		}
	default:
		return &errorDetail{
			Remediation: []string{
				"Use the backend errorCode as the primary automation signal.",
				"Refresh market and account state before retrying a changed request.",
			},
			Retryable: false,
		}
	}
}
