package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

// classifyPlaceOrderPhase is the routing layer for the new
// place_order phase + followUp fields. The original
// transcript hit the UNKNOWN-status case so we cover it explicitly to
// stop a regression that re-introduces the "is it live? do I retry?"
// confusion.
func TestClassifyPlaceOrderPhase(t *testing.T) {
	cases := []struct {
		name      string
		status    string
		isSuccess bool
		errorCode string
		wantPhase string
		wantHint  string
	}{
		{
			name:      "accepted_filled",
			status:    "FILLED",
			isSuccess: true,
			wantPhase: OrderPhaseAccepted,
			wantHint:  "get_open_orders",
		},
		{
			name:      "accepted_partial",
			status:    "PARTIALLY_FILLED",
			isSuccess: true,
			wantPhase: OrderPhaseAccepted,
		},
		{
			name:      "accepted_pending",
			status:    "PENDING",
			isSuccess: true,
			wantPhase: OrderPhaseAccepted,
		},
		{
			name:      "rejected_explicit_status",
			status:    "REJECTED",
			isSuccess: false,
			wantPhase: OrderPhaseRejected,
			wantHint:  "errorCode",
		},
		{
			name:      "rejected_via_error_code_with_unknown_status",
			status:    "UNKNOWN",
			isSuccess: true,
			errorCode: "INSUFFICIENT_MARGIN",
			wantPhase: OrderPhaseRejected,
		},
		{
			name:      "pending_confirmation_unknown_status",
			status:    "UNKNOWN",
			isSuccess: true,
			wantPhase: OrderPhasePendingConfirmation,
			wantHint:  "Do NOT retry",
		},
		{
			name:      "pending_confirmation_empty_status",
			status:    "",
			isSuccess: true,
			wantPhase: OrderPhasePendingConfirmation,
		},
		{
			name:      "cancelled_with_success_is_accepted",
			status:    "CANCELLED",
			isSuccess: true,
			wantPhase: OrderPhaseAccepted,
		},
		{
			name:      "cancelled_without_success_is_rejected",
			status:    "CANCELLED",
			isSuccess: false,
			wantPhase: OrderPhaseRejected,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			phase, followUp := classifyPlaceOrderPhase(tc.status, tc.isSuccess, tc.errorCode)
			if phase != tc.wantPhase {
				t.Fatalf("phase = %q, want %q", phase, tc.wantPhase)
			}
			if len(followUp) == 0 {
				t.Fatalf("expected at least one followUp hint, got none")
			}
			if tc.wantHint != "" && !containsSubstring(followUp, tc.wantHint) {
				t.Fatalf("expected followUp to mention %q, got %v", tc.wantHint, followUp)
			}
		})
	}
}

// nextStepsForAuthenticatedSession is what makes the authenticate tool
// self-explanatory. The original transcript showed the agent calling
// place_order immediately after authenticate and getting bounced by
// the read_only fallback; this helper has to surface the "you forgot
// set_guardrails" hint so a fresh session never silently rejects
// trading.
func TestNextStepsForAuthenticatedSession(t *testing.T) {
	t.Run("no_state_includes_set_guardrails_hint", func(t *testing.T) {
		steps := nextStepsForAuthenticatedSession(nil)
		if !containsSubstring(steps, "set_guardrails") {
			t.Fatalf("expected nil-state next steps to include set_guardrails hint, got %v", steps)
		}
	})

	t.Run("nil_guardrails_includes_set_guardrails_hint", func(t *testing.T) {
		state := &session.State{AuthMode: session.AuthModeAuthenticated, SubAccountID: 1, WalletAddress: "0x"}
		steps := nextStepsForAuthenticatedSession(state)
		if !containsSubstring(steps, "set_guardrails") {
			t.Fatalf("expected missing-guardrails next steps to include set_guardrails hint, got %v", steps)
		}
		if !containsSubstring(steps, "accept or edit") {
			t.Fatalf("expected missing-guardrails next steps to ask for user accept/edit, got %v", steps)
		}
	})

	t.Run("read_only_includes_upgrade_hint", func(t *testing.T) {
		state := &session.State{
			AuthMode:        session.AuthModeAuthenticated,
			SubAccountID:    1,
			WalletAddress:   "0x",
			AgentGuardrails: &guardrails.Config{Preset: guardrails.PresetReadOnly},
		}
		steps := nextStepsForAuthenticatedSession(state)
		if !containsSubstring(steps, "read_only") {
			t.Fatalf("expected read_only next steps to mention read_only, got %v", steps)
		}
	})

	t.Run("standard_preset_recommends_broker_or_preview_path", func(t *testing.T) {
		state := &session.State{
			AuthMode:      session.AuthModeAuthenticated,
			SubAccountID:  1,
			WalletAddress: "0x",
			AgentGuardrails: &guardrails.Config{
				Preset:              guardrails.PresetStandard,
				AllowedSymbols:      []string{"BTC-USDT"},
				AllowedOrderTypes:   []string{"LIMIT"},
				MaxOrderQuantity:    "1",
				MaxPositionQuantity: "10",
			},
		}
		steps := nextStepsForAuthenticatedSession(state)
		// We deliberately keep the wording loose — assert the agent is
		// pointed at SOMETHING actionable rather than a specific verb.
		if !containsSubstring(steps, "signed_place_order") {
			t.Fatalf("expected standard-preset next steps to mention an order tool, got %v", steps)
		}
		if !containsSubstring(steps, "accept or edit") {
			t.Fatalf("expected standard-preset next steps to ask for user accept/edit, got %v", steps)
		}
	})
}

