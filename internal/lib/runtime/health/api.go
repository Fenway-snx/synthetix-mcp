package health

import (
	"context"
	"fmt"
	"net"
	"net/http"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_runtime_health_handlers "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/handlers"
	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
)

type HealthHTTPReporter struct {
	logger      snx_lib_logging.Logger
	config      *Config
	stateReader snx_lib_runtime_health_types.IStateReader
	server      *http.Server
	ErrorChan   chan error
}

func NewHealthHTTPReporter(
	logger snx_lib_logging.Logger,
	config *Config,
	stateReader snx_lib_runtime_health_types.IStateReader,
) *HealthHTTPReporter {
	return &HealthHTTPReporter{
		logger:      logger,
		stateReader: stateReader,
		config:      config,
		ErrorChan:   make(chan error, 1),
	}
}

// This method returns an error immediately if the server cannot start (e.g., port is already in use).
// Runtime errors after successful startup are sent to the error channel returned by the ErrorChan.
// NOTE: this may only be called once on a given instance. Pretty please
func (h *HealthHTTPReporter) Start() error {
	mux := http.NewServeMux()

	healthHandler := snx_lib_runtime_health_handlers.NewHealthHandler(h.logger, h.stateReader)
	mux.Handle(h.config.Endpoint, &healthHandler)

	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.config.Port),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", h.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to start health API server: %w", err)
	}

	go func() {

		defer func() {
			if r := recover(); r != nil {
				h.ErrorChan <- fmt.Errorf("health API panic: %v", r)
			}
		}()

		h.logger.Info("Starting health API server",
			"endpoint", h.config.Endpoint,
			"port", h.config.Port,
		)

		if err := h.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			h.logger.Error("Health API server failed", "error", err)
			h.ErrorChan <- err
		}
	}()

	return nil
}

func (h *HealthHTTPReporter) Shutdown(ctx context.Context) error {
	if h.server == nil {
		return nil
	}
	h.logger.Info("Shutting down health API server")
	return h.server.Shutdown(ctx)
}
