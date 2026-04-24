package config

import (
	"strings"
	"testing"
	"time"

	snx_lib_api_ratelimiting "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/rate_limiting"
)

func TestApplyDefaultsSetsEveryZeroValue(t *testing.T) {
	cfg := &Config{}

	applyDefaults(cfg)

	if cfg.AuthCacheMaxEntries != DefaultAuthCacheMaxEntries {
		t.Fatalf("expected auth cache default %d, got %d", DefaultAuthCacheMaxEntries, cfg.AuthCacheMaxEntries)
	}
	if cfg.Environment != DefaultEnvironment {
		t.Fatalf("expected environment default %q, got %q", DefaultEnvironment, cfg.Environment)
	}
	if cfg.APIHTTPTimeout != DefaultAPIHTTPTimeout {
		t.Fatalf("expected API HTTP timeout default %s, got %s", DefaultAPIHTTPTimeout, cfg.APIHTTPTimeout)
	}
	if cfg.APIMarketCacheTTL != DefaultAPIMarketCacheTTL {
		t.Fatalf("expected API market cache TTL default %s, got %s", DefaultAPIMarketCacheTTL, cfg.APIMarketCacheTTL)
	}
	if cfg.HTTPIdleTimeout != DefaultHTTPIdleTimeout {
		t.Fatalf("expected HTTP idle timeout default %s, got %s", DefaultHTTPIdleTimeout, cfg.HTTPIdleTimeout)
	}
	if cfg.HTTPReadHeaderTimeout != DefaultHTTPReadHeaderTimeout {
		t.Fatalf("expected read header timeout default %s, got %s", DefaultHTTPReadHeaderTimeout, cfg.HTTPReadHeaderTimeout)
	}
	if cfg.MaxRequestBodyBytes != DefaultMaxRequestBodyBytes {
		t.Fatalf("expected request body default %d, got %d", DefaultMaxRequestBodyBytes, cfg.MaxRequestBodyBytes)
	}
	if cfg.MaxSubscriptionsPerSession != DefaultMaxSubscriptions {
		t.Fatalf("expected max subscriptions default %d, got %d", DefaultMaxSubscriptions, cfg.MaxSubscriptionsPerSession)
	}
	if cfg.ServerName != DefaultServerName {
		t.Fatalf("expected server name default %q, got %q", DefaultServerName, cfg.ServerName)
	}
	if cfg.ServerVersion != DefaultServerVersion {
		t.Fatalf("expected server version default %q, got %q", DefaultServerVersion, cfg.ServerVersion)
	}
	if cfg.SessionTTL != DefaultSessionTTL {
		t.Fatalf("expected session TTL default %s, got %s", DefaultSessionTTL, cfg.SessionTTL)
	}
	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Fatalf("expected shutdown timeout default %s, got %s", DefaultShutdownTimeout, cfg.ShutdownTimeout)
	}
}

