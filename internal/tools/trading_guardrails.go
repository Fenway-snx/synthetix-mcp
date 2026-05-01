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
	"github.com/synthetixio/synthetix-go/types"
)

func guardrailError(message string) error {
	return fmt.Errorf("guardrail violation: %s", message)
}

type guardrailPriceReader interface {
	GetMarketPrices(ctx context.Context) (map[string]types.MarketPriceResponse, error)
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
	priceReader guardrailPriceReader,
	normalized normalizedOrderOutput,
) []string {
	resolved, err := guardrailsForState(state)
	if err != nil {
		return []string{err.Error()}
	}
	var snapshot *risksnapshot.Snapshot
	if resolved.HasMaxPositionQuantity() || resolved.HasMaxPositionNotional() {
		var err error
		snapshot, err = snapshotManager.EnsureHydrated(ctx, sessionID, state.SubAccountID)
		if err != nil {
			return []string{fmt.Sprintf("risk snapshot hydration failed: %s", err.Error())}
		}
	}
	return guardrailOrderErrors(ctx, priceReader, resolved, snapshot, normalized)
}

func enforcePlaceOrderGuardrails(
	ctx context.Context,
	sessionID string,
	state *session.State,
	snapshotManager *risksnapshot.Manager,
	priceReader guardrailPriceReader,
	normalized normalizedOrderOutput,
) error {
	resolved, err := guardrailsForState(state)
	if err != nil {
		return err
	}

	var snapshot *risksnapshot.Snapshot
	if resolved.HasMaxPositionQuantity() || resolved.HasMaxPositionNotional() {
		var err error
		snapshot, err = snapshotManager.EnsureHydrated(ctx, sessionID, state.SubAccountID)
		if err != nil {
			return guardrailError(fmt.Sprintf("risk snapshot hydration failed: %s", err.Error()))
		}
	}

	violations := guardrailOrderViolations(ctx, priceReader, resolved, snapshot, normalized)
	if len(violations) == 0 {
		return nil
	}
	return firstViolationOrJoined(violations)
}

func enforceModifyOrderGuardrails(
	ctx context.Context,
	sessionID string,
	state *session.State,
	snapshotManager *risksnapshot.Manager,
	priceReader guardrailPriceReader,
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

	quantity := orderContext.Quantity
	if strings.TrimSpace(input.Quantity) != "" {
		parsed, err := decimal.NewFromString(input.Quantity)
		if err != nil {
			return nil, guardrailError("quantity must be a valid decimal")
		}
		quantity = parsed
	}
	if resolved.HasMaxOrderQuantity() && quantity.GreaterThan(resolved.MaxOrderQuantity) {
		return nil, guardrailError(fmt.Sprintf("quantity %s exceeds maxOrderQuantity %s", quantity.String(), resolved.MaxOrderQuantity.String()))
	}
	if resolved.HasMaxOrderNotional() {
		price, err := referencePriceForModifyOrder(ctx, priceReader, input, orderContext)
		if err != nil {
			return nil, guardrailError(err.Error())
		}
		notional := quantity.Mul(price)
		if notional.GreaterThan(resolved.MaxOrderNotional) {
			return nil, guardrailError(fmt.Sprintf("order notional %s exceeds maxOrderNotional %s", notional.String(), resolved.MaxOrderNotional.String()))
		}
	}
	if quantity.Equal(orderContext.Quantity) || (!resolved.HasMaxPositionQuantity() && !resolved.HasMaxPositionNotional()) || orderContext.ReduceOnly {
		return orderContext, nil
	}
	filledBeforeModify := orderContext.Quantity.Sub(orderContext.RemainingQuantity)
	modifiedRemainingQuantity := decimal.Max(quantity.Sub(filledBeforeModify), decimal.Zero)

	if resolved.HasMaxPositionQuantity() {
		if err := enforcePositionCap(snapshot, orderContext.Symbol, orderContext.Side, modifiedRemainingQuantity, resolved.MaxPositionQuantity, orderContext); err != nil {
			return nil, guardrailError(err.Error())
		}
	}
	if resolved.HasMaxPositionNotional() {
		markPrice, err := marketReferencePrice(ctx, priceReader, orderContext.Symbol)
		if err != nil {
			return nil, guardrailError(err.Error())
		}
		if err := enforcePositionNotionalCap(snapshot, orderContext.Symbol, orderContext.Side, modifiedRemainingQuantity, markPrice, resolved.MaxPositionNotional, orderContext); err != nil {
			return nil, guardrailError(err.Error())
		}
	}
	return orderContext, nil
}

