package tools

import (
	"strconv"
	"strings"

	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/synthetixio/synthetix-go/types"
)

// Derives the tool-facing order result from an upstream tagged status.
// Caller-supplied symbol and quantity preserve context missing on the wire.
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
		// Accepted but immediately cancelled, such as unmatched IOC.
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

// Flattens order-cancel responses into the existing output shape.
// Symbol stays empty because these wire entries do not carry it.
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

// Flattens cancel-all responses and treats non-empty messages as errors.
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

// Coerces wire order identifiers into the numeric-venue output schema.
// Unparsable venue IDs drop to zero instead of failing the whole response.
func orderIDFromIdentifier(id types.OrderIdentifier) orderIDOutput {
	venue, _ := strconv.ParseUint(strings.TrimSpace(id.VenueID), 10, 64)
	return orderIDOutput{
		ClientID: id.ClientID,
		VenueID:  venue,
	}
}

// Translates modify responses into the existing public output schema.
// Field names are retained for compatibility.
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
