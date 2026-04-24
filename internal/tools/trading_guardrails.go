package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/shopspring/decimal"
)

func guardrailError(message string) error {
	return fmt.Errorf("guardrail violation: %s", message)
}

func guardrailsForState(state *session.State) (*guardrails.Resolved, error) {
	if state == nil || state.AgentGuardrails == nil {
		return guardrails.Resolve(nil)
	}
	return guardrails.Resolve(state.AgentGuardrails)
}

func guardrailPreviewErrors(
	ctx context.Context,
	sessionID string,
	state *session.State,
	snapshotManager *risksnapshot.Manager,
	normalized normalizedOrderOutput,
) []string {
	resolved, err := guardrailsForState(state)
	if err != nil {
		return []string{err.Error()}
	}
	snapshot, err := snapshotManager.EnsureHydrated(ctx, sessionID, state.SubAccountID)
	if err != nil {
		return []string{fmt.Sprintf("risk snapshot hydration failed: %s", err.Error())}
	}
	return guardrailOrderErrors(resolved, snapshot, normalized)
}

func enforcePlaceOrderGuardrails(
	ctx context.Context,
	sessionID string,
	state *session.State,
	snapshotManager *risksnapshot.Manager,
	normalized normalizedOrderOutput,
) error {
	resolved, err := guardrailsForState(state)
	if err != nil {
		return err
	}

	snapshot, err := snapshotManager.EnsureHydrated(ctx, sessionID, state.SubAccountID)
	if err != nil {
		return guardrailError(fmt.Sprintf("risk snapshot hydration failed: %s", err.Error()))
	}

	errors := guardrailOrderErrors(resolved, snapshot, normalized)
	if len(errors) == 0 {
		return nil
	}
	return guardrailError(strings.Join(errors, "; "))
}

func enforceModifyOrderGuardrails(
	ctx context.Context,
	sessionID string,
	state *session.State,
	snapshotManager *risksnapshot.Manager,
	input modifyOrderInput,
	canonicalVenueOrderID uint64,
	canonicalClientOrderID string,
) (*risksnapshot.Order, error) {
	resolved, err := guardrailsForState(state)
	if err != nil {
		return nil, err
	}
	if resolved.IsReadOnly() {
		return nil, guardrailError("session is read_only and does not permit modify_order")
	}

	snapshot, err := snapshotManager.EnsureHydrated(ctx, sessionID, state.SubAccountID)
	if err != nil {
		return nil, guardrailError(fmt.Sprintf("risk snapshot hydration failed: %s", err.Error()))
	}

	orderContext, ok := snapshot.LookupOrder(canonicalVenueOrderIDString(canonicalVenueOrderID), canonicalClientOrderID)
	if !ok {
		return nil, guardrailError("open order not found in current subaccount session")
	}

	if !resolved.IsSymbolAllowed(orderContext.Symbol) {
		return nil, guardrailError(fmt.Sprintf("symbol %s is not allowed for this session", orderContext.Symbol))
	}
	if !resolved.IsOrderTypeAllowed(orderContext.OrderType) {
		return nil, guardrailError(fmt.Sprintf("order type %s is not allowed for this session", orderContext.OrderType))
	}
	if strings.TrimSpace(input.Quantity) == "" {
		return orderContext, nil
	}

	quantity, err := decimal.NewFromString(input.Quantity)
	if err != nil {
		return nil, guardrailError("quantity must be a valid decimal")
	}
	if quantity.GreaterThan(resolved.MaxOrderQuantity) {
		return nil, guardrailError(fmt.Sprintf("quantity %s exceeds maxOrderQuantity %s", quantity.String(), resolved.MaxOrderQuantity.String()))
	}
	if quantity.Equal(orderContext.Quantity) || !resolved.HasMaxPositionQuantity() || orderContext.ReduceOnly {
		return orderContext, nil
	}
	filledBeforeModify := orderContext.Quantity.Sub(orderContext.RemainingQuantity)
	modifiedRemainingQuantity := decimal.Max(quantity.Sub(filledBeforeModify), decimal.Zero)

	if err := enforcePositionCap(snapshot, orderContext.Symbol, orderContext.Side, modifiedRemainingQuantity, resolved.MaxPositionQuantity, orderContext); err != nil {
		return nil, guardrailError(err.Error())
	}
	return orderContext, nil
}

func enforceCancelOrderGuardrails(
	ctx context.Context,
	sessionID string,
	state *session.State,
	snapshotManager *risksnapshot.Manager,
	input cancelOrderInput,
	canonicalVenueOrderID string,
	canonicalClientOrderID string,
) (*risksnapshot.Order, error) {
	resolved, err := guardrailsForState(state)
	if err != nil {
		return nil, err
	}
	if resolved.IsReadOnly() {
		return nil, guardrailError("session is read_only and does not permit cancel_order")
	}

	snapshot, err := snapshotManager.EnsureHydrated(ctx, sessionID, state.SubAccountID)
	if err != nil {
		return nil, guardrailError(fmt.Sprintf("risk snapshot hydration failed: %s", err.Error()))
	}

	orderContext, ok := snapshot.LookupOrder(canonicalVenueOrderID, canonicalClientOrderID)
	if !ok {
		return nil, guardrailError("open order not found in current subaccount session")
	}
	if !resolved.IsSymbolAllowed(orderContext.Symbol) {
		return nil, guardrailError(fmt.Sprintf("symbol %s is not allowed for this session", orderContext.Symbol))
	}
	return orderContext, nil
}

