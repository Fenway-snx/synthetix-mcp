// quick_* tools collapse the multi-call onboarding choreography
// (lookup_subaccount → preview_auth_message → wallet sign → authenticate
// → set_guardrails → preview_trade_signature → wallet sign → place_order)
// into a single MCP call by delegating EIP-712 signing to the in-process
// agent broker. They are only registered when the broker is enabled
// (see services/mcp/internal/config.AgentBrokerConfig); when the broker
// is off the regular signing tools remain the canonical surface.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	"github.com/Fenway-snx/synthetix-mcp/internal/agentbroker"
	internal_auth "github.com/Fenway-snx/synthetix-mcp/internal/auth"
	"github.com/synthetixio/synthetix-go/types"
	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

// QuickAuthenticator is the subset of the auth manager's authenticate API
// the quick tools rely on. Mirrors SessionAuthenticator but lives in
// this file so the broker auto-auth path is independent of any future
// signature changes to the public authenticate tool.
type QuickAuthenticator interface {
	Authenticate(ctx context.Context, sessionID string, message string, signatureHex string) (*internal_auth.AuthenticateResult, error)
}

type quickPlaceOrderInput struct {
	previewOrderInput
}

type quickClosePositionInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the broker's resolved subaccount."`
	Symbol       string    `json:"symbol" jsonschema:"Market symbol of the position to close, e.g. BTC-USDT."`
	Quantity     string    `json:"quantity,omitempty" jsonschema:"Quantity to close as a decimal string. Omit for full position close."`
	Method       string    `json:"method,omitempty" jsonschema:"Close method: 'market' (default) or 'limit'. Use limit with limitPrice for price-sensitive closes."`
	LimitPrice   string    `json:"limitPrice,omitempty" jsonschema:"Limit price for method=limit closes. Ignored for market closes."`
}

type quickCancelOrderInput struct {
	SubAccountID  FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the broker's resolved subaccount."`
	VenueOrderID  string    `json:"venueOrderId,omitempty" jsonschema:"Exchange-assigned order ID. Provide this OR clientOrderId, not both."`
	ClientOrderID string    `json:"clientOrderId,omitempty" jsonschema:"Client-supplied order ID. Provide this OR venueOrderId, not both."`
}

type quickCancelAllInput struct {
	SubAccountID FlexInt64 `json:"subAccountId,omitempty" jsonschema:"Optional subaccount ID. Omit to use the broker's resolved subaccount."`
	Symbol       string    `json:"symbol,omitempty" jsonschema:"Cancel only orders for this market symbol, e.g. BTC-USDT. Omit to cancel all orders across all markets."`
}