func referencePriceForModifyOrder(ctx context.Context, priceReader guardrailPriceReader, input modifyOrderInput, orderContext *risksnapshot.Order) (decimal.Decimal, error) {
	if strings.TrimSpace(input.Price) != "" {
		return parsePositiveGuardrailDecimal(input.Price, "price")
	}
	if strings.TrimSpace(input.TriggerPrice) != "" {
		return parsePositiveGuardrailDecimal(input.TriggerPrice, "triggerPrice")
	}
	if orderContext != nil && orderContext.Price.GreaterThan(decimal.Zero) {
		return orderContext.Price, nil
	}
	if orderContext == nil {
		return decimal.Zero, fmt.Errorf("open order not found in current subaccount session")
	}
	return marketReferencePrice(ctx, priceReader, orderContext.Symbol)
}

func parsePositiveGuardrailDecimal(raw string, fieldName string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return decimal.Zero, fmt.Errorf("%s must be a valid decimal", fieldName)
	}
	if value.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("%s must be greater than zero", fieldName)
	}
	return value, nil
}

func marketReferencePrice(ctx context.Context, priceReader guardrailPriceReader, symbol string) (decimal.Decimal, error) {
	if priceReader == nil {
		return decimal.Zero, fmt.Errorf("market price reader is unavailable for notional guardrail enforcement")
	}
	prices, err := priceReader.GetMarketPrices(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("market price lookup failed: %s", err.Error())
	}
	normalizedSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	price, ok := prices[normalizedSymbol]
	if !ok {
		for candidateSymbol, candidatePrice := range prices {
			if strings.EqualFold(candidateSymbol, normalizedSymbol) {
				price = candidatePrice
				ok = true
				break
			}
		}
	}
	if !ok {
		return decimal.Zero, fmt.Errorf("market price for %s is unavailable", symbol)
	}
	for _, candidate := range []string{price.MarkPrice, price.IndexPrice, price.LastPrice} {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		parsed, err := parsePositiveGuardrailDecimal(candidate, "market price")
		if err == nil {
			return parsed, nil
		}
	}
	return decimal.Zero, fmt.Errorf("market price for %s is unavailable", symbol)
}

func referencePriceForOrder(ctx context.Context, priceReader guardrailPriceReader, normalized normalizedOrderOutput) (decimal.Decimal, error) {
	if strings.TrimSpace(normalized.Price) != "" {
		return parsePositiveGuardrailDecimal(normalized.Price, "price")
	}
	if strings.TrimSpace(normalized.TriggerPrice) != "" {
		return parsePositiveGuardrailDecimal(normalized.TriggerPrice, "triggerPrice")
	}
	return marketReferencePrice(ctx, priceReader, normalized.Symbol)
}

