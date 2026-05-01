package tools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
)

// Carries structured context for guardrail rejection cards.
// This avoids re-parsing upstream error strings at render time.
type guardrailViolation struct {
	Reason       string
	Field        guardrailField
	SubmittedQty decimal.Decimal
	SubmittedNot decimal.Decimal
	Limit        decimal.Decimal
	Symbol       string
	Side         string
	Resolved     *guardrails.Resolved
}

// Labels which configured limit produced the rejection.
// The value drives both headline and remediation rows.
type guardrailField string

const (
	guardrailFieldOrderQuantity    guardrailField = "MAX_ORDER_QUANTITY"
	guardrailFieldOrderNotional    guardrailField = "MAX_ORDER_NOTIONAL"
	guardrailFieldPositionQuantity guardrailField = "MAX_POSITION_QUANTITY"
	guardrailFieldPositionNotional guardrailField = "MAX_POSITION_NOTIONAL"
	guardrailFieldSymbolNotAllowed guardrailField = "SYMBOL_NOT_ALLOWED"
	guardrailFieldOrderTypeBlocked guardrailField = "ORDER_TYPE_BLOCKED"
	guardrailFieldReadOnly         guardrailField = "READ_ONLY_SESSION"
	guardrailFieldOther            guardrailField = "GUARDRAIL"
)

func (v *guardrailViolation) Error() string {
	if v == nil {
		return ""
	}
	return "guardrail violation: " + v.Reason
}

// Unwraps an error chain to find structured guardrail context.
func isGuardrailViolation(err error) *guardrailViolation {
	if err == nil {
		return nil
	}
	var v *guardrailViolation
	if errors.As(err, &v) {
		return v
	}
	return nil
}

// Renders a compact card explaining why a guardrail blocked an order.
// It shows the submitted value, cap, and next action without parsing JSON.
func renderGuardrailRejectionCard(v *guardrailViolation, normalized normalizedOrderOutput) string {
	if !cards.Enabled() || v == nil {
		return ""
	}
	side := strings.ToUpper(strings.TrimSpace(v.Side))
	if side == "" {
		side = strings.ToUpper(strings.TrimSpace(normalized.Side))
	}
	symbol := strings.ToUpper(strings.TrimSpace(v.Symbol))
	if symbol == "" {
		symbol = strings.ToUpper(strings.TrimSpace(normalized.Symbol))
	}
	headline := guardrailFieldHeadline(v.Field)
	title := "BLOCKED " + side
	if symbol != "" {
		title += "  " + symbol
	}
	title += "  · " + headline

	rows := guardrailRejectionDetailRows(v, normalized)
	remediation := guardrailRemediationRows(v)

	return cards.Card{
		Status: cards.StatusNegative,
		Title:  title,
		Sections: []cards.Section{
			{Rows: rows},
			{Rows: remediation},
		},
	}.Render()
}

func guardrailRejectionDetailRows(v *guardrailViolation, normalized normalizedOrderOutput) []cards.Row {
	rows := []cards.Row{
		{Label: "Reason:", Value: v.Reason, Hint: ""},
	}

	if normalized.Quantity != "" {
		rows = append(rows, cards.Row{
			Label: "Submitted qty:",
			Value: normalized.Quantity,
			Hint:  baseAsset(normalized.Symbol),
		})
	}
	if normalized.Price != "" && strings.ToUpper(normalized.Type) != "MARKET" {
		rows = append(rows, cards.Row{
			Label: "Limit price:",
			Value: "$" + normalized.Price,
			Hint:  "",
		})
	}

	if !v.SubmittedQty.IsZero() && (v.Field == guardrailFieldOrderQuantity || v.Field == guardrailFieldPositionQuantity) {
		f, _ := v.SubmittedQty.Float64()
		rows = append(rows, cards.Row{
			Label: "Tripped at:",
			Value: trimDecimalAbs(f),
			Hint:  baseAsset(normalized.Symbol),
		})
	}
	if !v.SubmittedNot.IsZero() && (v.Field == guardrailFieldOrderNotional || v.Field == guardrailFieldPositionNotional) {
		f, _ := v.SubmittedNot.Float64()
		rows = append(rows, cards.Row{
			Label: "Tripped at:",
			Value: cards.USD(absFloat(f), 0),
			Hint:  "notional",
		})
	}
	if !v.Limit.IsZero() {
		f, _ := v.Limit.Float64()
		switch v.Field {
		case guardrailFieldOrderQuantity, guardrailFieldPositionQuantity:
			rows = append(rows, cards.Row{
				Label: "Cap:",
				Value: trimDecimalAbs(f),
				Hint:  baseAsset(normalized.Symbol),
			})
		case guardrailFieldOrderNotional, guardrailFieldPositionNotional:
			rows = append(rows, cards.Row{
				Label: "Cap:",
				Value: cards.USD(absFloat(f), 0),
				Hint:  "notional",
			})
		}
	}
	return rows
}

