package backend

import (
	"context"
	"fmt"
	"net/http"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/synthetixio/synthetix-go/restinfo"
	"github.com/synthetixio/synthetix-go/resttrade"
	"github.com/synthetixio/synthetix-go/wsinfo"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	sdklogger "github.com/synthetixio/synthetix-go/logger"
)

// Bridges lib/logging into the SDK's BYO-logger interface. Identical
// signatures except for With's return type, so a thin re-wrap is all
// that's needed.
type sdkLoggerAdapter struct{ inner snx_lib_logging.Logger }

func (a sdkLoggerAdapter) Debug(msg string, kv ...any) { a.inner.Debug(msg, kv...) }
func (a sdkLoggerAdapter) Info(msg string, kv ...any)  { a.inner.Info(msg, kv...) }
func (a sdkLoggerAdapter) Warn(msg string, kv ...any)  { a.inner.Warn(msg, kv...) }
func (a sdkLoggerAdapter) Error(msg string, kv ...any) { a.inner.Error(msg, kv...) }
func (a sdkLoggerAdapter) With(kv ...any) sdklogger.Logger {
	return sdkLoggerAdapter{a.inner.With(kv...)}
}

// Clients bundles the public-API transport clients the mcp-service
// depends on. Standalone image: no internal-gRPC / NATS / Redis.
type Clients struct {
	RESTInfo  *restinfo.Client
	RESTTrade *resttrade.Client
	WSInfo    *wsinfo.Client

	logger        snx_lib_logging.Logger
	readyOverride func(context.Context) error
}

func (c *Clients) SetReadyOverride(fn func(context.Context) error) {
	if c == nil {
		return
	}
	c.readyOverride = fn
}

func NewClients(
	logger snx_lib_logging.Logger,
	cfg *config.Config,
) (*Clients, error) {
	if cfg.APIBaseURL == "" {
		return nil, fmt.Errorf("SNXMCP_API_BASE_URL is required")
	}

	sdkLog := sdkLoggerAdapter{inner: logger}

	restInfoClient, err := restinfo.NewClient(restinfo.Config{
		BaseURL:        cfg.APIBaseURL,
		HTTPTimeout:    cfg.APIHTTPTimeout,
		MarketCacheTTL: cfg.APIMarketCacheTTL,
		HTTPClient:     &http.Client{Timeout: cfg.APIHTTPTimeout},
		Logger:         sdkLog,
	})
	if err != nil {
		return nil, fmt.Errorf("build REST info client: %w", err)
	}

	restTradeClient, err := resttrade.NewClient(resttrade.Config{
		BaseURL:     cfg.APIBaseURL,
		HTTPTimeout: cfg.APIHTTPTimeout,
		HTTPClient:  &http.Client{Timeout: cfg.APIHTTPTimeout},
		Logger:      sdkLog,
	})
	if err != nil {
		return nil, fmt.Errorf("build REST trade client: %w", err)
	}

	wsInfoClient, err := wsinfo.NewClient(wsinfo.Config{
		BaseURL: cfg.APIBaseURL,
		Logger:  sdkLog,
	})
	if err != nil {
		return nil, fmt.Errorf("build WS info client: %w", err)
	}

	return &Clients{
		RESTInfo:  restInfoClient,
		RESTTrade: restTradeClient,
		WSInfo:    wsInfoClient,
		logger:    logger,
	}, nil
}

func (c *Clients) Close() error {
	if c == nil {
		return nil
	}
	if c.WSInfo != nil {
		if err := c.WSInfo.Close(); err != nil && c.logger != nil {
			c.logger.Warn("close wsinfo client failed", "error", err)
		}
	}
	return nil
}

// Best-effort REST liveness probe; standalone image has no internal
// infra to gate on.
func (c *Clients) Ready(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("backend clients not initialized")
	}
	if c.readyOverride != nil {
		return c.readyOverride(ctx)
	}
	if c.RESTInfo == nil {
		return fmt.Errorf("REST info client not initialized")
	}
	return nil
}
