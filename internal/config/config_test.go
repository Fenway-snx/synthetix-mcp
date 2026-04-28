package config

import (
	"strings"
	"testing"
	"time"
)

func TestApplyDefaultsSetsEveryZeroValue(t *testing.T) {
	cfg := &Config{}

	applyDefaults(cfg)

	if cfg.APIBaseURL != DefaultAPIBaseURL {
		t.Fatalf("expected API base URL default %q, got %q", DefaultAPIBaseURL, cfg.APIBaseURL)
	}
	if cfg.AgentBroker.DefaultPreset != DefaultAgentBrokerPreset {
		t.Fatalf("expected broker default preset %q, got %q", DefaultAgentBrokerPreset, cfg.AgentBroker.DefaultPreset)
	}
	if len(cfg.AgentBroker.DefaultAllowedSymbols) != 1 || cfg.AgentBroker.DefaultAllowedSymbols[0] != DefaultAgentBrokerAllowedSymbol {
		t.Fatalf("expected broker default symbols %s, got %#v", DefaultAgentBrokerAllowedSymbol, cfg.AgentBroker.DefaultAllowedSymbols)
	}
	if len(cfg.AgentBroker.DefaultAllowedTypes) != 2 || cfg.AgentBroker.DefaultAllowedTypes[0] != "LIMIT" || cfg.AgentBroker.DefaultAllowedTypes[1] != "MARKET" {
		t.Fatalf("expected broker default order types LIMIT,MARKET, got %#v", cfg.AgentBroker.DefaultAllowedTypes)
	}
	if cfg.AuthCacheMaxEntries != DefaultAuthCacheMaxEntries {
		t.Fatalf("expected auth cache default %d, got %d", DefaultAuthCacheMaxEntries, cfg.AuthCacheMaxEntries)
	}
	if cfg.EIP712ChainID != DefaultEIP712ChainID {
		t.Fatalf("expected EIP-712 chain ID default %d, got %d", DefaultEIP712ChainID, cfg.EIP712ChainID)
	}
	if cfg.EIP712DomainName != DefaultEIP712DomainName {
		t.Fatalf("expected EIP-712 domain name default %q, got %q", DefaultEIP712DomainName, cfg.EIP712DomainName)
	}
	if cfg.EIP712DomainVersion != DefaultEIP712DomainVersion {
		t.Fatalf("expected EIP-712 domain version default %q, got %q", DefaultEIP712DomainVersion, cfg.EIP712DomainVersion)
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
	if cfg.LogLevel != DefaultLogLevel {
		t.Fatalf("expected log level default %q, got %q", DefaultLogLevel, cfg.LogLevel)
	}
	if cfg.MaxRequestBodyBytes != DefaultMaxRequestBodyBytes {
		t.Fatalf("expected request body default %d, got %d", DefaultMaxRequestBodyBytes, cfg.MaxRequestBodyBytes)
	}
	if cfg.MaxSubscriptionsPerSession != DefaultMaxSubscriptions {
		t.Fatalf("expected max subscriptions default %d, got %d", DefaultMaxSubscriptions, cfg.MaxSubscriptionsPerSession)
	}
	if cfg.ServerAddress != DefaultServerAddress {
		t.Fatalf("expected server address default %q, got %q", DefaultServerAddress, cfg.ServerAddress)
	}
	if cfg.ServerName != DefaultServerName {
		t.Fatalf("expected server name default %q, got %q", DefaultServerName, cfg.ServerName)
	}
	if cfg.ServerVersion != DefaultServerVersion {
		t.Fatalf("expected server version default %q, got %q", DefaultServerVersion, cfg.ServerVersion)
	}
	if cfg.SessionStorePath != "" {
		t.Fatalf("expected session store path default empty, got %q", cfg.SessionStorePath)
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
		APIBaseURL: "https://example.invalid",
		AgentBroker: AgentBrokerConfig{
			DefaultAllowedSymbols: []string{"ETH-USDT"},
			DefaultAllowedTypes:   []string{"LIMIT"},
			DefaultPreset:         "read_only",
		},
		AuthCacheMaxEntries:        11,
		EIP712ChainID:              42,
		EIP712DomainName:           "Custom",
		EIP712DomainVersion:        "2",
		Environment:                "production",
		APIHTTPTimeout:             7 * time.Second,
		HTTPIdleTimeout:            8 * time.Second,
		HTTPReadHeaderTimeout:      9 * time.Second,
		LogLevel:                   "debug",
		MaxRequestBodyBytes:        123,
		MaxSubscriptionsPerSession: 14,
		ServerAddress:              "127.0.0.1:9999",
		ServerName:                 "custom-mcp",
		ServerVersion:              "2.3.4",
		SessionStorePath:           "./sessions.db",
		SessionTTL:                 17 * time.Second,
		ShutdownTimeout:            18 * time.Second,
	}

	applyDefaults(cfg)

	if cfg.APIBaseURL != "https://example.invalid" {
		t.Fatalf("expected explicit API base URL to be preserved, got %q", cfg.APIBaseURL)
	}
	if cfg.AgentBroker.DefaultPreset != "read_only" {
		t.Fatalf("expected explicit broker default preset to be preserved, got %q", cfg.AgentBroker.DefaultPreset)
	}
	if len(cfg.AgentBroker.DefaultAllowedSymbols) != 1 || cfg.AgentBroker.DefaultAllowedSymbols[0] != "ETH-USDT" {
		t.Fatalf("expected explicit broker symbols to be preserved, got %#v", cfg.AgentBroker.DefaultAllowedSymbols)
	}
	if len(cfg.AgentBroker.DefaultAllowedTypes) != 1 || cfg.AgentBroker.DefaultAllowedTypes[0] != "LIMIT" {
		t.Fatalf("expected explicit broker order types to be preserved, got %#v", cfg.AgentBroker.DefaultAllowedTypes)
	}
	if cfg.AuthCacheMaxEntries != 11 {
		t.Fatalf("expected explicit auth cache entries to be preserved, got %d", cfg.AuthCacheMaxEntries)
	}
	if cfg.EIP712ChainID != 42 {
		t.Fatalf("expected explicit EIP-712 chain ID to be preserved, got %d", cfg.EIP712ChainID)
	}
	if cfg.EIP712DomainName != "Custom" {
		t.Fatalf("expected explicit EIP-712 domain name to be preserved, got %q", cfg.EIP712DomainName)
	}
	if cfg.EIP712DomainVersion != "2" {
		t.Fatalf("expected explicit EIP-712 domain version to be preserved, got %q", cfg.EIP712DomainVersion)
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
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected explicit log level to be preserved, got %q", cfg.LogLevel)
	}
	if cfg.MaxRequestBodyBytes != 123 {
		t.Fatalf("expected explicit body size to be preserved, got %d", cfg.MaxRequestBodyBytes)
	}
	if cfg.MaxSubscriptionsPerSession != 14 {
		t.Fatalf("expected explicit max subscriptions to be preserved, got %d", cfg.MaxSubscriptionsPerSession)
	}
	if cfg.ServerAddress != "127.0.0.1:9999" {
		t.Fatalf("expected explicit server address to be preserved, got %q", cfg.ServerAddress)
	}
	if cfg.ServerName != "custom-mcp" {
		t.Fatalf("expected explicit server name to be preserved, got %q", cfg.ServerName)
	}
	if cfg.ServerVersion != "2.3.4" {
		t.Fatalf("expected explicit server version to be preserved, got %q", cfg.ServerVersion)
	}
	if cfg.SessionStorePath != "./sessions.db" {
		t.Fatalf("expected explicit session store path to be preserved, got %q", cfg.SessionStorePath)
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
	t.Setenv("SNXMCP_AGENT_BROKER_ENABLED", "false")
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
	if cfg.APIBaseURL != DefaultAPIBaseURL {
		t.Fatalf("expected default API base URL %q, got %q", DefaultAPIBaseURL, cfg.APIBaseURL)
	}
}

func TestLoadDefaultsAgentBrokerEnabled(t *testing.T) {
	t.Setenv("SNXMCP_AGENT_BROKER_ENABLED", "")
	t.Setenv("SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX", strings.Repeat("1", 64))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}
	if !cfg.AgentBroker.Enabled {
		t.Fatalf("expected agent broker to default enabled")
	}
}

func TestLoadAllowsExplicitAgentBrokerDisable(t *testing.T) {
	setRequiredLoadEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}
	if cfg.AgentBroker.Enabled {
		t.Fatalf("expected explicit agent broker disable to be preserved")
	}
}

