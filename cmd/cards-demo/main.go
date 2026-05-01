// Command cards-demo renders the full gallery of agent-trader
// card variants to stdout. It exists as a zero-friction visual
// smoke test: run `go run ./cmd/cards-demo` in your real terminal
// (Claude Code, iTerm, Alacritty) and eyeball that emoji, box
// drawing, and alignment all render correctly. Reviewers of PRs
// that touch internal/cards/ should run this before approving.
//
// It is not part of the MCP server binary and is safe to remove
// in a future cleanup once the card surface is stable.
package main

import (
	"fmt"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
)

func main() {
	win := cards.Card{
		Status: cards.StatusPositive,
		Title:  "CLOSED LONG ▲ BTC-USDT",
		Sections: []cards.Section{
			{Rows: []cards.Row{
				{Label: "Held for:", Value: "~15h 47min", Hint: "Opened: 2026-04-30 06:24 UTC"},
			}},
			{Rows: []cards.Row{
				{Label: "Side:", Value: "LONG ▲"},
				{Label: "Quantity:", Value: "0.001 BTC", Hint: "Notional: $76.30"},
				{Label: "Entry:", Value: "$75,683"},
				{Label: "Exit:", Value: "$76,300", Hint: "▲ +$617   (+0.82%)"},
			}},
			{Rows: []cards.Row{
				{Label: "Realized PnL:", Value: "+$0.62", Hint: "🟢 +0.82%"},
				{Label: "Fees paid:", Value: "$0.031", Hint: "(0.04% taker)"},
				{Label: "Net PnL:", Value: "+$0.589"},
			}},
		},
		Footnote: "Funding paid: -$0.066  (~15h at ~51% APY — longs paid shorts)",
	}
	lose := cards.Card{
		Status: cards.StatusNegative,
		Title:  "CLOSED LONG ▼ BTC-USDT",
		Sections: []cards.Section{
			{Rows: []cards.Row{
				{Label: "Held for:", Value: "~2h 13min", Hint: "Opened: 2026-05-01 18:45 UTC"},
			}},
			{Rows: []cards.Row{
				{Label: "Entry:", Value: "$76,420"},
				{Label: "Exit:", Value: "$75,880", Hint: "▼ -$540   (-0.71%)"},
			}},
			{Rows: []cards.Row{
				{Label: "Realized PnL:", Value: "-$0.54", Hint: "🔴 -0.71%"},
				{Label: "Net PnL:", Value: "-$0.571", Hint: "(after $0.031 fees)"},
			}},
		},
	}
	warn := cards.Card{
		Status: cards.StatusWarning,
		Title:  "GUARDRAIL TRIPPED",
		Sections: []cards.Section{{Rows: []cards.Row{
			{Label: "Guardrail:", Value: "max_order_notional"},
			{Label: "Requested:", Value: "$12,500", Hint: "⚠ +$2,500 over"},
			{Label: "Limit:", Value: "$10,000"},
		}}},
		Footnote: "Reduce size to ≤0.131 BTC or raise the cap in set_guardrails.",
	}
	crit := cards.Card{
		Status: cards.StatusCritical,
		Title:  "LIQUIDATED LONG BTC-USDT",
		Sections: []cards.Section{{Rows: []cards.Row{
			{Label: "Liq price:", Value: "$71,200"},
			{Label: "Held for:", Value: "~3h 21min"},
			{Label: "Realized PnL:", Value: "-$4.82", Hint: "🔥 -5.88%"},
		}}},
	}
	fmt.Println(win.Render())
	fmt.Println(lose.Render())
	fmt.Println(warn.Render())
	fmt.Println(crit.Render())
}
