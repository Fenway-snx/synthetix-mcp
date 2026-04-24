package config

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	snx_lib_api_ratelimiting "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/rate_limiting"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	libconfig "github.com/Fenway-snx/synthetix-mcp/internal/lib/config"
	snx_lib_metrics "github.com/Fenway-snx/synthetix-mcp/internal/lib/metrics"
	snx_lib_service "github.com/Fenway-snx/synthetix-mcp/internal/lib/service"
)

const (
	DefaultAPIHTTPTimeout             = 10 * time.Second
	DefaultAPIMarketCacheTTL          = 30 * time.Second
	DefaultAuthCacheMaxEntries        = snx_lib_auth.DefaultAuthCacheMaxEntries
	DefaultEnvironment                = "development"
	DefaultHTTPIdleTimeout            = 120 * time.Second
	DefaultHTTPReadHeaderTimeout      = 10 * time.Second
	DefaultMaxRequestBodyBytes int64  = 1 << 20
	DefaultMaxSubscriptions           = 25
	DefaultRiskSnapshotMaxAge         = 3 * time.Minute
	DefaultServerName                 = "synthetix-mcp"
	DefaultServerVersion              = "0.1.0"
	DefaultSessionTTL                 = 30 * time.Minute
	DefaultShutdownTimeout            = 10 * time.Second
)

// AgentBroker configures an optional, server-side EIP-712 signer. When
// enabled the MCP server holds a private key in process memory and uses
// it to fulfil the new `quick_*` intent tools (auto-authenticate +
// auto-sign trade actions). This collapses the "agent writes a TS file
// and rummages for PRIVATE_KEY in .env" onboarding loop into one tool
// call, at the cost of moving custody onto the server.
//
// Only one of PrivateKeyHex / PrivateKeyFile may be set. The mode is
// disabled by default and refuses to start unless ServerAddress binds
// to loopback (or AllowNonLoopback is explicitly enabled) so a
// misconfigured production deploy cannot accidentally expose a
// hot wallet to the public internet.
type AgentBrokerConfig struct {
	AllowNonLoopback        bool
	DefaultMaxOrderQty      string
	DefaultMaxPositionQty   string
	DefaultPreset           string
	DefaultAllowedSymbols   []string
	DefaultAllowedTypes     []string
	Enabled                 bool
	PrivateKeyFile          string
	PrivateKeyHex           string
	SubAccountID            int64
}

type Config struct {
	snx_lib_service.ServiceConfigCommon

	AgentBroker                AgentBrokerConfig
	// APIBaseURL is the public Synthetix REST+WS endpoint that
	// mcp-service talks to (e.g. "https://api.synthetix.io",
	// "http://localhost:8080"). Required; no built-in default.
	APIBaseURL                 string
	// APIHTTPTimeout caps the total round-trip for a single REST
	// request to APIBaseURL. Defaults to DefaultAPIHTTPTimeout (10s)
	// when unset; client-side tuning knob.
	APIHTTPTimeout             time.Duration
	// APIMarketCacheTTL controls how long the RESTInfo client caches
	// the full getMarkets response to back single-market lookups.
	// Defaults to DefaultAPIMarketCacheTTL (30s) when unset;
	// client-side tuning knob. Set to a negative duration to
	// disable caching entirely.
	APIMarketCacheTTL          time.Duration
	AuthCacheMaxEntries        int
	AuthRPSPerSubAccount       int
	EIP712ChainID              int64
	EIP712DomainName           string
	EIP712DomainVersion        string
	Environment                string
	HandlerTokenCosts          snx_lib_api_ratelimiting.HandlerTokenCosts
	HTTPIdleTimeout            time.Duration
	HTTPReadHeaderTimeout      time.Duration
	IPHandlerTokenCosts        snx_lib_api_ratelimiting.HandlerTokenCosts
	IPRateLimiterConfig        snx_lib_api_ratelimiting.PerIPRateLimiterConfig
	MaxRequestBodyBytes        int64
	MaxSubscriptionsPerSession int
	Metrics                    *snx_lib_metrics.Config
	OrderRateLimiterConfig     snx_lib_api_ratelimiting.PerSubAccountRateLimiterConfig
	PublicRPSPerIP             int
	RiskSnapshotMaxAge         time.Duration
	ServerAddress              string
	ServerName                 string
	ServerVersion              string
	SessionTTL                 time.Duration
	ShutdownTimeout            time.Duration
	TrustedProxyCIDRs          []string
}