// Wires up the broker-backed quick_* tools when the agent broker is
// configured. The caller is expected to skip this registration when
// broker or tradeReads is nil. Each tool walks the same three-phase
// pipeline: ensureBrokerSession (auto-authenticate + apply default
// guardrails), build the validated action payload, and POST through
// the REST shim which handles nonce allocation, broker signing,
// local pre-flight validation, and the envelope dispatch.
//
// Errors at any phase route through toolErrorResponse so the agent
// receives the same classified {error, message, remediation} shape
// it gets from the legacy signed-action tools.
func RegisterQuickTools(
	server *mcp.Server,
	deps *ToolDeps,
	broker *agentbroker.Broker,
	authenticator QuickAuthenticator,
	tradeReads *TradeReadClient,
	snapshotManager *risksnapshot.Manager,
) {
	if broker == nil || tradeReads == nil {
		return
	}

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "quick_place_order",
		Description: "Place one order in a single call using the server-side agent broker (auto-authenticate + auto-sign). The MCP server holds a private key in process memory and signs the EIP-712 trade action on your behalf, removing the need to call preview_auth_message / preview_trade_signature / authenticate manually. Identical input shape to preview_order; identical output shape to place_order.",
	}, func(in quickPlaceOrderInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickPlaceOrderInput) (*mcp.CallToolResult, placeOrderOutput, error) {
			validated, normalized, err := buildValidatedPlaceOrder(input.previewOrderInput)
			if err != nil {
				return toolErrorResponse[placeOrderOutput](err)
			}
			if err := enforcePlaceOrderGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, normalized); err != nil {
				return toolErrorResponse[placeOrderOutput](err)
			}
			resp, err := tradeReads.PlaceOrders(ctx, tc, validated, validated.Payload)
			if err != nil {
				return toolErrorResponse[placeOrderOutput](fmt.Errorf("quick_place_order: %w", err))
			}
			if resp == nil || len(resp.Statuses) == 0 {
				return toolErrorResponse[placeOrderOutput](fmt.Errorf("quick_place_order: empty response"))
			}
			result := mapPlaceOrderResultREST(resp.Statuses[0], normalized.Symbol, normalized.Quantity)
			applyPlacedOrderSnapshot(tc.SessionID, snapshotManager, normalized, result)
			return nil, result, nil
		})

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "quick_close_position",
		Description: "Close a position in a single call using the server-side agent broker. Same semantics as close_position but with the EIP-712 signing handled internally.",
	}, func(in quickClosePositionInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickClosePositionInput) (*mcp.CallToolResult, closePositionOutput, error) {
			positionSide, currentQuantity, err := resolveClosablePosition(ctx, tradeReads, tc, input.Symbol)
			if err != nil {
				return toolErrorResponse[closePositionOutput](err)
			}
			closeQuantity := currentQuantity
			if input.Quantity != "" {
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
				orderType = "LIMIT"
				timeInForce = "GTC"
				price = input.LimitPrice
			} else if closeMethod != "market" {
				return toolErrorResponse[closePositionOutput](fmt.Errorf("method must be market or limit"))
			}
			side := "SELL"
			if strings.EqualFold(positionSide, "short") {
				side = "BUY"
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
			if err := enforcePlaceOrderGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, normalized); err != nil {
				return toolErrorResponse[closePositionOutput](err)
			}
			resp, err := tradeReads.PlaceOrders(ctx, tc, validated, validated.Payload)
			if err != nil {
				return toolErrorResponse[closePositionOutput](fmt.Errorf("quick_close_position: %w", err))
			}
			if resp == nil || len(resp.Statuses) == 0 {
				return toolErrorResponse[closePositionOutput](fmt.Errorf("quick_close_position: empty response"))
			}
			result := mapPlaceOrderResultREST(resp.Statuses[0], normalized.Symbol, normalized.Quantity)
			applyPlacedOrderSnapshot(tc.SessionID, snapshotManager, normalized, result)
			return nil, closePositionOutput{
				placeOrderOutput:          result,
				ClosedQuantity:            closeQuantity.String(),
				RemainingPositionQuantity: currentQuantity.Sub(closeQuantity).String(),
			}, nil
		})

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "quick_cancel_order",
		Description: "Cancel one open order using the server-side agent broker. Same semantics as cancel_order but with EIP-712 signing handled internally.",
	}, func(in quickCancelOrderInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickCancelOrderInput) (*mcp.CallToolResult, cancelOrderOutput, error) {
			legacy := cancelOrderInput{
				SubAccountID:  input.SubAccountID,
				VenueOrderID:  input.VenueOrderID,
				ClientOrderID: input.ClientOrderID,
			}
			validated, venueOrderIDs, clientOrderIDs, err := buildCancelPayload(legacy)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			var canonicalVenueOrderID string
			if len(venueOrderIDs) > 0 {
				canonicalVenueOrderID = fmtVenueID(venueOrderIDs[0])
			}
			var canonicalClientOrderID string
			if len(clientOrderIDs) > 0 {
				canonicalClientOrderID = clientOrderIDs[0]
			}
			orderContext, err := enforceCancelOrderGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, legacy, canonicalVenueOrderID, canonicalClientOrderID)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			envelopePayload, err := cancelOrdersEnvelopePayload(validated)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			resp, err := tradeReads.CancelOrders(ctx, tc, validated, envelopePayload)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](fmt.Errorf("quick_cancel_order: %w", err))
			}
			var statuses []types.OrderStatus
			if resp != nil {
				statuses = resp.Statuses
			}
			result := mapCancelOrderResultREST(statuses)
			applyCancelledOrdersSnapshot(tc.SessionID, snapshotManager, result.Orders, orderContext)
			return nil, result, nil
		})

	addAuthenticatedQuickTool(server, deps, broker, authenticator, &mcp.Tool{
		Name:        "quick_cancel_all_orders",
		Description: "Cancel all open orders for the broker subaccount in a single call. Same semantics as cancel_all_orders but with EIP-712 signing handled internally.",
	}, func(in quickCancelAllInput) *int64 { return int64Optional(in.SubAccountID.Int64()) },
		func(ctx context.Context, tc ToolContext, input quickCancelAllInput) (*mcp.CallToolResult, cancelOrderOutput, error) {
			legacy := cancelAllOrdersInput{
				SubAccountID: input.SubAccountID,
				Symbol:       input.Symbol,
			}
			validated, symbols, err := buildCancelAllPayload(legacy)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			if err := enforceCancelAllGuardrails(ctx, tc.SessionID, tc.State, snapshotManager, legacy); err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			envelopePayload, err := cancelAllEnvelopePayload(validated)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](err)
			}
			resp, err := tradeReads.CancelAllOrders(ctx, tc, validated, envelopePayload)
			if err != nil {
				return toolErrorResponse[cancelOrderOutput](fmt.Errorf("quick_cancel_all_orders: %w", err))
			}
			var items types.CancelAllOrdersResponse
			if resp != nil {
				items = *resp
			}
			result := mapCancelAllOrdersResultREST(items)
			applyCancelAllSnapshot(tc.SessionID, snapshotManager, symbols, result.Orders)
			return nil, result, nil
		})
}

