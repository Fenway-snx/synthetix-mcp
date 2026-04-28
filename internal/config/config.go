package config

import (
	"fmt"
	"strings"
	"time"

	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	libconfig "github.com/Fenway-snx/synthetix-mcp/internal/lib/config"
)

const (
	DefaultAPIBaseURL                     = "https://papi.synthetix.io"
	DefaultAPIHTTPTimeout                 = 10 * time.Second
	DefaultAPIMarketCacheTTL              = 30 * time.Second
	DefaultAgentBrokerAllowedSymbol       = "*"
	DefaultAgentBrokerPreset              = "standard"
	DefaultAuthCacheMaxEntries            = snx_lib_auth.DefaultAuthCacheMaxEntries
	DefaultEIP712ChainID                  = 1
	DefaultEIP712DomainName               = "Synthetix"
	DefaultEIP712DomainVersion            = "1"
	DefaultEnvironment                    = "development"
	DefaultHTTPIdleTimeout                = 120 * time.Second
	DefaultHTTPReadHeaderTimeout          = 10 * time.Second
	DefaultLogLevel                       = "info"
	DefaultMaxRequestBodyBytes      int64 = 1 << 20
	DefaultMaxSubscriptions               = 25
	DefaultRiskSnapshotMaxAge             = 3 * time.Minute
	DefaultServerAddress                  = "127.0.0.1:8096"
	DefaultServerName                     = "synthetix-mcp"
	DefaultServerVersion                  = "0.1.0"
	DefaultSessionTTL                     = 30 * time.Minute
	DefaultShutdownTimeout                = 10 * time.Second
)

// AgentBroker configures an optional, server-side EIP-712 signer. When
// enabled the MCP server holds a private key in process memory and uses
// it to fulfil the canonical broker write tools (auto-authenticate +
// auto-sign trade actions). This collapses the "agent writes a TS file
// and rummages for PRIVATE_KEY in .env" onboarding loop into one tool
// call, at the cost of moving custody onto the server.
//
// Only one of PrivateKeyHex / PrivateKeyFile may be set. The mode is
// enabled by default and refuses to start unless ServerAddress binds
// to loopback (or AllowNonLoopback is explicitly enabled) so a
// misconfigured production deploy cannot accidentally expose a
// hot wallet to the public internet.
type AgentBrokerConfig struct {
	AllowNonLoopback           bool
	DefaultMaxOrderNotional    string
	DefaultMaxOrderQty         string
	DefaultMaxPositionNotional string
	DefaultMaxPositionQty      string
	DefaultPreset              string
	DefaultAllowedSymbols      []string
	DefaultAllowedTypes        []string
	Enabled                    bool
	PrivateKeyFile             string
	PrivateKeyHex              string
	SubAccountID               int64
}

type Config struct {
	AgentBroker AgentBrokerConfig
	// APIBaseURL is the public Synthetix REST+WS endpoint that
	// mcp-service talks to (e.g. "https://papi.synthetix.io",
	// "http://localhost:8080"). Defaults to DefaultAPIBaseURL.
	APIBaseURL string
	// APIHTTPTimeout caps the total round-trip for a single REST
	// request to APIBaseURL. Defaults to DefaultAPIHTTPTimeout (10s)
	// when unset; client-side tuning knob.
	APIHTTPTimeout time.Duration
	// APIMarketCacheTTL controls how long the RESTInfo client caches
	// the full getMarkets response to back single-market lookups.
	// Defaults to DefaultAPIMarketCacheTTL (30s) when unset;
	// client-side tuning knob. Set to a negative duration to
	// disable caching entirely.
	APIMarketCacheTTL          time.Duration
	AuthCacheMaxEntries        int
	EIP712ChainID              int64
	EIP712DomainName           string
	EIP712DomainVersion        string
	Environment                string
	HTTPIdleTimeout            time.Duration
	HTTPReadHeaderTimeout      time.Duration
	LogLevel                   string
	LogOutputJSON              bool
	LogTags                    string
	MaxRequestBodyBytes        int64
	MaxSubscriptionsPerSession int
	RiskSnapshotMaxAge         time.Duration
	ServerAddress              string
	ServerName                 string
	ServerVersion              string
	SessionStorePath           string
	SessionTTL                 time.Duration
	ShutdownTimeout            time.Duration
}