func Load() (*Config, error) {
	v, err := libconfig.Load("SNXMCP")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	var validationErrors []string

	metricsConfig, err := snx_lib_metrics.LoadConfig(v)
	if err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	ipRateLimiterConfig, err := snx_lib_api_ratelimiting.LoadIPRateLimiterConfig(v)
	if err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	orderRateLimiterConfig, err := snx_lib_api_ratelimiting.LoadOrderRateLimiterConfig(v)
	if err != nil {
		validationErrors = append(validationErrors, err.Error())
	}
	handlerTokenCosts, err := snx_lib_api_ratelimiting.LoadHandlerTokenCosts(v)
	if err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	ipHandlerTokenCosts, err := snx_lib_api_ratelimiting.LoadIPHandlerTokenCosts(v)
	if err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	serviceConfigCommon, err := snx_lib_service.LoadServiceConfigCommon(v)
	if err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	cfg := &Config{
		ServiceConfigCommon: serviceConfigCommon,
		AgentBroker: AgentBrokerConfig{
			AllowNonLoopback:      v.GetBool("agent_broker_allow_non_loopback"),
			DefaultAllowedSymbols: parseCommaSeparatedList(v.GetString("agent_broker_default_allowed_symbols")),
			DefaultAllowedTypes:   parseCommaSeparatedList(v.GetString("agent_broker_default_allowed_order_types")),
			DefaultMaxOrderQty:    v.GetString("agent_broker_default_max_order_quantity"),
			DefaultMaxPositionQty: v.GetString("agent_broker_default_max_position_quantity"),
			DefaultPreset:         v.GetString("agent_broker_default_preset"),
			Enabled:               v.GetBool("agent_broker_enabled"),
			PrivateKeyFile:        v.GetString("agent_broker_private_key_file"),
			PrivateKeyHex:         v.GetString("agent_broker_private_key_hex"),
			SubAccountID:          v.GetInt64("agent_broker_sub_account_id"),
		},
		AuthCacheMaxEntries:        v.GetInt("auth_cache_max_entries"),
		APIBaseURL:                 v.GetString("api_base_url"),
		APIHTTPTimeout:             v.GetDuration("api_http_timeout"),
		APIMarketCacheTTL:          v.GetDuration("api_market_cache_ttl"),
		EIP712ChainID:              v.GetInt64("eip712_chain_id"),
		EIP712DomainName:           v.GetString("eip712_domain_name"),
		EIP712DomainVersion:        v.GetString("eip712_domain_version"),
		Environment:                v.GetString("environment"),
		HandlerTokenCosts:          handlerTokenCosts,
		HTTPIdleTimeout:            v.GetDuration("http_idle_timeout"),
		HTTPReadHeaderTimeout:      v.GetDuration("http_read_header_timeout"),
		IPHandlerTokenCosts:        ipHandlerTokenCosts,
		IPRateLimiterConfig:        ipRateLimiterConfig,
		MaxRequestBodyBytes:        v.GetInt64("max_request_body_bytes"),
		MaxSubscriptionsPerSession: v.GetInt("max_subscriptions_per_session"),
		Metrics:                    metricsConfig,
		OrderRateLimiterConfig:     orderRateLimiterConfig,
		RiskSnapshotMaxAge:         v.GetDuration("risk_snapshot_max_age"),
		ServerAddress:              v.GetString("server_address"),
		ServerName:                 v.GetString("server_name"),
		ServerVersion:              v.GetString("server_version"),
		SessionTTL:                 v.GetDuration("session_ttl"),
		ShutdownTimeout:            v.GetDuration("shutdown_timeout"),
		TrustedProxyCIDRs:          parseCommaSeparatedList(v.GetString("trusted_proxy_cidrs")),
	}

	applyDefaults(cfg)
	cfg.PublicRPSPerIP = DerivedTokensPerSecond(int64(cfg.IPRateLimiterConfig.IPRateLimit), cfg.IPRateLimiterConfig.WindowMs)
	cfg.AuthRPSPerSubAccount = DerivedTokensPerSecond(int64(cfg.OrderRateLimiterConfig.GeneralRateLimit), cfg.OrderRateLimiterConfig.WindowMs)
	// Integer division above truncates to 0 for small limits relative to
	// the window. The values feed advisory hints in system://status,
	// system://server_info, and get_rate_limits, so a silent zero would
	// mislead client SDKs.
	if cfg.PublicRPSPerIP <= 0 {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"derived public_rps_per_ip must be greater than 0 "+
					"(ratelimiting_ip_rate_limit=%d over ratelimiting_window_ms=%d truncates to %d); "+
					"raise ratelimiting_ip_rate_limit or shorten ratelimiting_window_ms",
				cfg.IPRateLimiterConfig.IPRateLimit, cfg.IPRateLimiterConfig.WindowMs, cfg.PublicRPSPerIP,
			),
		)
	}
	if cfg.AuthRPSPerSubAccount <= 0 {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"derived auth_rps_per_subaccount must be greater than 0 "+
					"(ratelimiting_general_rate_limit=%d over ratelimiting_window_ms=%d truncates to %d); "+
					"raise ratelimiting_general_rate_limit or shorten ratelimiting_window_ms",
				cfg.OrderRateLimiterConfig.GeneralRateLimit, cfg.OrderRateLimiterConfig.WindowMs, cfg.AuthRPSPerSubAccount,
			),
		)
	}

	if cfg.LogLevel() == "" {
		validationErrors = append(validationErrors, "log_level is required")
	}
	if cfg.ServerAddress == "" {
		validationErrors = append(validationErrors, "server_address is required")
	}
	if cfg.APIBaseURL == "" {
		validationErrors = append(validationErrors, "api_base_url is required")
	}
	if cfg.EIP712DomainName == "" {
		validationErrors = append(validationErrors, "eip712_domain_name is required")
	}
	if cfg.EIP712DomainVersion == "" {
		validationErrors = append(validationErrors, "eip712_domain_version is required")
	}
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
	if err := validateTrustedProxyCIDRs(cfg.TrustedProxyCIDRs); err != nil {
		validationErrors = append(validationErrors, err.Error())
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
	if cfg.AuthCacheMaxEntries == 0 {
		cfg.AuthCacheMaxEntries = DefaultAuthCacheMaxEntries
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
	if cfg.MaxRequestBodyBytes == 0 {
		cfg.MaxRequestBodyBytes = DefaultMaxRequestBodyBytes
	}
	if cfg.MaxSubscriptionsPerSession == 0 {
		cfg.MaxSubscriptionsPerSession = DefaultMaxSubscriptions
	}
	if cfg.RiskSnapshotMaxAge == 0 {
		cfg.RiskSnapshotMaxAge = DefaultRiskSnapshotMaxAge
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
	if len(cfg.HandlerTokenCosts) == 0 {
		cfg.HandlerTokenCosts = defaultHandlerTokenCosts()
	}
	if len(cfg.IPHandlerTokenCosts) == 0 {
		cfg.IPHandlerTokenCosts = defaultIPHandlerTokenCosts()
	}
	if cfg.TrustedProxyCIDRs == nil {
		cfg.TrustedProxyCIDRs = []string{}
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
		return fmt.Errorf("agent_broker_enabled=true requires agent_broker_private_key_hex or agent_broker_private_key_file")
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

func validateTrustedProxyCIDRs(values []string) error {
	for _, value := range values {
		if _, err := netip.ParsePrefix(value); err != nil {
			return fmt.Errorf("trusted_proxy_cidrs contains invalid CIDR %q", value)
		}
	}
	return nil
}

func defaultHandlerTokenCosts() snx_lib_api_ratelimiting.HandlerTokenCosts {
	return snx_lib_api_ratelimiting.HandlerTokenCosts{
		"authenticate":            20,
		"cancel_all_orders":       2,
		"cancel_order":            2,
		"close_position":          5,
		"get_prompt":              5,
		"get_account_summary":     30,
		"get_candles":             200,
		"get_context":             100,
		"get_funding_payments":    100,
		"get_funding_rate":        250,
		"get_market_summary":      100,
		"get_open_orders":         10,
		"get_order_history":       50,
		"get_orderbook":           200,
		"get_performance_history": 100,
		"get_positions":           10,
		"get_rate_limits":         20,
		"get_recent_trades":       200,
		"get_server_info":         5,
		"get_session":             5,
		"get_trade_history":       50,
		"initialize":              1,
		"list_prompts":            5,
		"list_resource_templates": 5,
		"list_resources":          5,
		"list_markets":            50,
		"list_tools":              5,
		"lookup_subaccount":       30,
		"mcp_ping":                1,
		"modify_order":            5,
		"ping":                    1,
		"place_order":             5,
		"preview_auth_message":    5,
		"preview_order":           2,
		"preview_trade_signature": 2,
		// quick_* tools wrap their non-quick counterparts plus the
		// agent broker's auto-authenticate / auto-guardrails work
		// (amortised across the session TTL after the first call).
		// We charge ~2x the non-quick cost per-subaccount to reflect
		// that bootstrap amortisation; per-IP costs are elevated
		// further (see defaultIPHandlerTokenCosts) because cold
		// sessions trigger the EIP-712 boundary work.
		"quick_cancel_all_orders": 5,
		"quick_cancel_order":      5,
		"quick_close_position":    10,
		"quick_place_order":       10,
		"read_agent_guide":        5,
		"read_fee_schedule":       20,
		"read_market_specs":       100,
		"read_risk_limits":        20,
		"read_runbooks":           5,
		"read_server_info":        5,
		"read_status":             5,
		"restore_session":         5,
		"set_guardrails":          5,
		"subscribe":               10,
		"unsubscribe":             5,
	}
}

// Per-IP token costs up-weight unauthenticated handlers above their
// per-subaccount cost so a single IP cannot burn unbounded server work
// before any tenant identity is bound. Lock-in tests in config_test.go
// guard the elevated values for authenticate and preview_auth_message.
func defaultIPHandlerTokenCosts() snx_lib_api_ratelimiting.HandlerTokenCosts {
	costs := defaultHandlerTokenCosts()
	// Heavy: EIP-712 parse + ecrecover + subaccount lookup + session write.
	costs["authenticate"] = 100
	// Light typed-data construction; 2x per-subaccount keeps the boundary
	// gate without throttling multi-subaccount onboarding bursts.
	costs["preview_auth_message"] = 10
	// quick_* tools transparently invoke the broker's auto-authenticate
	// path on a cold session, which performs the same EIP-712 boundary
	// work as a manual `authenticate` call. Mirror the elevated IP cost
	// so that a single IP cannot bypass the auth-spray gate by going
	// through the broker.
	costs["quick_cancel_all_orders"] = 100
	costs["quick_cancel_order"] = 100
	costs["quick_close_position"] = 100
	costs["quick_place_order"] = 100
	return costs
}

func DerivedTokensPerSecond(limit int64, windowMs int64) int {
	if limit <= 0 || windowMs <= 0 {
		return 0
	}
	return int((limit * 1000) / windowMs)
}