func TestApplyDefaultsPreservesExplicitValues(t *testing.T) {
	cfg := &Config{
		AuthCacheMaxEntries:        11,
		Environment:                "production",
		APIHTTPTimeout:             7 * time.Second,
		HTTPIdleTimeout:            8 * time.Second,
		HTTPReadHeaderTimeout:      9 * time.Second,
		MaxRequestBodyBytes:        123,
		MaxSubscriptionsPerSession: 14,
		ServerName:                 "custom-mcp",
		ServerVersion:              "2.3.4",
		SessionTTL:                 17 * time.Second,
		ShutdownTimeout:            18 * time.Second,
	}

	applyDefaults(cfg)

	if cfg.AuthCacheMaxEntries != 11 {
		t.Fatalf("expected explicit auth cache entries to be preserved, got %d", cfg.AuthCacheMaxEntries)
	}
	if cfg.Environment != "production" {
		t.Fatalf("expected explicit environment to be preserved, got %q", cfg.Environment)
	}
	if cfg.APIHTTPTimeout != 7*time.Second {
		t.Fatalf("expected explicit API HTTP timeout to be preserved, got %s", cfg.APIHTTPTimeout)
	}
	if cfg.HTTPIdleTimeout != 8*time.Second {
		t.Fatalf("expected explicit HTTP idle timeout to be preserved, got %s", cfg.HTTPIdleTimeout)
	}
	if cfg.HTTPReadHeaderTimeout != 9*time.Second {
		t.Fatalf("expected explicit read header timeout to be preserved, got %s", cfg.HTTPReadHeaderTimeout)
	}
	if cfg.MaxRequestBodyBytes != 123 {
		t.Fatalf("expected explicit body size to be preserved, got %d", cfg.MaxRequestBodyBytes)
	}
	if cfg.MaxSubscriptionsPerSession != 14 {
		t.Fatalf("expected explicit max subscriptions to be preserved, got %d", cfg.MaxSubscriptionsPerSession)
	}
	if cfg.ServerName != "custom-mcp" {
		t.Fatalf("expected explicit server name to be preserved, got %q", cfg.ServerName)
	}
	if cfg.ServerVersion != "2.3.4" {
		t.Fatalf("expected explicit server version to be preserved, got %q", cfg.ServerVersion)
	}
	if cfg.SessionTTL != 17*time.Second {
		t.Fatalf("expected explicit session TTL to be preserved, got %s", cfg.SessionTTL)
	}
	if cfg.ShutdownTimeout != 18*time.Second {
		t.Fatalf("expected explicit shutdown timeout to be preserved, got %s", cfg.ShutdownTimeout)
	}
}

func setRequiredLoadEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SNXMCP_LOG_LEVEL", "debug")
	t.Setenv("SNXMCP_LOG_OUTPUT_JSON", "true")
	t.Setenv("SNXMCP_SERVER_ADDRESS", ":8090")
	t.Setenv("SNXMCP_API_BASE_URL", "https://api.synthetix.io")
	t.Setenv("SNXMCP_EIP712_DOMAIN_NAME", "Synthetix")
	t.Setenv("SNXMCP_EIP712_DOMAIN_VERSION", "1")
	t.Setenv("SNXMCP_EIP712_CHAIN_ID", "10")
	t.Setenv("SNXMCP_METRICS_PORT", "9100")
	t.Setenv("SNXMCP_DEPLOYMENT_MODE", "development")
}

func TestLoadAppliesDefaultsAndReturnsConfig(t *testing.T) {
	setRequiredLoadEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	if cfg.Environment != DefaultEnvironment {
		t.Fatalf("expected default environment %q, got %q", DefaultEnvironment, cfg.Environment)
	}
	if cfg.ServerName != DefaultServerName {
		t.Fatalf("expected default server name %q, got %q", DefaultServerName, cfg.ServerName)
	}
	if cfg.ServerVersion != DefaultServerVersion {
		t.Fatalf("expected default server version %q, got %q", DefaultServerVersion, cfg.ServerVersion)
	}
	if cfg.MaxSubscriptionsPerSession != DefaultMaxSubscriptions {
		t.Fatalf("expected default max subscriptions %d, got %d", DefaultMaxSubscriptions, cfg.MaxSubscriptionsPerSession)
	}
	if cfg.MaxRequestBodyBytes != DefaultMaxRequestBodyBytes {
		t.Fatalf("expected default max body bytes %d, got %d", DefaultMaxRequestBodyBytes, cfg.MaxRequestBodyBytes)
	}
	if cfg.APIHTTPTimeout != DefaultAPIHTTPTimeout {
		t.Fatalf("expected default API HTTP timeout %s, got %s", DefaultAPIHTTPTimeout, cfg.APIHTTPTimeout)
	}
	if cfg.SessionTTL != DefaultSessionTTL {
		t.Fatalf("expected default session TTL %s, got %s", DefaultSessionTTL, cfg.SessionTTL)
	}
	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Fatalf("expected default shutdown timeout %s, got %s", DefaultShutdownTimeout, cfg.ShutdownTimeout)
	}
	if cfg.Metrics == nil || cfg.Metrics.Port != 9100 {
		t.Fatalf("expected metrics config with port 9100, got %#v", cfg.Metrics)
	}
	if cfg.APIBaseURL != "https://api.synthetix.io" {
		t.Fatalf("expected api base url to be preserved, got %q", cfg.APIBaseURL)
	}
}