func enforcePositionNotionalCap(
	snapshot *risksnapshot.Snapshot,
	symbol string,
	side string,
	quantity decimal.Decimal,
	referencePrice decimal.Decimal,
	maxPositionNotional decimal.Decimal,
	excludedOrder *risksnapshot.Order,
) error {
	worstCaseQuantity, err := worstCasePositionQuantity(snapshot, symbol, side, quantity, excludedOrder)
	if err != nil {
		return err
	}
	worstCaseNotional := worstCaseQuantity.Mul(referencePrice)
	if worstCaseNotional.GreaterThan(maxPositionNotional) {
		return fmt.Errorf("worst-case position notional %s exceeds maxPositionNotional %s", worstCaseNotional.String(), maxPositionNotional.String())
	}
	return nil
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
	if resolved.IsSymbolAllowed(guardrails.WildcardAll) {
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
	ctx context.Context,
	priceReader guardrailPriceReader,
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
	if resolved.HasMaxOrderNotional() {
		price, err := referencePriceForOrder(ctx, priceReader, normalized)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			notional := quantity.Mul(price)
			if notional.GreaterThan(resolved.MaxOrderNotional) {
				errors = append(errors, fmt.Sprintf("order notional %s exceeds maxOrderNotional %s", notional.String(), resolved.MaxOrderNotional.String()))
			}
		}
	}
	if normalized.ReduceOnly || (!resolved.HasMaxPositionQuantity() && !resolved.HasMaxPositionNotional()) {
		return errors
	}

	if resolved.HasMaxPositionQuantity() {
		if err := enforcePositionCap(snapshot, normalized.Symbol, normalized.Side, quantity, resolved.MaxPositionQuantity, nil); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if resolved.HasMaxPositionNotional() {
		price, err := marketReferencePrice(ctx, priceReader, normalized.Symbol)
		if err != nil {
			errors = append(errors, err.Error())
		} else if err := enforcePositionNotionalCap(snapshot, normalized.Symbol, normalized.Side, quantity, price, resolved.MaxPositionNotional, nil); err != nil {
			errors = append(errors, err.Error())
		}
	}
	return errors
}

// Runs order checks and returns structured violations for rejection cards.
// String-only callers use the parallel error helper.
func guardrailOrderViolations(
	ctx context.Context,
	priceReader guardrailPriceReader,
	resolved *guardrails.Resolved,
	snapshot *risksnapshot.Snapshot,
	normalized normalizedOrderOutput,
) []guardrailViolation {
	out := make([]guardrailViolation, 0, 4)
	if resolved.IsReadOnly() {
		out = append(out, guardrailViolation{
			Reason:   "session is read_only and does not permit order placement",
			Field:    guardrailFieldReadOnly,
			Symbol:   normalized.Symbol,
			Side:     normalized.Side,
			Resolved: resolved,
		})
		return out
	}
	if !resolved.IsSymbolAllowed(normalized.Symbol) {
		out = append(out, guardrailViolation{
			Reason:   fmt.Sprintf("symbol %s is not allowed for this session", normalized.Symbol),
			Field:    guardrailFieldSymbolNotAllowed,
			Symbol:   normalized.Symbol,
			Side:     normalized.Side,
			Resolved: resolved,
		})
	}
	if !resolved.IsOrderTypeAllowed(normalized.Type) {
		out = append(out, guardrailViolation{
			Reason:   fmt.Sprintf("order type %s is not allowed for this session", normalized.Type),
			Field:    guardrailFieldOrderTypeBlocked,
			Symbol:   normalized.Symbol,
			Side:     normalized.Side,
			Resolved: resolved,
		})
	}
	quantity, err := decimal.NewFromString(normalized.Quantity)
	if err != nil {
		out = append(out, guardrailViolation{
			Reason:   "quantity must be a valid decimal",
			Field:    guardrailFieldOther,
			Symbol:   normalized.Symbol,
			Side:     normalized.Side,
			Resolved: resolved,
		})
		return out
	}
	if resolved.HasMaxOrderQuantity() && quantity.GreaterThan(resolved.MaxOrderQuantity) {
		out = append(out, guardrailViolation{
			Reason:       fmt.Sprintf("quantity %s exceeds maxOrderQuantity %s", quantity.String(), resolved.MaxOrderQuantity.String()),
			Field:        guardrailFieldOrderQuantity,
			SubmittedQty: quantity,
			Limit:        resolved.MaxOrderQuantity,
			Symbol:       normalized.Symbol,
			Side:         normalized.Side,
			Resolved:     resolved,
		})
	}
	if resolved.HasMaxOrderNotional() {
		price, err := referencePriceForOrder(ctx, priceReader, normalized)
		if err == nil {
			notional := quantity.Mul(price)
			if notional.GreaterThan(resolved.MaxOrderNotional) {
				out = append(out, guardrailViolation{
					Reason:       fmt.Sprintf("order notional %s exceeds maxOrderNotional %s", notional.String(), resolved.MaxOrderNotional.String()),
					Field:        guardrailFieldOrderNotional,
					SubmittedNot: notional,
					Limit:        resolved.MaxOrderNotional,
					Symbol:       normalized.Symbol,
					Side:         normalized.Side,
					Resolved:     resolved,
				})
			}
		}
	}
	if normalized.ReduceOnly || (!resolved.HasMaxPositionQuantity() && !resolved.HasMaxPositionNotional()) {
		return out
	}
	if resolved.HasMaxPositionQuantity() {
		if err := enforcePositionCap(snapshot, normalized.Symbol, normalized.Side, quantity, resolved.MaxPositionQuantity, nil); err != nil {
			out = append(out, guardrailViolation{
				Reason:   err.Error(),
				Field:    guardrailFieldPositionQuantity,
				Limit:    resolved.MaxPositionQuantity,
				Symbol:   normalized.Symbol,
				Side:     normalized.Side,
				Resolved: resolved,
			})
		}
	}
	if resolved.HasMaxPositionNotional() {
		price, err := marketReferencePrice(ctx, priceReader, normalized.Symbol)
		if err != nil {
			out = append(out, guardrailViolation{
				Reason:   err.Error(),
				Field:    guardrailFieldOther,
				Symbol:   normalized.Symbol,
				Side:     normalized.Side,
				Resolved: resolved,
			})
		} else if err := enforcePositionNotionalCap(snapshot, normalized.Symbol, normalized.Side, quantity, price, resolved.MaxPositionNotional, nil); err != nil {
			out = append(out, guardrailViolation{
				Reason:   err.Error(),
				Field:    guardrailFieldPositionNotional,
				Limit:    resolved.MaxPositionNotional,
				Symbol:   normalized.Symbol,
				Side:     normalized.Side,
				Resolved: resolved,
			})
		}
	}
	return out
}

// Returns the primary typed violation while preserving all reasons.
func firstViolationOrJoined(violations []guardrailViolation) error {
	if len(violations) == 0 {
		return nil
	}
	primary := violations[0]
	reasons := make([]string, 0, len(violations))
	for _, v := range violations {
		reasons = append(reasons, v.Reason)
	}
	primary.Reason = strings.Join(reasons, "; ")
	v := primary
	return &v
}

func enforcePositionCap(
	snapshot *risksnapshot.Snapshot,
	symbol string,
	side string,
	quantity decimal.Decimal,
	maxPositionQuantity decimal.Decimal,
	excludedOrder *risksnapshot.Order,
) error {
	worstCase, err := worstCasePositionQuantity(snapshot, symbol, side, quantity, excludedOrder)
	if err != nil {
		return err
	}
	if worstCase.GreaterThan(maxPositionQuantity) {
		return fmt.Errorf("worst-case position quantity %s exceeds maxPositionQuantity %s", worstCase.String(), maxPositionQuantity.String())
	}
	return nil
}

func worstCasePositionQuantity(
	snapshot *risksnapshot.Snapshot,
	symbol string,
	side string,
	quantity decimal.Decimal,
	excludedOrder *risksnapshot.Order,
) (decimal.Decimal, error) {
	if snapshot == nil {
		return decimal.Zero, fmt.Errorf("risk snapshot is unavailable for guardrail enforcement")
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
	return worstCase, nil
}
