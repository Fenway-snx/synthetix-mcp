package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/cards"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
)

const (
	tradeJournalURI    = "account://trade-journal"
	tradeJournalLookbk = 14 * 24 * time.Hour
	tradeJournalLimit  = int32(500)
)

// registerTradeJournal wires the account://trade-journal resource.
// The resource is read-only and must be authenticated; it pulls the
// last 14 days of fills via the trade-history REST endpoint, then
// serves both a JSON aggregate (programmatic) and a rendered
// markdown card (human/agent-glanceable) in the same response. The
// card section is omitted when SNXMCP_CARDS_ENABLED=false.
func registerTradeJournal(server *mcp.Server, deps *tools.ToolDeps, tradeReads *tools.TradeReadClient) {
	store := deps.Store
	verifier := deps.Verifier

	server.AddResource(&mcp.Resource{
		Description: "Last 14 days of fills aggregated into a daily PnL journal: net PnL, fee burn, win-rate, per-symbol breakdown, and recent closed-trade mini-cards. Authenticated only.",
		MIMEType:    "text/markdown",
		Name:        "trade_journal",
		Title:       "Trade Journal",
		URI:         tradeJournalURI,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		state, err := sessionStateForRead(ctx, store, req, verifier)
		if err != nil {
			return nil, err
		}
		if state == nil || state.AuthMode != session.AuthModeAuthenticated || state.SubAccountID <= 0 {
			return nil, errors.New("trade journal requires an authenticated session; authenticate first then re-read account://trade-journal")
		}
		if tradeReads == nil {
			return nil, sanitizeResourceBackendError("trade journal is temporarily unavailable: REST trade backend is not configured")
		}

		now := time.Now().UTC()
		startMs := now.Add(-tradeJournalLookbk).UnixMilli()
		params := map[string]any{
			"startTime": startMs,
			"endTime":   now.UnixMilli(),
			"limit":     tradeJournalLimit,
		}

		tc := tools.ToolContext{State: state}
		resp, err := tradeReads.GetTrades(ctx, tc, params)
		if err != nil {
			return nil, sanitizeResourceBackendError("trade journal is temporarily unavailable")
		}
		var trades []tradeJournalEntry
		if resp != nil {
			trades = mapTradesToJournalEntries(resp.Trades)
		}

		journal := buildTradeJournal(trades, now, tradeJournalLookbk)

		body := renderTradeJournalDocument(journal, state.SubAccountID)
		return textResourceResult(tradeJournalURI, "text/markdown", body), nil
	})
}

// tradeJournalEntry is the local, decimal-safe representation of a
// single fill that the journal aggregates over. We mirror only the
// fields that the journal uses; anything else stays in the upstream
// JSON if a downstream consumer needs it.
type tradeJournalEntry struct {
	TradedAt    time.Time
	Symbol      string
	Direction   string
	Side        string
	Price       decimal.Decimal
	Quantity    decimal.Decimal
	ClosedPnl   decimal.Decimal
	Fee         decimal.Decimal
	FillType    string
	OrderType   string
	Liquidation bool
}