func containsSubstring(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}

// contextCapabilitiesFromFlags is the single point that decides whether
// get_context tells the agent "use broker tools" or "sign locally". The
// regressions we want to catch are subtle: forgetting to populate
// brokerTools when the broker is on, or forgetting the "never paste
// signatures" guidance when it is off.
func TestContextCapabilitiesFromFlagsBrokerEnabled(t *testing.T) {
	caps := contextCapabilitiesFromFlags(true, "standard")
	if !caps.AgentBroker.Enabled {
		t.Fatalf("expected AgentBroker.Enabled = true when broker flag set")
	}
	if caps.SigningPolicy != "broker" {
		t.Fatalf("expected SigningPolicy=broker, got %q", caps.SigningPolicy)
	}
	if len(caps.AgentBroker.BrokerTools) == 0 {
		t.Fatalf("expected BrokerTools to be populated when broker enabled")
	}
	wantBrokerTools := map[string]bool{
		"place_order":       true,
		"close_position":    true,
		"cancel_order":      true,
		"cancel_all_orders": true,
	}
	for _, tool := range caps.AgentBroker.BrokerTools {
		if !wantBrokerTools[tool] {
			t.Fatalf("unexpected broker tool advertised: %q", tool)
		}
		delete(wantBrokerTools, tool)
	}
	if len(wantBrokerTools) != 0 {
		t.Fatalf("expected all four broker tools advertised, missing %v", wantBrokerTools)
	}
	if caps.AgentBroker.DefaultPreset != "standard" {
		t.Fatalf("expected DefaultPreset to flow through, got %q", caps.AgentBroker.DefaultPreset)
	}
	if !containsSubstring(caps.RecommendedFlow, "place_order") {
		t.Fatalf("expected recommendedFlow to point at place_order, got %v", caps.RecommendedFlow)
	}
	if !containsSubstring(caps.RecommendedFlow, "accept or edit") {
		t.Fatalf("expected recommendedFlow to include guardrail accept/edit step, got %v", caps.RecommendedFlow)
	}
}

func TestContextCapabilitiesFromFlagsBrokerDisabled(t *testing.T) {
	caps := contextCapabilitiesFromFlags(false, "")
	if caps.AgentBroker.Enabled {
		t.Fatalf("expected AgentBroker.Enabled = false when broker flag clear")
	}
	if caps.SigningPolicy != "client" {
		t.Fatalf("expected SigningPolicy=client, got %q", caps.SigningPolicy)
	}
	if len(caps.AgentBroker.BrokerTools) != 0 {
		t.Fatalf("expected no brokerTools when broker disabled, got %v", caps.AgentBroker.BrokerTools)
	}
	// The "never paste signatures into chat" rule is the actual
	// behavioural change we are protecting against. Without it the
	// agent in the original transcript happily dumped typedData into
	// chat for the user to sign.
	if !strings.Contains(caps.AgentBroker.Note, "Never ask") &&
		!strings.Contains(caps.AgentBroker.Note, "never ask") {
		t.Fatalf("expected disabled-broker note to repeat the no-paste rule, got %q", caps.AgentBroker.Note)
	}
	joined := strings.Join(caps.RecommendedFlow, " | ")
	for _, mustMention := range []string{"preview_auth_message", "authenticate", "accept or edit", "set_guardrails", "signed_place_order"} {
		if !strings.Contains(joined, mustMention) {
			t.Fatalf("expected client-signing recommendedFlow to mention %q, got %q", mustMention, joined)
		}
	}
	if !strings.Contains(joined, "do NOT prompt the user") &&
		!strings.Contains(joined, "refuse the trade") {
		t.Fatalf("expected client-signing recommendedFlow to instruct refusal when no local key, got %q", joined)
	}
}