func guardrailRemediationRows(v *guardrailViolation) []cards.Row {
	out := []cards.Row{}
	switch v.Field {
	case guardrailFieldOrderQuantity, guardrailFieldOrderNotional:
		out = append(out,
			cards.Row{Label: "Try:", Value: "split the order into smaller chunks", Hint: ""},
			cards.Row{Label: "Or:", Value: "loosen via set_guardrails", Hint: "operator action"},
		)
	case guardrailFieldPositionQuantity, guardrailFieldPositionNotional:
		out = append(out,
			cards.Row{Label: "Try:", Value: "trim or close existing exposure first", Hint: ""},
			cards.Row{Label: "Or:", Value: "raise position cap via set_guardrails", Hint: ""},
		)
	case guardrailFieldSymbolNotAllowed:
		out = append(out,
			cards.Row{Label: "Try:", Value: "use an allow-listed symbol", Hint: "see get_session"},
			cards.Row{Label: "Or:", Value: "extend allow-list via set_guardrails", Hint: ""},
		)
	case guardrailFieldOrderTypeBlocked:
		out = append(out,
			cards.Row{Label: "Try:", Value: "use an allow-listed order type", Hint: "see get_session"},
		)
	case guardrailFieldReadOnly:
		out = append(out,
			cards.Row{Label: "Try:", Value: "switch session out of read_only", Hint: "set_guardrails preset=balanced"},
		)
	default:
		out = append(out, cards.Row{Label: "Try:", Value: "review get_session.guardrails", Hint: ""})
	}
	return out
}

func guardrailFieldHeadline(f guardrailField) string {
	switch f {
	case guardrailFieldOrderQuantity:
		return "ORDER QTY OVER CAP"
	case guardrailFieldOrderNotional:
		return "ORDER NOTIONAL OVER CAP"
	case guardrailFieldPositionQuantity:
		return "POSITION QTY OVER CAP"
	case guardrailFieldPositionNotional:
		return "POSITION NOTIONAL OVER CAP"
	case guardrailFieldSymbolNotAllowed:
		return "SYMBOL NOT ALLOWED"
	case guardrailFieldOrderTypeBlocked:
		return "ORDER TYPE BLOCKED"
	case guardrailFieldReadOnly:
		return "READ-ONLY SESSION"
	default:
		return "GUARDRAIL"
	}
}

// Adds a visual guardrail card to a tool error when structured context exists.
// The zero output value supports drop-in handler use.
func guardrailRejectionResponse[Out any](err error, normalized normalizedOrderOutput) (*mcp.CallToolResult, Out, error) {
	result := toolErrorResult(err)
	out := initializedZeroOutput[Out]()
	v := isGuardrailViolation(err)
	if v == nil {
		return result, out, nil
	}
	card := renderGuardrailRejectionCard(v, normalized)
	if card == "" {
		return result, out, nil
	}
	return cards.AttachText(result, card), out, nil
}

func trimDecimalAbs(v float64) string {
	abs := absFloat(v)
	s := fmt.Sprintf("%.6f", abs)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}
