package metrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Server provides HTTP endpoint for Prometheus/Datadog metrics scraping
type Server struct {
	server *http.Server
	logger snx_lib_logging.Logger
}

// NewServer creates a new metrics HTTP server (without pprof)
// port: The port to listen on (e.g., 9090)
// logger: Logger instance for logging
func NewServer(
	logger snx_lib_logging.Logger,
	port int,
) *Server {
	return NewServerWithPprof(logger, port, false, 0, 0)
}

// NewServerWithPprof creates a new metrics HTTP server with optional pprof endpoints
// port: The port to listen on (e.g., 9090)
// logger: Logger instance for logging
// enablePprof: If true, registers pprof debug endpoints on the same server
// blockProfileRate: Block profile rate (0 = disabled, 1 = all events). Ignored if enablePprof is false.
// mutexProfileFraction: Mutex profile fraction (0 = disabled, 1 = all events). Ignored if enablePprof is false.
func NewServerWithPprof(
	logger snx_lib_logging.Logger,
	port int,
	enablePprof bool,
	blockProfileRate, mutexProfileFraction int,
) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle("/metrics", promhttp.Handler())

	// Register pprof endpoints if enabled
	if enablePprof {
		// Configure runtime profiling rates
		if blockProfileRate > 0 {
			runtime.SetBlockProfileRate(blockProfileRate)
			logger.Info("Block profiling enabled", "rate", blockProfileRate)
		}
		if mutexProfileFraction > 0 {
			runtime.SetMutexProfileFraction(mutexProfileFraction)
			logger.Info("Mutex profiling enabled", "fraction", mutexProfileFraction)
		}

		// Register standard pprof HTTP handlers
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

		logger.Info("Pprof endpoints enabled on metrics server", "prefix", "/debug/pprof/")
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Server{
		server: server,
		logger: logger,
	}
}

// Start starts the metrics HTTP server in a background goroutine
func (m *Server) Start() error {
	m.logger.Info("Starting metrics HTTP server", "addr", m.server.Addr)

	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Metrics server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the metrics server
func (m *Server) Stop(ctx context.Context) error {
	m.logger.Info("Stopping metrics HTTP server")
	return m.server.Shutdown(ctx)
}