// Extracts the on-wire params bytes (the ActionPayload pointer) from
// a validated cancel-orders action. Handles both venue-id and CLOID
// flavours; everything else is a bug at the caller since
// buildCancelPayload only returns those two.
func cancelOrdersEnvelopePayload(validated any) (any, error) {
	switch v := validated.(type) {
	case *validation.ValidatedCancelOrdersAction:
		return v.Payload, nil
	case *validation.ValidatedCancelOrdersByCloidAction:
		return v.Payload, nil
	default:
		return nil, fmt.Errorf("unsupported validated cancel action: %T", validated)
	}
}

func cancelAllEnvelopePayload(validated any) (any, error) {
	v, ok := validated.(*validation.ValidatedCancelAllOrdersAction)
	if !ok {
		return nil, fmt.Errorf("unsupported validated cancelAll action: %T", validated)
	}
	return v.Payload, nil
}

// Extracts the on-wire params bytes (the ActionPayload pointer)
// from a validated modify-order action. buildModifyPayload emits
// either the venue-id or CLOID flavour.
func modifyOrderEnvelopePayload(validated any) (any, error) {
	switch v := validated.(type) {
	case *validation.ValidatedModifyOrderAction:
		return v.Payload, nil
	case *validation.ValidatedModifyOrderByCloidAction:
		return v.Payload, nil
	default:
		return nil, fmt.Errorf("unsupported validated modify action: %T", validated)
	}
}

