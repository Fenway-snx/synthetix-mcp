package redis

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	go_redis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Alias for redis' Nil
const Nil = go_redis.Nil

// Client wraps the Redis cluster client with common operations
type SnxClient struct {
	*go_redis.ClusterClient
}

// Config holds Redis configuration
type Config struct {
	Addr          string // config: "addr" — Redis cluster address (any node)
	Username      string // config: "username"
	Password      string // config: "password"
	PoolSize      int    // config: "pool_size"
	MinIdleConns  int    // config: "min_idle_conns"
	MaxRetries    int    // config: "max_retries"
	IsTLS         bool   // config: "is_tls"
	TLSSkipVerify bool   // config: "tls_skip_verify"
}

// LoadConfig loads Redis configuration from Viper with the given key prefix
// For example, if keyPrefix is "redis", it will look for go_redis.addr, go_redis.password, etc.
// Returns an error if required fields are missing or invalid.
func LoadConfig(v *viper.Viper) (*Config, error) {
	keyPrefix := "redis"
	cfg := &Config{
		Addr:          v.GetString(keyPrefix + ".addr"),
		Username:      v.GetString(keyPrefix + ".username"),
		Password:      v.GetString(keyPrefix + ".password"),
		PoolSize:      v.GetInt(keyPrefix + ".pool_size"),
		MinIdleConns:  v.GetInt(keyPrefix + ".min_idle_conns"),
		MaxRetries:    v.GetInt(keyPrefix + ".max_retries"),
		IsTLS:         v.GetBool(keyPrefix + ".is_tls"),
		TLSSkipVerify: v.GetBool(keyPrefix + ".tls_skip_verify"),
	}

	// Validate the configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	return cfg, nil
}

// NewClient creates a new Redis cluster client
func NewClient(
	logger snx_lib_logging.Logger,
	cfg Config,
) (*SnxClient, error) {
	return NewClientWithHostnameMapping(logger, cfg, nil)
}

// NewClientWithHostnameMapping creates a new Redis cluster client with optional hostname mapping.
// The hostnameMap parameter allows mapping Docker service names (e.g., "redis") to localhost
// for testing scenarios where tests run on the host but Redis runs in Docker.
// If hostnameMap is nil, no mapping is applied.
func NewClientWithHostnameMapping(
	logger snx_lib_logging.Logger,
	cfg Config,
	hostnameMap map[string]string,
) (*SnxClient, error) {

	var tlsConfig *tls.Config
	if cfg.IsTLS {
		// Create default TLS configuration
		tlsConfig = &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
	}

	opts := &go_redis.ClusterOptions{
		Addrs:        []string{cfg.Addr},
		Username:     cfg.Username,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		TLSConfig:    tlsConfig,
	}

	// If hostname mapping is provided, use a custom dialer to map hostnames
	if hostnameMap != nil {
		opts.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := parseAddr(addr)
			if err != nil {
				return nil, err
			}

			// Map hostname if mapping exists
			if mappedHost, ok := hostnameMap[host]; ok {
				addr = fmt.Sprintf("%s:%s", mappedHost, port)
			}

			dialer := &net.Dialer{}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	logger.Info("Connecting to Redis cluster", "addr", cfg.Addr)
	rdb := go_redis.NewClusterClient(opts)
	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
	}

	return &SnxClient{ClusterClient: rdb}, nil
}

// parseAddr parses an address string in the format "host:port" and returns host and port
func parseAddr(addr string) (host, port string, err error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid address format: %s", addr)
	}
	return parts[0], parts[1], nil
}

// NOTE: this shameful HACK is necessary in order to prevent a lot of
// hurried last-minute refactoring and abstraction around the
// trade-handler and whitelist-arbitrator.
func (rc *SnxClient) IsValid() bool {
	return rc.ClusterClient != nil
}

// SetJSON stores a JSON-encoded value
func (rc *SnxClient) SetJSON(ctx context.Context, key string, value any, expiration time.Duration) (*go_redis.StatusCmd, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	setResult := rc.Set(ctx, key, data, expiration)

	if setResult.Err() != nil {
		return nil, fmt.Errorf("failed to set JSON: %w", setResult.Err())
	}

	return setResult, nil
}

// GetJSON retrieves and decodes a JSON value
func GetJSON[T any](
	ctx context.Context,
	rc *SnxClient,
	key string,
	dest *T,
) error {
	data, err := rc.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// FlushDB clears all data from all nodes in the cluster
func (rc *SnxClient) FlushDB(ctx context.Context) error {
	// In cluster mode, we need to flush each node
	return rc.ForEachShard(ctx, func(ctx context.Context, shard *go_redis.Client) error {
		return shard.FlushDB(ctx).Err()
	})
}

func (c *Config) validate() error {
	var errs []string

	if c.Addr == "" {
		errs = append(errs, "addr is required")
	}

	if c.PoolSize < 0 {
		errs = append(errs, "pool_size must be greater than or equal to 0")
	}

	if c.MinIdleConns < 0 {
		errs = append(errs, "min_idle_conns must be greater than or equal to 0")
	}

	if c.MaxRetries < 0 {
		errs = append(errs, "max_retries must be greater than or equal to 0")
	}

	if len(errs) > 0 {
		return fmt.Errorf("redis config validation failed: %s", strings.Join(errs, ";\n"))
	}

	return nil
}
