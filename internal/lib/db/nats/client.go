package nats

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/spf13/viper"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Client wraps the NATS client
type Client struct {
	JetStream        jetstream.JetStream
	InstrumentedConn *InstrumentedConn
}

// Config holds NATS configuration
type Config struct {
	Name                   string // config: "name"
	Url                    string // config: "url"
	User                   string // config: "user"
	Password               string // config: "password"
	MaxReconnects          int    // config: "max_reconnects"
	ReconnectWaitInSeconds int    // config: "reconnect_wait_in_seconds"
	AsyncPublishMaxPending int    // config: "async_publish_max_pending"
	AckWorkerCount         int    // config: "ack_worker_count" — number of worker goroutines for ACK processing
	ReplicationFactor      int    // config: "replication_factor" — number of stream replicas across NATS cluster nodes
}

const (
	// Default limit for in-flight async publishes
	DefaultAsyncPublishMaxPending = 4096

	// Default number of worker goroutines processing async ACKs
	DefaultAckWorkerCount = 16
)

func LoadConfig(v *viper.Viper) (*Config, error) {
	keyPrefix := "nats"

	asyncMaxPending := v.GetInt(keyPrefix + ".async_publish_max_pending")
	if asyncMaxPending <= 0 {
		asyncMaxPending = DefaultAsyncPublishMaxPending
	}

	ackWorkerCount := v.GetInt(keyPrefix + ".ack_worker_count")
	if ackWorkerCount <= 0 {
		ackWorkerCount = DefaultAckWorkerCount
	}

	cfg := &Config{
		Name:                   v.GetString(keyPrefix + ".name"),
		Url:                    v.GetString(keyPrefix + ".url"),
		User:                   v.GetString(keyPrefix + ".user"),
		Password:               v.GetString(keyPrefix + ".password"),
		MaxReconnects:          v.GetInt(keyPrefix + ".max_reconnects"),
		ReconnectWaitInSeconds: v.GetInt(keyPrefix + ".reconnect_wait_in_seconds"),
		AsyncPublishMaxPending: asyncMaxPending,
		AckWorkerCount:         ackWorkerCount,
		ReplicationFactor:      v.GetInt(keyPrefix + ".replication_factor"),
	}

	// Validate the configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid nats config: %w", err)
	}

	return cfg, nil
}

func NewNatsClient(
	logger snx_lib_logging.Logger,
	cfg Config,
) (*Client, error) {
	options := []nats.Option{
		nats.Name(cfg.Name),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(time.Duration(time.Second * time.Duration(cfg.ReconnectWaitInSeconds))),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("Disconnected from NATS", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("Reconnected to NATS", "url", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Error("NATS error", "error", err, "subject", sub.Subject)
		}),
	}

	// Connect to NATS
	logger.Info("Connecting to NATS", "url", cfg.Url)

	if cfg.User != "" && cfg.Password != "" {
		options = append(options, nats.UserInfo(cfg.User, cfg.Password))
	}

	nc, err := nats.Connect(cfg.Url, options...)
	if err != nil {
		return nil, err
	}

	// Create JetStream context with async publish limit
	js, err := jetstream.New(nc, jetstream.WithPublishAsyncMaxPending(cfg.AsyncPublishMaxPending))
	if err != nil {
		return nil, err
	}

	logger.Info("JetStream initialized", "async_publish_max_pending", cfg.AsyncPublishMaxPending)

	resolver := NewSubjectToStreamNameResolver()

	return &Client{
		JetStream:        NewInstrumentedJetStream(js, resolver),
		InstrumentedConn: NewInstrumentedConn(nc, resolver),
	}, nil
}

func (c *Client) Close() {
	c.InstrumentedConn.Conn.Close()
}

func (c *Config) validate() error {
	var errs []string

	if c.Name == "" {
		errs = append(errs, "name is required")
	}
	if c.Url == "" {
		errs = append(errs, "url is required")
	}

	if c.User == "" {
		errs = append(errs, "user is required")
	}
	if c.Password == "" {
		errs = append(errs, "password is required")
	}

	if c.MaxReconnects < -1 {
		errs = append(errs, "max_reconnects must be greater than -1")
	}

	if c.ReconnectWaitInSeconds < 1 {
		errs = append(errs, "reconnect_wait_in_seconds must be greater than 0")
	}

	if c.ReplicationFactor < 1 {
		errs = append(errs, "replication_factor must be >= 1 (no default; set explicitly)")
	}

	if len(errs) > 0 {
		return fmt.Errorf("nats config validation failed: %s", strings.Join(errs, "; \n"))
	}

	return nil
}