func Load() (*Config, error) {
	v, err := libconfig.Load("SNXMCP")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	var validationErrors []string

	agentBrokerEnabled := true
	if v.IsSet("agent_broker_enabled") {
		agentBrokerEnabled = v.GetBool("agent_broker_enabled")
	}

	cfg := &Config{
		AgentBroker: AgentBrokerConfig{
			AllowNonLoopback:           v.GetBool("agent_broker_allow_non_loopback"),
			DefaultAllowedSymbols:      parseCommaSeparatedList(v.GetString("agent_broker_default_allowed_symbols")),
			DefaultAllowedTypes:        parseCommaSeparatedList(v.GetString("agent_broker_default_allowed_order_types")),
			DefaultMaxOrderNotional:    v.GetString("agent_broker_default_max_order_notional"),
			DefaultMaxOrderQty:         v.GetString("agent_broker_default_max_order_quantity"),
			DefaultMaxPositionNotional: v.GetString("agent_broker_default_max_position_notional"),
			DefaultMaxPositionQty:      v.GetString("agent_broker_default_max_position_quantity"),
			DefaultPreset:              v.GetString("agent_broker_default_preset"),
			Enabled:                    agentBrokerEnabled,
			PrivateKeyFile:             v.GetString("agent_broker_private_key_file"),
			PrivateKeyHex:              v.GetString("agent_broker_private_key_hex"),
			SubAccountID:               v.GetInt64("agent_broker_sub_account_id"),
		},
		AuthCacheMaxEntries:        v.GetInt("auth_cache_max_entries"),
		APIBaseURL:                 v.GetString("api_base_url"),
		APIHTTPTimeout:             v.GetDuration("api_http_timeout"),
		APIMarketCacheTTL:          v.GetDuration("api_market_cache_ttl"),
		EIP712ChainID:              v.GetInt64("eip712_chain_id"),
		EIP712DomainName:           v.GetString("eip712_domain_name"),
		EIP712DomainVersion:        v.GetString("eip712_domain_version"),
		Environment:                v.GetString("environment"),
		HTTPIdleTimeout:            v.GetDuration("http_idle_timeout"),
		HTTPReadHeaderTimeout:      v.GetDuration("http_read_header_timeout"),
		LogLevel:                   v.GetString("log_level"),
		LogOutputJSON:              v.GetBool("log_output_json"),
		LogTags:                    v.GetString("log_tags"),
		MaxRequestBodyBytes:        v.GetInt64("max_request_body_bytes"),
		MaxSubscriptionsPerSession: v.GetInt("max_subscriptions_per_session"),
		RiskSnapshotMaxAge:         v.GetDuration("risk_snapshot_max_age"),
		ServerAddress:              v.GetString("server_address"),
		ServerName:                 v.GetString("server_name"),
		ServerVersion:              v.GetString("server_version"),
		SessionStorePath:           v.GetString("session_store_path"),
		SessionTTL:                 v.GetDuration("session_ttl"),
		ShutdownTimeout:            v.GetDuration("shutdown_timeout"),
	}

	applyDefaults(cfg)

	if cfg.EIP712ChainID <= 0 {
		validationErrors = append(validationErrors, "eip712_chain_id must be greater than 0")
	}
	if cfg.MaxRequestBodyBytes <= 0 {
		validationErrors = append(validationErrors, "max_request_body_bytes must be greater than 0")
	}
	if cfg.MaxSubscriptionsPerSession <= 0 {
		validationErrors = append(validationErrors, "max_subscriptions_per_session must be greater than 0")
	}
	if cfg.AuthCacheMaxEntries <= 0 {
		validationErrors = append(validationErrors, "auth_cache_max_entries must be greater than 0")
	}
	if cfg.RiskSnapshotMaxAge <= 0 {
		validationErrors = append(validationErrors, "risk_snapshot_max_age must be greater than 0")
	}
	if cfg.SessionTTL <= 0 {
		validationErrors = append(validationErrors, "session_ttl must be greater than 0")
	}
	if cfg.ShutdownTimeout <= 0 {
		validationErrors = append(validationErrors, "shutdown_timeout must be greater than 0")
	}
	if err := validateAgentBrokerConfig(&cfg.AgentBroker, cfg.ServerAddress); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("mcp config validation failed: %s", strings.Join(validationErrors, ";\n"))
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = DefaultAPIBaseURL
	}
	if cfg.AgentBroker.DefaultPreset == "" {
		cfg.AgentBroker.DefaultPreset = DefaultAgentBrokerPreset
	}
	if len(cfg.AgentBroker.DefaultAllowedSymbols) == 0 {
		cfg.AgentBroker.DefaultAllowedSymbols = []string{DefaultAgentBrokerAllowedSymbol}
	}
	if len(cfg.AgentBroker.DefaultAllowedTypes) == 0 {
		cfg.AgentBroker.DefaultAllowedTypes = []string{"LIMIT", "MARKET"}
	}
	if cfg.AuthCacheMaxEntries == 0 {
		cfg.AuthCacheMaxEntries = DefaultAuthCacheMaxEntries
	}
	if cfg.EIP712ChainID == 0 {
		cfg.EIP712ChainID = DefaultEIP712ChainID
	}
	if cfg.EIP712DomainName == "" {
		cfg.EIP712DomainName = DefaultEIP712DomainName
	}
	if cfg.EIP712DomainVersion == "" {
		cfg.EIP712DomainVersion = DefaultEIP712DomainVersion
	}
	if cfg.Environment == "" {
		cfg.Environment = DefaultEnvironment
	}
	if cfg.APIHTTPTimeout == 0 {
		cfg.APIHTTPTimeout = DefaultAPIHTTPTimeout
	}
	if cfg.APIMarketCacheTTL == 0 {
		cfg.APIMarketCacheTTL = DefaultAPIMarketCacheTTL
	}
	if cfg.HTTPIdleTimeout == 0 {
		cfg.HTTPIdleTimeout = DefaultHTTPIdleTimeout
	}
	if cfg.HTTPReadHeaderTimeout == 0 {
		cfg.HTTPReadHeaderTimeout = DefaultHTTPReadHeaderTimeout
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = DefaultLogLevel
	}
	if cfg.MaxRequestBodyBytes == 0 {
		cfg.MaxRequestBodyBytes = DefaultMaxRequestBodyBytes
	}
	if cfg.MaxSubscriptionsPerSession == 0 {
		cfg.MaxSubscriptionsPerSession = DefaultMaxSubscriptions
	}
	if cfg.RiskSnapshotMaxAge == 0 {
		cfg.RiskSnapshotMaxAge = DefaultRiskSnapshotMaxAge
	}
	if cfg.ServerAddress == "" {
		cfg.ServerAddress = DefaultServerAddress
	}
	if cfg.ServerName == "" {
		cfg.ServerName = DefaultServerName
	}
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = DefaultServerVersion
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = DefaultSessionTTL
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = DefaultShutdownTimeout
	}
}

