package cards

import (
	"strings"
	"testing"
	"time"
)

// Confirms emoji, ANSI, and mixed strings have expected display widths.
func TestDisplayWidthCoversEmojiAndANSI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"ascii", "hello", 5},
		{"green_dot_is_two_cells", "🟢", 2},
		{"red_dot_is_two_cells", "🔴", 2},
		{"up_arrow_is_one_cell", "▲", 1},
		{"dollar_sign_is_one_cell", "$", 1},
		{"ansi_green_is_zero_cells", "\x1b[32m+$617\x1b[0m", 5},
		{"mixed_title", "🟢 CLOSED LONG ▲ BTC-USDT", 25},
		{"em_dash_is_one_cell", "—", 1},
		{"empty_is_zero", "", 0},
	}
	for _, tc := range cases {
		if got := displayWidth(tc.in); got != tc.want {
			t.Errorf("%s: displayWidth(%q) = %d, want %d", tc.name, tc.in, got, tc.want)
		}
	}
}

// Confirms padding uses display cells rather than bytes.
func TestPadRightFillsExactCells(t *testing.T) {
	padded := padRight("🟢 win", 10)
	if got := displayWidth(padded); got != 10 {
		t.Fatalf("padRight display width = %d, want 10; raw=%q", got, padded)
	}
}

