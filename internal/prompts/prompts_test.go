package prompts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterPromptsListAndRead(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	Register(server, false)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	result, err := cs.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatalf("list prompts failed: %v", err)
	}
	if len(result.Prompts) != 10 {
		t.Fatalf("expected 10 prompts, got %d", len(result.Prompts))
	}
	promptArgs := map[string]int{}
	for _, prompt := range result.Prompts {
		promptArgs[prompt.Name] = len(prompt.Arguments)
	}
	if promptArgs["quickstart"] != 4 {
		t.Fatalf("expected quickstart to expose 4 arguments, got %d", promptArgs["quickstart"])
	}
	if promptArgs["startup-validation"] != 1 {
		t.Fatalf("expected startup-validation to expose 1 argument, got %d", promptArgs["startup-validation"])
	}
	if promptArgs["position-risk-report"] != 1 {
		t.Fatalf("expected position-risk-report to expose 1 argument, got %d", promptArgs["position-risk-report"])
	}
	if promptArgs["pre-trade-checklist"] != 5 {
		t.Fatalf("expected pre-trade-checklist to expose 5 arguments, got %d", promptArgs["pre-trade-checklist"])
	}
	if promptArgs["protect_session_with_dead_man_switch"] != 1 {
		t.Fatalf("expected protect_session_with_dead_man_switch to expose 1 argument, got %d", promptArgs["protect_session_with_dead_man_switch"])
	}

	prompt, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name:      "market-analysis",
		Arguments: map[string]string{"symbol": "BTC-USDT"},
	})
	if err != nil {
		t.Fatalf("get prompt failed: %v", err)
	}
	if len(prompt.Messages) != 1 {
		t.Fatalf("expected one prompt message, got %d", len(prompt.Messages))
	}
	text, ok := prompt.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", prompt.Messages[0].Content)
	}
	if want := "market://specs/BTC-USDT"; text.Text == "" || !strings.Contains(text.Text, want) {
		t.Fatalf("expected prompt text to contain %q, got %q", want, text.Text)
	}

	checklistPrompt, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name: "pre-trade-checklist",
		Arguments: map[string]string{
			"symbol":       "ETH-USDT",
			"side":         "BUY",
			"quantity":     "2.5",
			"price":        "2300",
			"subAccountId": "77",
		},
	})
	if err != nil {
		t.Fatalf("get pre-trade prompt failed: %v", err)
	}
	checklistText, ok := checklistPrompt.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected checklist text content, got %T", checklistPrompt.Messages[0].Content)
	}
	for _, want := range []string{"BUY", "ETH-USDT", "2.5", "2300", "77", "guardrails", "one trade confirmation"} {
		if !strings.Contains(checklistText.Text, want) {
			t.Fatalf("expected checklist prompt to contain %q, got %q", want, checklistText.Text)
		}
	}

	quickstartPrompt, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name: "quickstart",
		Arguments: map[string]string{
			"subAccountId": "42",
			"symbol":       "BTC-USDT",
			"side":         "SELL",
			"quantity":     "0.01",
		},
	})
	if err != nil {
		t.Fatalf("get quickstart prompt failed: %v", err)
	}
	quickstartText, ok := quickstartPrompt.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected quickstart text content, got %T", quickstartPrompt.Messages[0].Content)
	}
	// Wallet-path quickstart must surface the local-signing tools so the
	// agent renders a concrete script rather than a generic template.
	for _, want := range []string{"SELL", "BTC-USDT", "0.01", "42", "preview_auth_message", "preview_trade_signature", "set_guardrails", "signed_place_order", "at most once"} {
		if !strings.Contains(quickstartText.Text, want) {
			t.Fatalf("expected wallet-path quickstart prompt to contain %q, got %q", want, quickstartText.Text)
		}
	}
	// Wallet-path body must NOT advertise canonical broker submission as the signed path
	// and routing the agent through them when the broker is disabled
	// would surface NOT_FOUND from the tool registry.
	if strings.Contains(quickstartText.Text, "Call place_order") {
		t.Fatalf("expected wallet-path quickstart NOT to call place_order, got %q", quickstartText.Text)
	}
}

// Locks the broker-on rendering path. Same prompt name + arguments,
// but the body must route exclusively through canonical broker tools and must not
// instruct the agent to call authenticate / preview_* / signed_* tools
// (which are duplicated work when the self-hosted broker is enabled and tempt the
// agent to dump signature payloads to the user).
func TestQuickstartPromptBrokerEnabled(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	Register(server, true)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	prompt, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name: "quickstart",
		Arguments: map[string]string{
			"symbol":   "BTC-USDT",
			"side":     "BUY",
			"quantity": "0.01",
		},
	})
	if err != nil {
		t.Fatalf("get broker quickstart prompt failed: %v", err)
	}
	text, ok := prompt.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected quickstart text content, got %T", prompt.Messages[0].Content)
	}
	for _, want := range []string{"BUY", "BTC-USDT", "0.01", "place_order", "close_position", "cancel_order", "guardrails", "at most once"} {
		if !strings.Contains(text.Text, want) {
			t.Fatalf("expected broker quickstart prompt to contain %q, got %q", want, text.Text)
		}
	}
	for _, banned := range []string{"Call preview_auth_message", "Call preview_trade_signature", "Call authenticate", "Call signed_place_order", "Call signed_cancel_order", "Call signed_modify_order", "Call signed_close_position"} {
		if strings.Contains(text.Text, banned) {
			t.Fatalf("expected broker quickstart prompt NOT to contain %q, got %q", banned, text.Text)
		}
	}
}