func TestLoadReturnsAggregatedValidationErrors(t *testing.T) {
	t.Setenv("SNXMCP_METRICS_PORT", "9100")

	_, err := Load()
	if err == nil {
		t.Fatal("expected config validation to fail")
	}

	message := err.Error()
	if !strings.Contains(message, "log_level is required") {
		t.Fatalf("expected missing log level validation, got %s", message)
	}
	if !strings.Contains(message, "server_address is required") {
		t.Fatalf("expected missing server address validation, got %s", message)
	}
	if !strings.Contains(message, "api_base_url is required") {
		t.Fatalf("expected missing api base url validation, got %s", message)
	}
	if !strings.Contains(message, "eip712_chain_id must be greater than 0") {
		t.Fatalf("expected missing chain id validation, got %s", message)
	}
}

func TestDerivedTokensPerSecondComputesCorrectly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		limit    int64
		windowMs int64
		want     int
	}{
		{"standard 20 per 1s window", 20, 1000, 20},
		{"50 per 5s window", 50, 5000, 10},
		{"100 per 10s window", 100, 10000, 10},
		{"zero limit", 0, 1000, 0},
		{"negative limit", -5, 1000, 0},
		{"zero window", 20, 0, 0},
		{"negative window", 20, -1000, 0},
		{"sub-second window 500ms", 10, 500, 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DerivedTokensPerSecond(tc.limit, tc.windowMs)
			if got != tc.want {
				t.Fatalf("DerivedTokensPerSecond(%d, %d) = %d, want %d", tc.limit, tc.windowMs, got, tc.want)
			}
		})
	}
}

func TestLoadDerivesDerivedRPSValues(t *testing.T) {
	setRequiredLoadEnv(t)
	t.Setenv("SNXMCP_RATELIMITING_IP_RATE_LIMIT", "50")
	t.Setenv("SNXMCP_RATELIMITING_WINDOW_MS", "5000")
	t.Setenv("SNXMCP_RATELIMITING_GENERAL_RATE_LIMIT", "200")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}

	expectedPublicRPS := DerivedTokensPerSecond(50, 5000)
	if cfg.PublicRPSPerIP != expectedPublicRPS {
		t.Fatalf("expected PublicRPSPerIP = %d, got %d", expectedPublicRPS, cfg.PublicRPSPerIP)
	}

	expectedAuthRPS := DerivedTokensPerSecond(200, 5000)
	if cfg.AuthRPSPerSubAccount != expectedAuthRPS {
		t.Fatalf("expected AuthRPSPerSubAccount = %d, got %d", expectedAuthRPS, cfg.AuthRPSPerSubAccount)
	}
}

// Small rate limits relative to the window truncate to zero through the
// integer division in DerivedTokensPerSecond. Load must reject the
// resulting config so the values surfaced via system://status and
// get_rate_limits are never silently poisoned.
func TestLoadRejectsZeroDerivedRPS(t *testing.T) {
	setRequiredLoadEnv(t)
	t.Setenv("SNXMCP_RATELIMITING_IP_RATE_LIMIT", "5")
	t.Setenv("SNXMCP_RATELIMITING_WINDOW_MS", "10000")
	t.Setenv("SNXMCP_RATELIMITING_GENERAL_RATE_LIMIT", "5")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected Load to reject zero-truncated derived RPS values")
	}
	msg := err.Error()
	for _, want := range []string{
		"derived public_rps_per_ip must be greater than 0",
		"ratelimiting_ip_rate_limit=5",
		"ratelimiting_window_ms=10000",
		"derived auth_rps_per_subaccount must be greater than 0",
		"ratelimiting_general_rate_limit=5",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected validation error to mention %q, got: %s", want, msg)
		}
	}
}

