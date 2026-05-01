package tools

import (
	"strconv"
	"strings"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// Formats base-10 IDs for card rows.
func uint64ToString(v uint64) string {
	return strconv.FormatUint(v, 10)
}

// Summarizes order outcome, submitted intent, and next action in one card.
// The card makes filled, resting, and rejected states visually distinct.
func renderPlaceOrderCard(normalized normalizedOrderOutput, result placeOrderOutput) string {
	status, title := placeOrderCardStatusAndTitle(normalized, result)

	// Show submitted intent before outcome-specific rows.
	symbol := firstNonEmptyLocal(result.Symbol, normalized.Symbol)
	side := strings.ToUpper(strings.TrimSpace(normalized.Side))
	if side == "" {
		side = "?"
	}
	orderType := strings.ToUpper(strings.TrimSpace(normalized.Type))
	if orderType == "" {
		orderType = "?"
	}
	priceFloat, _ := decimalOrZero(normalized.Price).Float64()

	requestedValue := orderType
	if normalized.ReduceOnly {
		requestedValue = requestedValue + " (reduce-only)"
	}
	if normalized.PostOnly {
		requestedValue = requestedValue + " (post-only)"
	}
	if tif := strings.ToUpper(strings.TrimSpace(normalized.TimeInForce)); tif != "" {
		requestedValue = requestedValue + " " + tif
	}

	quantityValue := formatQuantity(decimalOrZero(normalized.Quantity)) + " " + baseAsset(symbol)
	if priceFloat > 0 {
		quantityValue = quantityValue + "  @  " + cards.USD(priceFloat, priceDecimals(priceFloat))
	}

	baseRows := []cards.Row{
		{Label: "Symbol:", Value: symbol, Hint: sidePill(side)},
		{Label: "Type:", Value: requestedValue},
		{Label: "Requested:", Value: quantityValue},
	}

	// Outcome rows depend on terminal status.
	outcomeRows := placeOrderOutcomeRows(result, side, symbol)

	card := cards.Card{
		Status: status,
		Title:  title,
		Sections: []cards.Section{
			{Rows: baseRows},
			{Rows: outcomeRows},
		},
	}

	if len(result.FollowUp) > 0 {
		card.Footnote = "Next: " + truncateForRow(strings.Join(result.FollowUp, " · "), 72)
	}

	return card.Render()
}

// Maps an outcome to a header color and title verb.
func placeOrderCardStatusAndTitle(normalized normalizedOrderOutput, result placeOrderOutput) (cards.Status, string) {
	side := strings.ToUpper(strings.TrimSpace(normalized.Side))
	symbol := firstNonEmptyLocal(result.Symbol, normalized.Symbol)

	switch result.Status {
	case "FILLED":
		arrow := "▲"
		if side == "SELL" {
			arrow = "▼"
		}
		return cards.StatusPositive, "FILLED " + side + " " + arrow + " " + symbol
	case "ACCEPTED":
		return cards.StatusNeutral, "RESTING " + side + " " + symbol
	case "CANCELLED":
		// IOC that didn't match, or a post-only that would have
		// crossed — the order was accepted then immediately
		// killed. Warning, not error, because nothing is broken.
		return cards.StatusWarning, "CANCELLED " + side + " " + symbol
	case "REJECTED":
		return cards.StatusNegative, "REJECTED " + side + " " + symbol
	default:
		return cards.StatusNeutral, "SUBMITTED " + side + " " + symbol
	}
}

// Produces a short LONG/SHORT chip for the symbol row.
func sidePill(side string) string {
	switch side {
	case "BUY":
		return "LONG ▲"
	case "SELL":
		return "SHORT ▼"
	default:
		return side
	}
}

// Picks per-status row shapes and hides irrelevant zero fields.
func placeOrderOutcomeRows(result placeOrderOutput, side, symbol string) []cards.Row {
	_ = side
	switch result.Status {
	case "FILLED":
		avgFloat, _ := decimalOrZero(result.AvgPrice).Float64()
		cumFloat, _ := decimalOrZero(result.CumQty).Float64()
		origFloat, _ := decimalOrZero(result.OrigQty).Float64()
		notional := avgFloat * cumFloat

		fillLabel := "Fully filled"
		fillHint := ""
		if origFloat > 0 && cumFloat > 0 && cumFloat < origFloat {
			// Partial fills need explicit labeling to avoid "whole order" confusion.
			fillLabel = "Partially filled"
			fillHint = "filled " + formatQuantity(decimalOrZero(result.CumQty)) + " of " +
				formatQuantity(decimalOrZero(result.OrigQty)) + " " + baseAsset(symbol)
		}

		return []cards.Row{
			{Label: "Status:", Value: fillLabel, Hint: fillHint},
			{Label: "Avg fill:", Value: cards.USD(avgFloat, priceDecimals(avgFloat)), Hint: "Filled qty: " + formatQuantity(decimalOrZero(result.CumQty)) + " " + baseAsset(symbol)},
			{Label: "Notional:", Value: cards.USD(notional, 2)},
		}
	case "ACCEPTED":
		rows := []cards.Row{
			{Label: "Status:", Value: "Resting on book"},
		}
		if result.OrderID.VenueID != 0 || result.OrderID.ClientID != "" {
			idLabel := ""
			switch {
			case result.OrderID.VenueID != 0:
				idLabel = "venue:" + uint64ToString(result.OrderID.VenueID)
			default:
				idLabel = "client:" + result.OrderID.ClientID
			}
			rows = append(rows, cards.Row{Label: "Order ID:", Value: truncateForRow(idLabel, 48)})
		}
		return rows
	case "CANCELLED":
		msg := result.Message
		if msg == "" {
			msg = "Accepted, then immediately cancelled (IOC/post-only)."
		}
		return []cards.Row{
			{Label: "Status:", Value: "Cancelled"},
			{Label: "Detail:", Value: truncateForRow(msg, 52)},
		}
	case "REJECTED":
		msg := result.Message
		if msg == "" {
			msg = "No detail provided by the matching engine."
		}
		rows := []cards.Row{
			{Label: "Status:", Value: "Rejected", Hint: result.ErrorCode},
			{Label: "Reason:", Value: truncateForRow(msg, 52)},
		}
		if result.ErrorDetail != nil && len(result.ErrorDetail.Remediation) > 0 {
			rows = append(rows, cards.Row{Label: "Fix:", Value: truncateForRow(result.ErrorDetail.Remediation[0], 52)})
		}
		return rows
	default:
		return []cards.Row{
			{Label: "Status:", Value: firstNonEmptyLocal(result.Status, "SUBMITTED")},
		}
	}
}