func TestLoadReadsBrokerDefaultGuardrailEnvironment(t *testing.T) {
	t.Setenv("SNXMCP_AGENT_BROKER_PRIVATE_KEY_HEX", strings.Repeat("1", 64))
	t.Setenv("SNXMCP_AGENT_BROKER_DEFAULT_PRESET", "standard")
	t.Setenv("SNXMCP_AGENT_BROKER_DEFAULT_ALLOWED_SYMBOLS", "BTC-USDT, ETH-USDT")
	t.Setenv("SNXMCP_AGENT_BROKER_DEFAULT_ALLOWED_ORDER_TYPES", "LIMIT, MARKET")
	t.Setenv("SNXMCP_AGENT_BROKER_DEFAULT_MAX_ORDER_NOTIONAL", "1000")
	t.Setenv("SNXMCP_AGENT_BROKER_DEFAULT_MAX_POSITION_QUANTITY", "2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}
	if got := cfg.AgentBroker.DefaultAllowedSymbols; len(got) != 2 || got[0] != "BTC-USDT" || got[1] != "ETH-USDT" {
		t.Fatalf("expected allowed symbols from env, got %#v", got)
	}
	if got := cfg.AgentBroker.DefaultAllowedTypes; len(got) != 2 || got[0] != "LIMIT" || got[1] != "MARKET" {
		t.Fatalf("expected allowed order types from env, got %#v", got)
	}
	if cfg.AgentBroker.DefaultMaxOrderNotional != "1000" {
		t.Fatalf("expected max order notional from env, got %q", cfg.AgentBroker.DefaultMaxOrderNotional)
	}
	if cfg.AgentBroker.DefaultMaxPositionQty != "2" {
		t.Fatalf("expected max position quantity from env, got %q", cfg.AgentBroker.DefaultMaxPositionQty)
	}
}

func TestLoadBrokerEnabledWithoutKeyExplainsHowToProceed(t *testing.T) {
	_, err := Load()
	if err == nil {
		t.Fatal("expected config validation to fail")
	}

	message := err.Error()
	if !strings.Contains(message, "agent_broker_enabled=true requires agent_broker_private_key_hex or agent_broker_private_key_file") {
		t.Fatalf("expected broker key validation, got %s", message)
	}
	if !strings.Contains(message, "./scripts/setup-broker-key.sh") {
		t.Fatalf("expected terminal setup remediation, got %s", message)
	}
	if !strings.Contains(message, "--no-broker") {
		t.Fatalf("expected read-only/external-wallet remediation, got %s", message)
	}
}
