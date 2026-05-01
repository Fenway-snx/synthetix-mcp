package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	internal_auth "github.com/Fenway-snx/synthetix-mcp/internal/auth"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
)

type shutdownStubClients struct {
	closeErr error
	readyErr error
	closed   bool
}

type shutdownStubAuthManager struct {
	closeErr error
	closed   bool
}

type shutdownStubStreaming struct {
	closeErr error
	closed   bool
}

func (s *shutdownStubClients) Close() error {
	s.closed = true
	return s.closeErr
}

func (s *shutdownStubClients) Ready(context.Context) error {
	return s.readyErr
}

func (s *shutdownStubAuthManager) Authenticate(context.Context, string, string, string) (*internal_auth.AuthenticateResult, error) {
	panic("unexpected Authenticate call")
}

func (s *shutdownStubAuthManager) Close() error {
	s.closed = true
	return s.closeErr
}

func (s *shutdownStubAuthManager) ValidateTradeAction(string, int64, int64, int64, snx_lib_api_types.RequestAction, any, snx_lib_auth.TradeSignature) error {
	panic("unexpected ValidateTradeAction call")
}

func (s *shutdownStubAuthManager) VerifySessionAccess(context.Context, string, int64) error {
	panic("unexpected VerifySessionAccess call")
}

func (s *shutdownStubStreaming) Close() error {
	s.closed = true
	return s.closeErr
}

func explicitServerConfig() *config.Config {
	return &config.Config{
		APIHTTPTimeout:             time.Second,
		HTTPIdleTimeout:            time.Second,
		HTTPReadHeaderTimeout:      time.Second,
		MaxRequestBodyBytes:        1 << 20,
		ServerAddress:              "127.0.0.1:0",
		ServerName:                 "synthetix-mcp",
		ServerVersion:              "0.1.0",
		SessionTTL:                 30 * time.Minute,
		MaxSubscriptionsPerSession: 10,
	}
}

func TestLivenessCheckReturnsLiveStatus(t *testing.T) {
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), explicitServerConfig(), &stubClients{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	srv.livenessCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
	if !strings.Contains(rec.Body.String(), `"status":"live"`) {
		t.Fatalf("expected live status body, got %s", rec.Body.String())
	}
}

func TestHealthCheckReturnsUnhealthyPayload(t *testing.T) {
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), explicitServerConfig(), &stubClients{
		readyErr: errors.New("store unavailable"),
	}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.healthCheck(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"unhealthy"`) {
		t.Fatalf("expected unhealthy body, got %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `store unavailable`) {
		t.Fatalf("expected readiness error to be redacted, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `dependency readiness check failed`) {
		t.Fatalf("expected redacted readiness error in body, got %s", rec.Body.String())
	}
}

func TestPublicHealthErrorRecognizesWrappedDeadlineExceeded(t *testing.T) {
	got := publicHealthError(errors.Join(errors.New("store unavailable"), context.DeadlineExceeded))
	if got != "dependency readiness check timed out" {
		t.Fatalf("expected wrapped deadline exceeded to be recognized, got %q", got)
	}
}

func TestStartAndShutdownLifecycleSucceeds(t *testing.T) {
	clients := &shutdownStubClients{}
	authManager := &shutdownStubAuthManager{}
	streamingManager := &shutdownStubStreaming{}
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), explicitServerConfig(), clients, authManager, streamingManager, nil)

	if err := srv.Start(); err != nil {
		t.Fatalf("expected server start to succeed, got %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected shutdown to succeed, got %v", err)
	}
	if !clients.closed {
		t.Fatal("expected client close to be invoked")
	}
	if !authManager.closed {
		t.Fatal("expected auth manager close to be invoked")
	}
	if !streamingManager.closed {
		t.Fatal("expected streaming manager close to be invoked")
	}
}

func TestStartReturnsListenErrorForInvalidAddress(t *testing.T) {
	cfg := explicitServerConfig()
	cfg.ServerAddress = "127.0.0.1:not-a-port"
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), cfg, &stubClients{}, nil, nil, nil)

	err := srv.Start()
	if err == nil {
		t.Fatal("expected start to fail for invalid listen address")
	}
	if !strings.Contains(err.Error(), "listen on 127.0.0.1:not-a-port") {
		t.Fatalf("unexpected listen error %v", err)
	}
}

func TestShutdownReturnsAuthManagerErrorBeforeLowerPriorityErrors(t *testing.T) {
	clients := &shutdownStubClients{closeErr: errors.New("client close failed")}
	authManager := &shutdownStubAuthManager{closeErr: errors.New("auth close failed")}
	streamingManager := &shutdownStubStreaming{closeErr: errors.New("stream close failed")}
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), explicitServerConfig(), clients, authManager, streamingManager, nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if err == nil {
		t.Fatal("expected shutdown to return an error")
	}
	if err.Error() != "auth close failed" {
		t.Fatalf("expected auth manager error precedence, got %v", err)
	}
	if !clients.closed {
		t.Fatal("expected clients to still be closed even when auth close fails")
	}
	if !streamingManager.closed {
		t.Fatal("expected streaming manager to still be closed even when auth close fails")
	}
}
