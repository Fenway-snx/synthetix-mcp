package tools

import (
	"strconv"
	"strings"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

// uint64ToString is a trivial base-10 formatter used only by the
// place-order card's Order-ID row. Inlined here to keep the card
// file's imports minimal — strconv is the right std-lib answer
// but the rest of the card only needs strings.
func uint64ToString(v uint64) string {
	return strconv.FormatUint(v, 10)
}

// renderPlaceOrderCard summarises the outcome of a place_order
// call in one compact card. The card has three jobs, in order
// of importance:
//
//  1. Make it unmistakable whether the order FILLED, is RESTING
//     on the book, or was REJECTED. A trader scanning their feed
//     should never have to parse JSON to answer "did it go
//     through?".
//  2. Surface the numbers that matter for the state it's in —
//     avg fill price + cumulative qty for fills, resting price
//     + quantity for rests, error code + human message for
//     rejects.
//  3. Prompt the natural next step (from the FollowUp hints the
//     tool layer already populates) so the agent has a concrete
//     next call rather than having to re-plan.
//
// normalized carries the order-as-submitted (side, type, price,
// quantity) so the card can show what was *intended* even when
// the fill echoes a different price or the reject returns none
// of the original context.
func renderPlaceOrderCard(normalized normalizedOrderOutput, result placeOrderOutput) string {
	status, title := placeOrderCardStatusAndTitle(normalized, result)

	// Base section: the always-present "here's what you tried
	// to do" context. Side + type + quantity @ price anchors
	// the reader whether the outcome was a fill, rest, or
	// reject.
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

	// Outcome section: what actually happened. The exact rows
	// depend on the terminal status because the semantically
	// relevant fields differ.
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

// placeOrderCardStatusAndTitle maps an outcome to a header colour
// and title verb. Keep the logic simple and obvious: a single
// spot to re-read when debugging a "why is the card green?"
// report from a user.
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

// sidePill produces a short visual chip for the Symbol row hint
// — "LONG ▲" for buys, "SHORT ▼" for sells. Puts a trading-
// native label next to the symbol so the agent doesn't have to
// mentally remap BUY/SELL to long/short.
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

// placeOrderOutcomeRows picks per-status row shapes. We don't
// try to present every field in every state: showing "AvgPrice:
// 0" on a rejected order is worse than hiding the row.
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
			// Partial fill: the match engine did SOME of the
			// order and the rest cancelled (or is left resting,
			// but the status-level here is FILLED so the rest
			// was killed). Spell this out — partials are a
			// common source of "did my whole order go?" bugs.
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