func parseCommaSeparatedList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}

// Hard-fails the load when AgentBroker is enabled with no key, or when
// the server is bound to a non-loopback interface unless the operator
// has explicitly opted into it. Returning an error keeps the failure
// at startup rather than silently disabling the broker tools at
// runtime, which would silently regress the onboarding UX.
func validateAgentBrokerConfig(cfg *AgentBrokerConfig, serverAddress string) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	hasInline := strings.TrimSpace(cfg.PrivateKeyHex) != ""
	hasFile := strings.TrimSpace(cfg.PrivateKeyFile) != ""
	switch {
	case !hasInline && !hasFile:
		return fmt.Errorf(
			"agent_broker_enabled=true requires agent_broker_private_key_hex or agent_broker_private_key_file; " +
				"run ./scripts/setup-broker-key.sh from a terminal, set SNXMCP_AGENT_BROKER_PRIVATE_KEY_FILE, or start read-only/external-wallet mode with --no-broker",
		)
	case hasInline && hasFile:
		return fmt.Errorf("agent_broker_private_key_hex and agent_broker_private_key_file are mutually exclusive")
	}
	if !cfg.AllowNonLoopback && !addressIsLoopback(serverAddress) {
		return fmt.Errorf(
			"agent_broker_enabled=true requires server_address to bind to loopback "+
				"(got %q); set agent_broker_allow_non_loopback=true to opt out and accept that "+
				"the broker private key is reachable from the bound interface",
			serverAddress,
		)
	}
	return nil
}

// Loopback or unspecified-host means "trustable local". An empty host
// such as ":8096" listens on all interfaces and is rejected unless the
// operator opted in via AllowNonLoopback. Hostnames that resolve later
// (e.g. docker DNS) are also rejected here; the broker is intended for
// laptop/dev use and operators running it elsewhere can flip the flag.
func addressIsLoopback(serverAddress string) bool {
	host := serverAddress
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	switch host {
	case "127.0.0.1", "::1", "localhost":
		return true
	default:
		return false
	}
}
