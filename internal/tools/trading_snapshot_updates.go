package tools

import (
	"strconv"
	"strings"

	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/shopspring/decimal"
)

func applyCancelAllSnapshot(sessionID string, snapshotManager *risksnapshot.Manager, symbols []string, orders []cancelOrderItemOutput) {
	if snapshotManager == nil || sessionID == "" {
		return
	}
	successfulOrders := successfulCancelledOrders(orders)
	if len(successfulOrders) == 0 {
		return
	}
	if len(symbols) == 1 && symbols[0] == "*" {
		for _, order := range successfulOrders {
			if order.Symbol == "" {
				continue
			}
			snapshotManager.RemoveOrdersBySymbol(sessionID, order.Symbol)
		}
		return
	}
	refs := make([]risksnapshot.OrderRef, 0, len(successfulOrders))
	for _, order := range successfulOrders {
		refs = append(refs, orderRefFromOutput(order.OrderID))
	}
	snapshotManager.RemoveOrders(sessionID, refs)
}

func applyCancelledOrdersSnapshot(
	sessionID string,
	snapshotManager *risksnapshot.Manager,
	orders []cancelOrderItemOutput,
	_ *risksnapshot.Order,
) {
	if snapshotManager == nil || sessionID == "" {
		return
	}
	successfulOrders := successfulCancelledOrders(orders)
	if len(successfulOrders) == 0 {
		return
	}
	refs := make([]risksnapshot.OrderRef, 0, len(successfulOrders))
	for _, order := range successfulOrders {
		refs = append(refs, orderRefFromOutput(order.OrderID))
	}
	snapshotManager.RemoveOrders(sessionID, refs)
}

func applyModifiedOrderSnapshot(
	sessionID string,
	snapshotManager *risksnapshot.Manager,
	existingOrder *risksnapshot.Order,
	input modifyOrderInput,
	result modifyOrderOutput,
) {
	if snapshotManager == nil || sessionID == "" || existingOrder == nil {
		return
	}
	if !modifyResultMutatesSnapshot(result) {
		return
	}
	if modifyResultRemovesOrder(result) {
		snapshotManager.RemoveOrders(sessionID, []risksnapshot.OrderRef{{
			ClientOrderID: existingOrder.ClientOrderID,
			VenueOrderID:  existingOrder.VenueOrderID,
		}})
		return
	}

	updated := *existingOrder
	if strings.TrimSpace(input.Quantity) != "" || result.Quantity != "" || result.Status == "PARTIALLY_FILLED" {
		totalQuantity, remainingQuantity, err := quantitiesForModifiedOrder(result, input, existingOrder)
		if err == nil {
			updated.Quantity = totalQuantity
			updated.RemainingQuantity = remainingQuantity
		}
	}
	if result.OrderID.ClientID != "" {
		updated.ClientOrderID = result.OrderID.ClientID
	}
	if result.OrderID.VenueID != 0 {
		updated.VenueOrderID = strconv.FormatUint(result.OrderID.VenueID, 10)
	}
	snapshotManager.UpsertOrder(sessionID, updated)
}

func applyPlacedOrderSnapshot(sessionID string, snapshotManager *risksnapshot.Manager, normalized normalizedOrderOutput, result placeOrderOutput) {
	if snapshotManager == nil || sessionID == "" {
		return
	}
	if !placeResultCreatesOrMaintainsLiveOrder(result) {
		return
	}

	remainingQuantity, err := remainingQuantityForPlacedOrder(normalized, result)
	if err != nil {
		return
	}
	quantity, err := decimal.NewFromString(normalized.Quantity)
	if err != nil {
		return
	}
	snapshotManager.UpsertOrder(sessionID, risksnapshot.Order{
		ClientOrderID:     firstNonEmpty(result.OrderID.ClientID, normalized.ClientOrderID),
		OrderType:         strings.ToUpper(strings.TrimSpace(normalized.Type)),
		Quantity:          quantity,
		ReduceOnly:        normalized.ReduceOnly,
		RemainingQuantity: remainingQuantity,
		Side:              strings.ToUpper(strings.TrimSpace(normalized.Side)),
		Symbol:            normalized.Symbol,
		VenueOrderID:      venueOrderIDString(result.OrderID),
	})
}

func orderRefFromOutput(orderID orderIDOutput) risksnapshot.OrderRef {
	return risksnapshot.OrderRef{
		ClientOrderID: orderID.ClientID,
		VenueOrderID:  venueOrderIDString(orderID),
	}
}

func venueOrderIDString(orderID orderIDOutput) string {
	if orderID.VenueID == 0 {
		return ""
	}
	return strconv.FormatUint(orderID.VenueID, 10)
}

func modifyResultMutatesSnapshot(result modifyOrderOutput) bool {
	return result.ErrorCode == "" && result.ErrorMessage == ""
}

func modifyResultRemovesOrder(result modifyOrderOutput) bool {
	switch result.Status {
	case "CANCELLED", "EXPIRED", "FILLED", "REJECTED":
		return true
	default:
		return false
	}
}

func placeResultCreatesOrMaintainsLiveOrder(result placeOrderOutput) bool {
	if !result.IsSuccess || result.ErrorCode != "" {
		return false
	}
	switch result.Status {
	case "ACCEPTED", "MODIFIED", "PARTIALLY_FILLED", "PENDING":
		return true
	default:
		return false
	}
}

func quantitiesForModifiedOrder(
	result modifyOrderOutput,
	input modifyOrderInput,
	existingOrder *risksnapshot.Order,
) (decimal.Decimal, decimal.Decimal, error) {
	totalQuantity := existingOrder.Quantity
	if result.Quantity != "" {
		parsed, err := decimal.NewFromString(result.Quantity)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		totalQuantity = parsed
	} else if strings.TrimSpace(input.Quantity) != "" {
		parsed, err := decimal.NewFromString(input.Quantity)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		totalQuantity = parsed
	}

	if result.Status == "PARTIALLY_FILLED" && result.CumulativeFillQty != "" {
		filled, err := decimal.NewFromString(result.CumulativeFillQty)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		return totalQuantity, decimal.Max(totalQuantity.Sub(filled), decimal.Zero), nil
	}

	filledBeforeModify := existingOrder.Quantity.Sub(existingOrder.RemainingQuantity)
	return totalQuantity, decimal.Max(totalQuantity.Sub(filledBeforeModify), decimal.Zero), nil
}

func remainingQuantityForPlacedOrder(normalized normalizedOrderOutput, result placeOrderOutput) (decimal.Decimal, error) {
	quantity, err := decimal.NewFromString(normalized.Quantity)
	if err != nil {
		return decimal.Zero, err
	}
	if result.Status != "PARTIALLY_FILLED" || result.CumQty == "" {
		return quantity, nil
	}
	filled, err := decimal.NewFromString(result.CumQty)
	if err != nil {
		return decimal.Zero, err
	}
	return quantity.Sub(filled), nil
}

func successfulCancelledOrders(orders []cancelOrderItemOutput) []cancelOrderItemOutput {
	out := make([]cancelOrderItemOutput, 0, len(orders))
	for _, order := range orders {
		if order.ErrorCode != "" || order.ErrorMessage != "" {
			continue
		}
		out = append(out, order)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