func canonicalVenueOrderIDString(venueOrderID uint64) string {
	if venueOrderID == 0 {
		return ""
	}
	return strconv.FormatUint(venueOrderID, 10)
}

func enforceCancelAllGuardrails(
	ctx context.Context,
	sessionID string,
	state *session.State,
	snapshotManager *risksnapshot.Manager,
	input cancelAllOrdersInput,
) error {
	resolved, err := guardrailsForState(state)
	if err != nil {
		return err
	}
	if resolved.IsReadOnly() {
		return guardrailError("session is read_only and does not permit cancel_all_orders")
	}

	if strings.TrimSpace(input.Symbol) != "" {
		if !resolved.IsSymbolAllowed(input.Symbol) {
			return guardrailError(fmt.Sprintf("symbol %s is not allowed for this session", input.Symbol))
		}
		return nil
	}

	snapshot, err := snapshotManager.EnsureHydrated(ctx, sessionID, state.SubAccountID)
	if err != nil {
		return guardrailError(fmt.Sprintf("risk snapshot hydration failed: %s", err.Error()))
	}
	for _, order := range snapshot.AllOpenOrders() {
		if !resolved.IsSymbolAllowed(order.Symbol) {
			return guardrailError(fmt.Sprintf("symbol %s is not allowed for this session", order.Symbol))
		}
	}
	return nil
}

func guardrailOrderErrors(
	resolved *guardrails.Resolved,
	snapshot *risksnapshot.Snapshot,
	normalized normalizedOrderOutput,
) []string {
	if resolved.IsReadOnly() {
		return []string{"session is read_only and does not permit order placement"}
	}

	errors := make([]string, 0, 4)
	if !resolved.IsSymbolAllowed(normalized.Symbol) {
		errors = append(errors, fmt.Sprintf("symbol %s is not allowed for this session", normalized.Symbol))
	}
	if !resolved.IsOrderTypeAllowed(normalized.Type) {
		errors = append(errors, fmt.Sprintf("order type %s is not allowed for this session", normalized.Type))
	}

	quantity, err := decimal.NewFromString(normalized.Quantity)
	if err != nil {
		errors = append(errors, "quantity must be a valid decimal")
		return errors
	}
	if resolved.HasMaxOrderQuantity() && quantity.GreaterThan(resolved.MaxOrderQuantity) {
		errors = append(errors, fmt.Sprintf("quantity %s exceeds maxOrderQuantity %s", quantity.String(), resolved.MaxOrderQuantity.String()))
	}
	if normalized.ReduceOnly || !resolved.HasMaxPositionQuantity() {
		return errors
	}

	if err := enforcePositionCap(snapshot, normalized.Symbol, normalized.Side, quantity, resolved.MaxPositionQuantity, nil); err != nil {
		errors = append(errors, err.Error())
	}
	return errors
}

func enforcePositionCap(
	snapshot *risksnapshot.Snapshot,
	symbol string,
	side string,
	quantity decimal.Decimal,
	maxPositionQuantity decimal.Decimal,
	excludedOrder *risksnapshot.Order,
) error {
	if snapshot == nil {
		return fmt.Errorf("risk snapshot is unavailable for guardrail enforcement")
	}

	currentSignedQuantity := snapshot.SignedPosition(symbol)
	pendingBuys, pendingSells := snapshot.PendingExposure(symbol)
	if excludedOrder != nil && strings.EqualFold(excludedOrder.Symbol, symbol) && !excludedOrder.ReduceOnly {
		if strings.EqualFold(excludedOrder.Side, "SELL") {
			pendingSells = pendingSells.Sub(excludedOrder.RemainingQuantity)
		} else {
			pendingBuys = pendingBuys.Sub(excludedOrder.RemainingQuantity)
		}
	}
	if strings.EqualFold(side, "SELL") {
		pendingSells = pendingSells.Add(quantity)
	} else {
		pendingBuys = pendingBuys.Add(quantity)
	}

	longScenario := currentSignedQuantity.Add(pendingBuys).Abs()
	shortScenario := currentSignedQuantity.Sub(pendingSells).Abs()
	worstCase := decimal.Max(longScenario, shortScenario)
	if worstCase.GreaterThan(maxPositionQuantity) {
		return fmt.Errorf("worst-case position quantity %s exceeds maxPositionQuantity %s", worstCase.String(), maxPositionQuantity.String())
	}
	return nil
}