// mapTradesToJournalEntries converts the upstream synthetix-go trade
// types into journal entries. We accept `any` here so the helper can
// be exercised in unit tests without dragging the full REST payload
// shape into the test fixtures — anything with the documented JSON
// keys round-trips cleanly through json.Marshal/Unmarshal.
func mapTradesToJournalEntries(trades any) []tradeJournalEntry {
	if trades == nil {
		return nil
	}
	raw, err := json.Marshal(trades)
	if err != nil {
		return nil
	}
	var rows []struct {
		TradedAt               string `json:"tradedAt"`
		Symbol                 string `json:"symbol"`
		Direction              string `json:"direction"`
		Side                   string `json:"side"`
		FilledPrice            string `json:"filledPrice"`
		FilledQuantity         string `json:"filledQuantity"`
		ClosedPnl              string `json:"closedPnl"`
		Fee                    string `json:"fee"`
		FillType               string `json:"fillType"`
		OrderType              string `json:"orderType"`
		TriggeredByLiquidation bool   `json:"triggeredByLiquidation"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}
	out := make([]tradeJournalEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, tradeJournalEntry{
			TradedAt:    parseRESTTime(r.TradedAt),
			Symbol:      strings.ToUpper(strings.TrimSpace(r.Symbol)),
			Direction:   strings.ToUpper(strings.TrimSpace(r.Direction)),
			Side:        strings.ToUpper(strings.TrimSpace(r.Side)),
			Price:       parseDecimal(r.FilledPrice),
			Quantity:    parseDecimal(r.FilledQuantity),
			ClosedPnl:   parseDecimal(r.ClosedPnl),
			Fee:         parseDecimal(r.Fee),
			FillType:    strings.ToUpper(strings.TrimSpace(r.FillType)),
			OrderType:   strings.ToUpper(strings.TrimSpace(r.OrderType)),
			Liquidation: r.TriggeredByLiquidation,
		})
	}
	return out
}

// tradeJournal is the aggregated shape the renderer consumes. Kept
// internal — the resource serves rendered text rather than this
// struct, so we don't need to lock the schema.
type tradeJournal struct {
	WindowStart time.Time
	WindowEnd   time.Time
	Days        []tradeJournalDay
	BySymbol    []tradeJournalSymbol
	Recent      []tradeJournalEntry
	NetPnl      decimal.Decimal
	GrossWins   decimal.Decimal
	GrossLosses decimal.Decimal
	NetFees     decimal.Decimal
	Wins        int
	Losses      int
	Flat        int
	TotalTrades int
	Liquidations int
}

type tradeJournalDay struct {
	Date    time.Time
	NetPnl  decimal.Decimal
	Fee     decimal.Decimal
	Trades  int
	Wins    int
	Losses  int
}

type tradeJournalSymbol struct {
	Symbol string
	NetPnl decimal.Decimal
	Volume decimal.Decimal
	Trades int
	Wins   int
	Losses int
}

func buildTradeJournal(entries []tradeJournalEntry, now time.Time, lookback time.Duration) tradeJournal {
	j := tradeJournal{
		WindowStart: now.Add(-lookback).UTC(),
		WindowEnd:   now.UTC(),
	}
	if len(entries) == 0 {
		return j
	}

	sort.SliceStable(entries, func(i, k int) bool {
		return entries[i].TradedAt.Before(entries[k].TradedAt)
	})

	dayBuckets := map[time.Time]*tradeJournalDay{}
	symBuckets := map[string]*tradeJournalSymbol{}

	for _, e := range entries {
		j.TotalTrades++
		j.NetFees = j.NetFees.Add(e.Fee)

		if e.ClosedPnl.IsPositive() {
			j.Wins++
			j.GrossWins = j.GrossWins.Add(e.ClosedPnl)
		} else if e.ClosedPnl.IsNegative() {
			j.Losses++
			j.GrossLosses = j.GrossLosses.Add(e.ClosedPnl)
		} else {
			j.Flat++
		}
		j.NetPnl = j.NetPnl.Add(e.ClosedPnl)
		if e.Liquidation {
			j.Liquidations++
		}

		day := time.Date(e.TradedAt.Year(), e.TradedAt.Month(), e.TradedAt.Day(), 0, 0, 0, 0, time.UTC)
		bucket, ok := dayBuckets[day]
		if !ok {
			bucket = &tradeJournalDay{Date: day}
			dayBuckets[day] = bucket
		}
		bucket.Trades++
		bucket.NetPnl = bucket.NetPnl.Add(e.ClosedPnl)
		bucket.Fee = bucket.Fee.Add(e.Fee)
		switch {
		case e.ClosedPnl.IsPositive():
			bucket.Wins++
		case e.ClosedPnl.IsNegative():
			bucket.Losses++
		}

		sym, ok := symBuckets[e.Symbol]
		if !ok {
			sym = &tradeJournalSymbol{Symbol: e.Symbol}
			symBuckets[e.Symbol] = sym
		}
		sym.Trades++
		sym.NetPnl = sym.NetPnl.Add(e.ClosedPnl)
		sym.Volume = sym.Volume.Add(e.Price.Mul(e.Quantity).Abs())
		switch {
		case e.ClosedPnl.IsPositive():
			sym.Wins++
		case e.ClosedPnl.IsNegative():
			sym.Losses++
		}
	}

	for _, d := range dayBuckets {
		j.Days = append(j.Days, *d)
	}
	sort.SliceStable(j.Days, func(i, k int) bool { return j.Days[i].Date.Before(j.Days[k].Date) })

	for _, s := range symBuckets {
		j.BySymbol = append(j.BySymbol, *s)
	}
	sort.SliceStable(j.BySymbol, func(i, k int) bool {
		return j.BySymbol[i].NetPnl.Cmp(j.BySymbol[k].NetPnl) > 0
	})

	closed := make([]tradeJournalEntry, 0, len(entries))
	for _, e := range entries {
		if !e.ClosedPnl.IsZero() {
			closed = append(closed, e)
		}
	}
	if len(closed) > 0 {
		recent := make([]tradeJournalEntry, len(closed))
		copy(recent, closed)
		sort.SliceStable(recent, func(i, k int) bool { return recent[i].TradedAt.After(recent[k].TradedAt) })
		if len(recent) > 5 {
			recent = recent[:5]
		}
		j.Recent = recent
	}

	return j
}

// renderTradeJournalDocument renders the journal as markdown with an
// optional ASCII card block at the top. Markdown is used so chat
// clients (Claude Desktop, Cursor) get bold headers and tables, while
// terminal clients (Claude Code) still see the raw text and the card
// block above it.
func renderTradeJournalDocument(j tradeJournal, subAccountID int64) string {
	out := strings.Builder{}

	if cards.Enabled() {
		out.WriteString(renderTradeJournalCard(j, subAccountID))
		out.WriteString("\n\n")
	}

	out.WriteString(fmt.Sprintf("# Trade Journal (subaccount %d)\n\n", subAccountID))
	out.WriteString(fmt.Sprintf("Window: %s → %s (UTC)\n\n",
		j.WindowStart.Format("2006-01-02"),
		j.WindowEnd.Format("2006-01-02 15:04"),
	))
	if j.TotalTrades == 0 {
		out.WriteString("_No fills in window._\n")
		return out.String()
	}

	netPnlF, _ := j.NetPnl.Float64()
	winsLosses := j.Wins + j.Losses
	winRate := 0.0
	if winsLosses > 0 {
		winRate = float64(j.Wins) / float64(winsLosses) * 100
	}
	feeF, _ := j.NetFees.Float64()

	out.WriteString("## Summary\n\n")
	out.WriteString(fmt.Sprintf("- Net PnL: **%s**\n", cards.SignedUSD(netPnlF, 2)))
	out.WriteString(fmt.Sprintf("- Win rate: **%.1f%%** (%d / %d closed)\n", winRate, j.Wins, winsLosses))
	out.WriteString(fmt.Sprintf("- Fills: **%d** (wins %d / losses %d / flat %d)\n", j.TotalTrades, j.Wins, j.Losses, j.Flat))
	out.WriteString(fmt.Sprintf("- Total fees: **%s**\n", cards.USD(feeF, 2)))
	if j.Liquidations > 0 {
		out.WriteString(fmt.Sprintf("- Liquidations: **%d**\n", j.Liquidations))
	}
	out.WriteString("\n")

	out.WriteString("## Daily PnL\n\n")
	out.WriteString("| Date | Net PnL | Trades | W/L |\n")
	out.WriteString("|------|---------|--------|-----|\n")
	for _, d := range j.Days {
		dF, _ := d.NetPnl.Float64()
		out.WriteString(fmt.Sprintf("| %s | %s | %d | %d/%d |\n",
			d.Date.Format("2006-01-02"),
			cards.SignedUSD(dF, 2),
			d.Trades,
			d.Wins, d.Losses,
		))
	}
	out.WriteString("\n")

	out.WriteString("## Per-symbol breakdown\n\n")
	out.WriteString("| Symbol | Net PnL | Trades | W/L | Volume |\n")
	out.WriteString("|--------|---------|--------|-----|--------|\n")
	for _, s := range j.BySymbol {
		nF, _ := s.NetPnl.Float64()
		vF, _ := s.Volume.Float64()
		out.WriteString(fmt.Sprintf("| %s | %s | %d | %d/%d | $%s |\n",
			s.Symbol,
			cards.SignedUSD(nF, 2),
			s.Trades,
			s.Wins, s.Losses,
			compactJournalNumber(vF),
		))
	}
	out.WriteString("\n")

	if len(j.Recent) > 0 {
		out.WriteString("## Recent closed trades\n\n")
		for _, e := range j.Recent {
			pnlF, _ := e.ClosedPnl.Float64()
			priceF, _ := e.Price.Float64()
			qtyF, _ := e.Quantity.Float64()
			arrow := "▲"
			if pnlF < 0 {
				arrow = "▼"
			} else if pnlF == 0 {
				arrow = "◆"
			}
			out.WriteString(fmt.Sprintf("- `%s` %s %s %s — exit %s × %s → %s\n",
				e.TradedAt.UTC().Format("2006-01-02 15:04"),
				arrow,
				strings.ToUpper(e.Side),
				e.Symbol,
				cards.USD(priceF, journalPriceDecimals(priceF)),
				trimDecimal(qtyF),
				cards.SignedUSD(pnlF, 2),
			))
		}
		out.WriteString("\n")
	}

	return out.String()
}

// renderTradeJournalCard is the compact 80-col card that sits at the
// top of the journal. Daily PnL is rendered as a bar chart where each
// day is one cell tall and the magnitude scales to the largest day.
func renderTradeJournalCard(j tradeJournal, subAccountID int64) string {
	netF, _ := j.NetPnl.Float64()
	status := cards.SignedStatus(netF)
	if j.TotalTrades == 0 {
		return cards.Card{
			Status: cards.StatusNeutral,
			Title:  fmt.Sprintf("TRADE JOURNAL  #%d", subAccountID),
			Sections: []cards.Section{
				{Rows: []cards.Row{{Label: "No fills in window.", Value: "", Hint: ""}}},
			},
		}.Render()
	}

	winsLosses := j.Wins + j.Losses
	winRate := 0.0
	if winsLosses > 0 {
		winRate = float64(j.Wins) / float64(winsLosses) * 100
	}
	feeF, _ := j.NetFees.Float64()

	summaryRows := []cards.Row{
		{Label: "Net PnL:", Value: cards.SignedUSD(netF, 2), Hint: fmt.Sprintf("%d fills · %.1f%% win-rate", j.TotalTrades, winRate)},
		{Label: "Wins / losses:", Value: fmt.Sprintf("%d W / %d L", j.Wins, j.Losses), Hint: fmt.Sprintf("flat %d", j.Flat)},
		{Label: "Total fees:", Value: cards.USD(feeF, 2), Hint: ""},
	}
	if j.Liquidations > 0 {
		summaryRows = append(summaryRows, cards.Row{
			Label: "Liquidations:",
			Value: fmt.Sprintf("%d", j.Liquidations),
			Hint:  "review get_trade_history",
		})
	}

	dailyBar := dailyPnlBars(j.Days)
	dailyRows := []cards.Row{
		{Label: "Daily PnL:", Value: dailyBar, Hint: fmt.Sprintf("last %d days", len(j.Days))},
	}

	symbolRows := make([]cards.Row, 0, len(j.BySymbol))
	maxSyms := len(j.BySymbol)
	if maxSyms > 5 {
		maxSyms = 5
	}
	for _, s := range j.BySymbol[:maxSyms] {
		nF, _ := s.NetPnl.Float64()
		arrow := glyphForFloat(nF)
		symbolRows = append(symbolRows, cards.Row{
			Label: arrow + " " + s.Symbol,
			Value: cards.SignedUSD(nF, 2),
			Hint:  fmt.Sprintf("%d trades · %dW/%dL", s.Trades, s.Wins, s.Losses),
		})
	}

	sectionsList := []cards.Section{
		{Rows: summaryRows},
		{Rows: dailyRows},
	}
	if len(symbolRows) > 0 {
		sectionsList = append(sectionsList, cards.Section{Rows: symbolRows})
	}

	return cards.Card{
		Status:   status,
		Title:    fmt.Sprintf("TRADE JOURNAL  #%d  %dd window", subAccountID, int(j.WindowEnd.Sub(j.WindowStart).Hours()/24)),
		Sections: sectionsList,
	}.Render()
}

// dailyPnlBars renders one row of `▲▲▆▂▁ ▼▼▼` glyphs scaled to the
// largest absolute day in the window. Empty days are rendered as a
// dot to keep the time axis even.
func dailyPnlBars(days []tradeJournalDay) string {
	if len(days) == 0 {
		return "—"
	}
	max := 0.0
	for _, d := range days {
		f, _ := d.NetPnl.Abs().Float64()
		if f > max {
			max = f
		}
	}
	if max == 0 {
		return strings.Repeat("·", len(days))
	}
	upGlyphs := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	downGlyphs := []rune{'⡀', '⡄', '⡆', '⡇', '⡏', '⡟', '⡿', '⣿'}
	out := strings.Builder{}
	for _, d := range days {
		f, _ := d.NetPnl.Float64()
		switch {
		case f == 0:
			out.WriteRune('·')
		case f > 0:
			idx := int((f / max) * float64(len(upGlyphs)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(upGlyphs) {
				idx = len(upGlyphs) - 1
			}
			out.WriteRune(upGlyphs[idx])
		default:
			absF := -f
			idx := int((absF / max) * float64(len(downGlyphs)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(downGlyphs) {
				idx = len(downGlyphs) - 1
			}
			out.WriteRune(downGlyphs[idx])
		}
	}
	return out.String()
}

func glyphForFloat(v float64) string {
	switch {
	case v > 0:
		return "▲"
	case v < 0:
		return "▼"
	default:
		return "◆"
	}
}

func parseDecimal(s string) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// parseRESTTime accepts the RFC3339 / RFC3339Nano shape that the
// trade-history REST endpoint uses, and falls back to a numeric
// millisecond epoch (the REST list endpoints use that for createdAt
// in some payloads). Unparseable values become zero, which the
// renderer flags as "—".
func parseRESTTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

func journalPriceDecimals(p float64) int {
	abs := p
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1000:
		return 0
	case abs >= 1:
		return 2
	case abs >= 0.01:
		return 4
	default:
		return 6
	}
}

func trimDecimal(v float64) string {
	s := fmt.Sprintf("%.6f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func compactJournalNumber(v float64) string {
	abs := v
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1_000_000_000:
		return trimZ(fmt.Sprintf("%.2f", v/1_000_000_000)) + "B"
	case abs >= 1_000_000:
		return trimZ(fmt.Sprintf("%.2f", v/1_000_000)) + "M"
	case abs >= 1_000:
		return trimZ(fmt.Sprintf("%.2f", v/1_000)) + "K"
	case abs >= 1:
		return fmt.Sprintf("%.0f", v)
	case abs == 0:
		return "0"
	default:
		return trimZ(fmt.Sprintf("%.4f", v))
	}
}

func trimZ(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}