// buildServerInfoAgentBroker mirrors the same broker/no-broker decision
// inside get_server_info. It is a separate helper rather than a shared
// one because the surfaces have different audiences (capabilities is a
// playbook; serverInfo is a one-line capability advertisement) — covering
// both prevents them drifting silently.
func TestBuildServerInfoAgentBrokerEnabled(t *testing.T) {
	deps := &ToolDeps{Cfg: &config.Config{
		AgentBroker: config.AgentBrokerConfig{Enabled: true, DefaultPreset: "standard"},
	}}
	info := buildServerInfoAgentBroker(deps)
	if !info.Enabled {
		t.Fatalf("expected Enabled=true")
	}
	if info.DefaultPreset != "standard" {
		t.Fatalf("expected DefaultPreset to flow through, got %q", info.DefaultPreset)
	}
	if len(info.BrokerTools) != 4 {
		t.Fatalf("expected four brokerTools, got %v", info.BrokerTools)
	}
	if !strings.Contains(info.Note, "place_order") {
		t.Fatalf("expected note to call out place_order, got %q", info.Note)
	}
}

func TestBuildServerInfoAgentBrokerDisabled(t *testing.T) {
	deps := &ToolDeps{Cfg: &config.Config{}}
	info := buildServerInfoAgentBroker(deps)
	if info.Enabled {
		t.Fatalf("expected Enabled=false when broker disabled")
	}
	if len(info.BrokerTools) != 0 {
		t.Fatalf("expected empty brokerTools when disabled, got %v", info.BrokerTools)
	}
	if !strings.Contains(info.Note, "never ask") && !strings.Contains(info.Note, "Never ask") {
		t.Fatalf("expected disabled-broker note to include no-paste rule, got %q", info.Note)
	}
}

// Minimal BrokerStatusProvider for tests; tools/ stays off the
// agentbroker import (see ToolDeps.BrokerStatus).
type fakeBrokerStatus struct {
	snap BrokerStatusSnapshot
}

func (f fakeBrokerStatus) Status() BrokerStatusSnapshot { return f.snap }

// Locks the "operator can read off scope and expiry from
// get_server_info" guarantee. A regression here would only surface
// as a degraded experience after the broker had been rotated.
func TestBuildServerInfoAgentBrokerDelegatedSurfacesScope(t *testing.T) {
	expiry := time.Now().Add(72 * time.Hour).Unix()
	deps := &ToolDeps{
		Cfg: &config.Config{AgentBroker: config.AgentBrokerConfig{
			Enabled: true, DefaultPreset: "standard",
		}},
		BrokerStatus: fakeBrokerStatus{snap: BrokerStatusSnapshot{
			ChainID:          1,
			DefaultPreset:    "standard",
			DelegationID:     7,
			ExpiresAtUnix:    expiry,
			OwnerAddress:     "0xfeedface",
			Permissions:      []string{"trading"},
			SubAccountID:     1234,
			SubaccountSource: "delegated",
			WalletAddress:    "0xdeadbeef",
		}},
	}
	info := buildServerInfoAgentBroker(deps)
	if info.SubAccountID != 1234 {
		t.Fatalf("expected subaccount surfaced, got %d", info.SubAccountID)
	}
	if info.OwnerAddress != "0xfeedface" {
		t.Fatalf("expected owner surfaced, got %q", info.OwnerAddress)
	}
	if info.WalletAddress != "0xdeadbeef" {
		t.Fatalf("expected broker wallet surfaced, got %q", info.WalletAddress)
	}
	if info.SubaccountSource != "delegated" {
		t.Fatalf("expected delegated source, got %q", info.SubaccountSource)
	}
	if info.ExpiresAtUnix != expiry {
		t.Fatalf("expected expiry %d, got %d", expiry, info.ExpiresAtUnix)
	}
	if !strings.Contains(info.Note, "Delegation expires") {
		t.Fatalf("expected note to surface expiry warning, got %q", info.Note)
	}
}

// Owned-key posture must render an unmistakeable warning — silently
// passing it through would let the broker withdraw collateral and
// manage delegations alongside trading.
func TestBuildServerInfoAgentBrokerOwnedAddsWarning(t *testing.T) {
	deps := &ToolDeps{
		Cfg: &config.Config{AgentBroker: config.AgentBrokerConfig{
			Enabled: true, DefaultPreset: "standard",
		}},
		BrokerStatus: fakeBrokerStatus{snap: BrokerStatusSnapshot{
			SubAccountID:     42,
			SubaccountSource: "owned",
			Permissions:      []string{"*"},
			WalletAddress:    "0xdeadbeef",
		}},
	}
	info := buildServerInfoAgentBroker(deps)
	if !strings.Contains(info.Note, "WARNING") {
		t.Fatalf("expected owned-mode warning in note, got %q", info.Note)
	}
	if !strings.Contains(info.Note, "onboard-agent-key") {
		t.Fatalf("expected note to point at onboarding script, got %q", info.Note)
	}
}
