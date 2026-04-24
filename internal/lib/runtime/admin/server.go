package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

const (
	adminHealthEndpoint     = "/admin/health"
	serverIdleTimeout       = 60 * time.Second
	serverReadHeaderTimeout = 10 * time.Second
	serverReadTimeout       = 30 * time.Second
	serverWriteTimeout      = 30 * time.Second
)

type AdminHTTPServer struct {
	logger snx_lib_logging.Logger
	config *Config
	mux    *http.ServeMux
	server *http.Server

	mu      sync.Mutex
	started bool
}

func NewAdminHTTPServer(logger snx_lib_logging.Logger, cfg *Config) *AdminHTTPServer {
	mux := http.NewServeMux()

	mux.HandleFunc(adminHealthEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &AdminHTTPServer{
		config: cfg,
		logger: logger,
		mux:    mux,
	}
}

// Registers an HTTP handler on the admin server.
// Must be called before Start().
func (s *AdminHTTPServer) RegisterRoute(pattern string, handler http.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		s.logger.Warn("attempted to register route after server start",
			"pattern", pattern,
		)
		return
	}

	s.mux.Handle(pattern, handler)
}

func (s *AdminHTTPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		Handler:           s.mux,
		IdleTimeout:       serverIdleTimeout,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
	}

	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to start admin HTTP server: %w", err)
	}

	s.started = true

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("admin HTTP server panic", "panic", r)
			}
		}()

		s.logger.Info("starting admin HTTP server", "port", s.config.Port)
		if serveErr := s.server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			s.logger.Error("admin HTTP server failed", "error", serveErr)
		}
	}()

	return nil
}

func (s *AdminHTTPServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	s.logger.Info("shutting down admin HTTP server")
	shutdownErr := s.server.Shutdown(ctx)
	s.started = false

	return shutdownErr
}
