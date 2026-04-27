package tools

import (
	"strconv"
	"strings"

	"github.com/synthetixio/synthetix-go/types"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

// Derives the tool-facing placeOrder shape from an upstream
// per-order OrderStatus. The REST wire format is a tagged bag
// (Resting / Filled / Canceled / error), so the
// mapper picks the first populated branch and synthesises the
// Status / Accepted / IsSuccess / AvgPrice / CumQty fields the
// existing output schema (and snapshot-updater) expects. Symbol
// and OrigQty are not on the wire per entry — the caller passes
// them from the normalised input so the output still carries the
// context a downstream agent needs.
func mapPlaceOrderResultREST(status types.OrderStatus, symbol, origQty string) placeOrderOutput {
	out := placeOrderOutput{
		Meta:    newResponseMeta(string(session.AuthModeAuthenticated)),
		OrigQty: origQty,
		Symbol:  symbol,
	}

	switch {
	case status.Filled != nil:
		out.Status = "FILLED"
		out.IsSuccess = true
		out.OrderID = orderIDFromIdentifier(status.Filled.OrderID)
		out.AvgPrice = status.Filled.AvgPrice
		out.CumQty = status.Filled.TotalSize
	case status.Resting != nil:
		out.Status = "ACCEPTED"
		out.IsSuccess = true
		out.OrderID = orderIDFromIdentifier(status.Resting.OrderID)
	case status.Canceled != nil:
		// On a place request, Canceled means the order was
		// accepted and then immediately cancelled (e.g. IOC with
		// no matchable resting depth). Still accepted, but
		// nothing rests on the book.
		out.Status = "CANCELLED"
		out.IsSuccess = true
		out.OrderID = orderIDFromIdentifier(status.Canceled.OrderID)
	default:
		out.Status = "REJECTED"
		out.ErrorCode = status.ErrorCode
		out.Message = status.Error
		if status.ErrorOrderId != nil {
			out.OrderID = orderIDFromIdentifier(status.ErrorOrderId.Order)
		}
	}

	phase, followUp := classifyPlaceOrderPhase(out.Status, out.IsSuccess, out.ErrorCode)
	out.Phase = phase
	out.Accepted = phase == OrderPhaseAccepted
	out.FollowUp = followUp
	out.ErrorDetail = errorDetailForCode(out.ErrorCode)
	return out
}

// Flattens the cancelOrders / cancelOrdersByCloid response into
// the existing cancelOrderOutput shape. Symbol is empty because
// the wire entries don't carry it; downstream snapshot logic only
// needs symbol for wildcard cancel-all, which uses a different
// mapper.
func mapCancelOrderResultREST(statuses []types.OrderStatus) cancelOrderOutput {
	orders := make([]cancelOrderItemOutput, 0, len(statuses))
	for _, s := range statuses {
		orders = append(orders, cancelItemFromOrderStatus(s))
	}
	return cancelOrderOutput{
		Meta:   newResponseMeta(string(session.AuthModeAuthenticated)),
		Orders: orders,
	}
}

// Flattens the cancelAllOrders response (bare array with
// per-entry message + symbol) into cancelOrderOutput. A non-empty
// Message is treated as an error and surfaced via ErrorMessage so
// the snapshot updater's empty-error filter continues to work; the
// upstream handler doesn't emit a machine code.
func mapCancelAllOrdersResultREST(items types.CancelAllOrdersResponse) cancelOrderOutput {
	orders := make([]cancelOrderItemOutput, 0, len(items))
	for _, item := range items {
		out := cancelOrderItemOutput{
			OrderID: orderIDFromIdentifier(item.Order),
		}
		if item.Symbol != nil {
			out.Symbol = *item.Symbol
		}
		if strings.TrimSpace(item.Message) != "" {
			out.ErrorMessage = item.Message
		}
		orders = append(orders, out)
	}
	return cancelOrderOutput{
		Meta:   newResponseMeta(string(session.AuthModeAuthenticated)),
		Orders: orders,
	}
}

func cancelItemFromOrderStatus(s types.OrderStatus) cancelOrderItemOutput {
	out := cancelOrderItemOutput{}
	switch {
	case s.Canceled != nil:
		out.OrderID = orderIDFromIdentifier(s.Canceled.OrderID)
	case s.Resting != nil:
		out.OrderID = orderIDFromIdentifier(s.Resting.OrderID)
	case s.Filled != nil:
		out.OrderID = orderIDFromIdentifier(s.Filled.OrderID)
	default:
		out.ErrorCode = s.ErrorCode
		out.ErrorMessage = s.Error
		out.ErrorDetail = errorDetailForCode(s.ErrorCode)
		if s.ErrorOrderId != nil {
			out.OrderID = orderIDFromIdentifier(s.ErrorOrderId.Order)
		}
	}
	return out
}

// Coerces the wire-level composite order id (venueId as a decimal
// string, optional clientId) into the numeric-venue output the
// snapshot updater and tool schema expect. An unparsable venueId
// is dropped to zero rather than failing the whole response — the
// upstream contract promises a decimal string but staging
// downgrades have been observed to hand back empty strings on
// errored statuses.
func orderIDFromIdentifier(id types.OrderIdentifier) orderIDOutput {
	venue, _ := strconv.ParseUint(strings.TrimSpace(id.VenueID), 10, 64)
	return orderIDOutput{
		ClientID: id.ClientID,
		VenueID:  venue,
	}
}

// Translates the flat modifyOrder wire response into the existing
// modifyOrderOutput schema. The wire format gained Order (composite
// id) alongside a deprecated flat OrderID, and splits Error +
// ErrorCode; field names on the output (AveragePrice vs AvgPrice,
// CumulativeFillQty vs CumQty) are an intentional schema choice
// retained for compatibility to avoid breaking the public tool
// surface — the match-12 output-drift policy covers that divergence
// via meta.migration, not a rename.
func mapModifyOrderResultREST(resp types.ModifyOrderResponse) modifyOrderOutput {
	return modifyOrderOutput{
		Meta:              newResponseMeta(string(session.AuthModeAuthenticated)),
		AveragePrice:      resp.AvgPrice,
		CumulativeFillQty: resp.CumQty,
		ErrorCode:         resp.ErrorCode,
		ErrorDetail:       errorDetailForCode(resp.ErrorCode),
		ErrorMessage:      resp.Error,
		OrderID:           orderIDFromIdentifier(resp.Order),
		Price:             resp.Price,
		Quantity:          resp.Quantity,
		Status:            resp.Status,
		TriggerPrice:      resp.TriggerPrice,
	}
}