// addAuthenticatedQuickTool wraps addPublicTool with the broker auto-auth
// shim. The middleware sequence is:
//
//	load session (ignore not-found) → broker auto-auth (if not yet
//	authenticated) → broker default guardrails (if not yet set) → handler
//
// We use addPublicTool rather than addAuthenticatedTool because the
// quick_* tools are designed to work even on the very first call, before
// the agent has touched authenticate themselves. The auto-auth shim
// upgrades the session to authenticated in-flight before delegating to
// the inner handler with a guaranteed-valid ToolContext.
func addAuthenticatedQuickTool[In, Out any](
	server *mcp.Server,
	deps *ToolDeps,
	broker *agentbroker.Broker,
	authenticator QuickAuthenticator,
	tool *mcp.Tool,
	subAccountID func(In) *int64,
	handler func(ctx context.Context, tc ToolContext, input In) (*mcp.CallToolResult, Out, error),
) {
	addPublicTool(server, deps, tool, func(ctx context.Context, tc ToolContext, input In) (*mcp.CallToolResult, Out, error) {
		state, err := ensureBrokerSession(ctx, deps, broker, authenticator, tc.SessionID, tc.State)
		if err != nil {
			return toolErrorResponse[Out](err)
		}
		if requested := subAccountID(input); requested != nil && *requested != 0 && *requested != state.SubAccountID {
			return toolErrorResponse[Out](fmt.Errorf("requested subaccount does not match broker subaccount"))
		}
		return handler(ctx, ToolContext{SessionID: tc.SessionID, State: state}, input)
	})
}

// ensureBrokerSession returns a ready-to-trade session.State for the
// current MCP session, performing one or more of the following on
// demand:
//
//   - If the session is unauthenticated, run broker.SignAuthMessage and
//     authenticator.Authenticate to bind the session to the broker
//     wallet's subaccount.
//   - If the session is missing guardrails (i.e. would fall back to
//     read_only), seed it with the broker's GuardrailDefaults so the
//     subsequent trading call passes the guardrail check.
//
// On the happy path (session already authenticated AND guardrails
// already configured) this is a single in-memory session read, no
// signing.
func ensureBrokerSession(
	ctx context.Context,
	deps *ToolDeps,
	broker *agentbroker.Broker,
	authenticator QuickAuthenticator,
	sessionID string,
	state *session.State,
) (*session.State, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("mcp session ID is required")
	}

	if state == nil || state.AuthMode != session.AuthModeAuthenticated || state.SubAccountID <= 0 {
		subAccountID, err := broker.EnsureSubAccount(ctx)
		if err != nil {
			return nil, err
		}
		message, signatureHex, err := broker.SignAuthMessage(subAccountID)
		if err != nil {
			return nil, err
		}
		if _, err := authenticator.Authenticate(ctx, sessionID, message, signatureHex); err != nil {
			return nil, fmt.Errorf("agent broker auto-authenticate: %w", err)
		}
		refreshed, err := loadSessionState(ctx, deps.Store, sessionID)
		if err != nil {
			return nil, err
		}
		if refreshed == nil {
			return nil, fmt.Errorf("agent broker auto-authenticate succeeded but session state was not persisted")
		}
		state = refreshed
	}

	// Apply broker guardrail defaults if the agent has never called
	// set_guardrails. Existing guardrails (read_only or otherwise) are
	// always preserved — operators that explicitly downgraded the
	// session to read_only do not want a trading call to silently
	// upgrade them back to standard.
	if state.AgentGuardrails == nil {
		defaults := broker.GuardrailDefaults()
		state.AgentGuardrails = &guardrails.Config{
			AllowedOrderTypes:   append([]string{}, defaults.AllowedOrderTypes...),
			AllowedSymbols:      append([]string{}, defaults.AllowedSymbols...),
			MaxOrderQuantity:    defaults.MaxOrderQuantity,
			MaxPositionQuantity: defaults.MaxPositionQuantity,
			Preset:              defaults.Preset,
		}
		if err := deps.Store.Save(ctx, sessionID, state, deps.Cfg.SessionTTL); err != nil {
			return nil, fmt.Errorf("agent broker apply guardrail defaults: %w", err)
		}
	}
	return state, nil
}

func fmtVenueID(id uint64) string {
	return fmt.Sprintf("%d", id)
}

// (intentionally re-using existing helpers from trading_tools.go and
// trading_helpers — buildValidatedPlaceOrder, buildCancelPayload,
// buildCancelAllPayload, enforce*Guardrails, mapPlaceOrderResultREST,
// mapCancelOrderResultREST, etc.)