// Confirms signed values use the expected direction glyphs.
func TestSignedWithArrowFlipsGlyphOnSign(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{617, "▲ +617"},
		{-540, "▼ -540"},
		{0, "◆ 0"},
	}
	for _, tc := range cases {
		if got := SignedWithArrow(tc.in, 0); got != tc.want {
			t.Errorf("SignedWithArrow(%v, 0) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Locks the money format used across cards.
func TestSignedUSDFormatsWithThousandsSeparator(t *testing.T) {
	cases := []struct {
		val      float64
		decimals int
		want     string
	}{
		{76300, 0, "+$76,300"},
		{-540, 0, "-$540"},
		{0.589, 3, "+$0.589"},
		{0, 2, "$0.00"},
		{1234567.89, 2, "+$1,234,567.89"},
	}
	for _, tc := range cases {
		if got := SignedUSD(tc.val, tc.decimals); got != tc.want {
			t.Errorf("SignedUSD(%v, %d) = %q, want %q", tc.val, tc.decimals, got, tc.want)
		}
	}
}

// Locks held-for formatting across duration buckets.
func TestDurationHumanizesAcrossBuckets(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Second, "~45s"},
		{8 * time.Minute, "~8min"},
		{2*time.Hour + 13*time.Minute, "~2h 13min"},
		{15*time.Hour + 47*time.Minute, "~15h 47min"},
		{3 * time.Hour, "~3h"},
		{26*time.Hour + 30*time.Minute, "~1d 2h"},
		{48 * time.Hour, "~2d"},
	}
	for _, tc := range cases {
		if got := Duration(tc.d); got != tc.want {
			t.Errorf("Duration(%s) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

// Covers the sign-to-status mapping used for header glyphs.
func TestSignedStatusPicksCorrectEnum(t *testing.T) {
	cases := []struct {
		in   float64
		want Status
	}{
		{0.62, StatusPositive},
		{-0.54, StatusNegative},
		{0, StatusFlat},
		{0.00001, StatusFlat},
	}
	for _, tc := range cases {
		if got := SignedStatus(tc.in); got != tc.want {
			t.Errorf("SignedStatus(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// Confirms the full winning-PnL reference card stays aligned.
func TestCardRenderWinningPnLShape(t *testing.T) {
	card := Card{
		Status: StatusPositive,
		Title:  "CLOSED LONG ▲ BTC-USDT",
		Sections: []Section{
			{Rows: []Row{
				{Label: "Held for:", Value: "~15h 47min", Hint: "Opened: 2026-04-30 06:24 UTC"},
			}},
			{Rows: []Row{
				{Label: "Side:", Value: "LONG ▲"},
				{Label: "Quantity:", Value: "0.001 BTC", Hint: "Notional: $76.30"},
				{Label: "Entry:", Value: "$75,683"},
				{Label: "Exit:", Value: "$76,300", Hint: "▲ +$617   (+0.82%)"},
			}},
			{Rows: []Row{
				{Label: "Realized PnL:", Value: "+$0.62", Hint: "🟢 +0.82%"},
				{Label: "Fees paid:", Value: "$0.031", Hint: "(0.04% taker)"},
				{Label: "Net PnL:", Value: "+$0.589"},
			}},
		},
		Footnote: "Funding paid: -$0.066  (~15h at ~51% APY — longs paid shorts)",
	}

	got := card.Render()

	assertEveryLineIsCardWidth(t, got)

	// Header must show the positive status glyph only.
	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(firstLine, "🟢") {
		t.Errorf("winning card header missing 🟢; got %q", firstLine)
	}
	if strings.Contains(firstLine, "🔴") {
		t.Errorf("winning card header unexpectedly contains 🔴; got %q", firstLine)
	}
}

// Confirms losing-trade cards reverse the status treatment.
func TestCardRenderLosingPnLShape(t *testing.T) {
	card := Card{
		Status: StatusNegative,
		Title:  "CLOSED LONG ▼ BTC-USDT",
		Sections: []Section{
			{Rows: []Row{
				{Label: "Held for:", Value: "~2h 13min", Hint: "Opened: 2026-05-01 18:45 UTC"},
			}},
			{Rows: []Row{
				{Label: "Entry:", Value: "$76,420"},
				{Label: "Exit:", Value: "$75,880", Hint: "▼ -$540   (-0.71%)"},
			}},
			{Rows: []Row{
				{Label: "Realized PnL:", Value: "-$0.54", Hint: "🔴 -0.71%"},
				{Label: "Net PnL:", Value: "-$0.571", Hint: "(after $0.031 fees)"},
			}},
		},
	}

	got := card.Render()
	assertEveryLineIsCardWidth(t, got)

	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(firstLine, "🔴") {
		t.Errorf("losing card header missing 🔴; got %q", firstLine)
	}
	if strings.Contains(firstLine, "🟢") {
		t.Errorf("losing card header unexpectedly contains 🟢; got %q", firstLine)
	}
}

// Pins the break-even glyph for zero-PnL cards.
func TestCardRenderFlatStatus(t *testing.T) {
	card := Card{Status: StatusFlat, Title: "CLOSED LONG ◆ BTC-USDT"}
	got := card.Render()
	if !strings.Contains(got, "⚪") {
		t.Errorf("flat card missing ⚪; got:\n%s", got)
	}
}

// Pins warning and critical glyphs for high-risk outcomes.
func TestCardRenderWarningAndCriticalStatuses(t *testing.T) {
	warn := Card{Status: StatusWarning, Title: "GUARDRAIL TRIPPED"}.Render()
	if !strings.Contains(warn, "🟡") {
		t.Errorf("warning card missing 🟡; got:\n%s", warn)
	}
	crit := Card{Status: StatusCritical, Title: "LIQUIDATED"}.Render()
	if !strings.Contains(crit, "🔥") {
		t.Errorf("critical card missing 🔥; got:\n%s", crit)
	}
}

// Confirms empty cards render without spurious whitespace.
func TestCardRenderEmptyCardYieldsEmptyString(t *testing.T) {
	if got := (Card{}).Render(); got != "" {
		t.Fatalf("empty card should render as \"\"; got %q", got)
	}
}

// Asserts every rendered line uses the fixed card width.
func assertEveryLineIsCardWidth(t *testing.T, rendered string) {
	t.Helper()
	trimmed := strings.TrimRight(rendered, "\n")
	for i, line := range strings.Split(trimmed, "\n") {
		if got := displayWidth(line); got != CardWidth {
			t.Errorf("line %d width = %d, want %d; line=%q", i, got, CardWidth, line)
		}
	}
}