func TestDefaultHandlerTokenCostsCoverRateLimitedOperations(t *testing.T) {
	t.Parallel()

	expectedOperations := []string{
		"authenticate",
		"cancel_all_orders",
		"cancel_order",
		"close_position",
		"get_account_summary",
		"get_candles",
		"get_context",
		"get_funding_payments",
		"get_funding_rate",
		"get_market_summary",
		"get_open_orders",
		"get_order_history",
		"get_orderbook",
		"get_performance_history",
		"get_positions",
		"get_prompt",
		"get_rate_limits",
		"get_recent_trades",
		"get_server_info",
		"get_session",
		"get_trade_history",
		"initialize",
		"list_markets",
		"list_prompts",
		"list_resource_templates",
		"list_resources",
		"list_tools",
		"lookup_subaccount",
		"mcp_ping",
		"modify_order",
		"ping",
		"place_order",
		"preview_auth_message",
		"preview_order",
		"preview_trade_signature",
		"quick_cancel_all_orders",
		"quick_cancel_order",
		"quick_close_position",
		"quick_place_order",
		"read_agent_guide",
		"read_fee_schedule",
		"read_market_specs",
		"read_risk_limits",
		"read_runbooks",
		"read_server_info",
		"read_status",
		"restore_session",
		"set_guardrails",
		"subscribe",
		"unsubscribe",
	}

	for _, operationName := range expectedOperations {
		action := snx_lib_api_ratelimiting.RequestAction(operationName)
		if _, ok := defaultHandlerTokenCosts()[action]; !ok {
			t.Fatalf("expected default handler token cost for %q", operationName)
		}
		if _, ok := defaultIPHandlerTokenCosts()[action]; !ok {
			t.Fatalf("expected default IP handler token cost for %q", operationName)
		}
	}
}

// Lock in the per-IP weight for authenticate as the primary boundary
// defense against single-IP auth-spray. Drift should be deliberate.
func TestAuthenticateIPCostIsElevated(t *testing.T) {
	ipCosts := defaultIPHandlerTokenCosts()
	subCosts := defaultHandlerTokenCosts()

	authAction := snx_lib_api_ratelimiting.RequestAction("authenticate")
	const wantIPCost = 100

	if got := ipCosts[authAction]; got != wantIPCost {
		t.Fatalf("expected authenticate IP cost %d, got %d", wantIPCost, got)
	}
	if ipCosts[authAction] <= subCosts[authAction] {
		t.Fatalf(
			"expected authenticate IP cost (%d) to exceed per-subaccount cost (%d)",
			ipCosts[authAction], subCosts[authAction],
		)
	}
}

// Lock in the per-IP weight for preview_auth_message. The cost is modest
// because the handler only builds typed data, but it must stay above the
// per-subaccount cost to deny single-IP amplification.
func TestPreviewAuthMessageIPCostIsElevated(t *testing.T) {
	ipCosts := defaultIPHandlerTokenCosts()
	subCosts := defaultHandlerTokenCosts()

	previewAction := snx_lib_api_ratelimiting.RequestAction("preview_auth_message")
	const wantIPCost = 10

	if got := ipCosts[previewAction]; got != wantIPCost {
		t.Fatalf("expected preview_auth_message IP cost %d, got %d", wantIPCost, got)
	}
	if ipCosts[previewAction] <= subCosts[previewAction] {
		t.Fatalf(
			"expected preview_auth_message IP cost (%d) to exceed per-subaccount cost (%d)",
			ipCosts[previewAction], subCosts[previewAction],
		)
	}
}
