package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_metrics "github.com/Fenway-snx/synthetix-mcp/internal/lib/metrics"
	snx_lib_runtime "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime"
	snx_lib_runtime_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/server"
)

type stubMetricsService struct {
	startErr error
	stopErr  error
	started  bool
	stopped  bool
}

type stubMCPService struct {
	startErr        error
	shutdownErr     error
	started         bool
	shutdownCalled  bool
	runtimeErrsChan chan error
}

func (s *stubMetricsService) Start() error {
	s.started = true
	return s.startErr
}

func (s *stubMetricsService) Stop(context.Context) error {
	s.stopped = true
	return s.stopErr
}

func (s *stubMCPService) Start() error {
	s.started = true
	return s.startErr
}

func (s *stubMCPService) Shutdown(context.Context) error {
	s.shutdownCalled = true
	return s.shutdownErr
}

func (s *stubMCPService) Errors() <-chan error {
	return s.runtimeErrsChan
}

func testDCAndEC() (snx_lib_runtime.DiagnosticContext, snx_lib_runtime.ExecutionContext) {
	dc := snx_lib_runtime_doubles.NewStubDiagnosticContext(nil)
	ec := snx_lib_runtime_doubles.NewStubExecutionContext(context.Background())
	return dc, ec
}

func testMainConfig() *config.Config {
	return &config.Config{
		Environment:     "test",
		ServerAddress:   ":8090",
		ServerName:      "synthetix-mcp",
		ServerVersion:   "0.1.0",
		ShutdownTimeout: time.Second,
		Metrics: &snx_lib_metrics.Config{
			Port: 9100,
		},
	}
}

func restoreMainSeams() {
	newMCPService = func(logger snx_lib_logging.Logger, cfg *config.Config) (mcpService, error) {
		return server.New(logger, cfg)
	}
	newMetricsService = func(logger snx_lib_logging.Logger, cfg *config.Config) metricsService {
		return snx_lib_metrics.NewServerWithPprof(
			logger,
			cfg.Metrics.Port,
			cfg.Metrics.PprofEnabled,
			cfg.Metrics.BlockProfileRate,
			cfg.Metrics.MutexProfileFraction,
		)
	}
}

func TestStartPanicsWhenServiceCreationFails(t *testing.T) {
	t.Cleanup(restoreMainSeams)

	newMCPService = func(snx_lib_logging.Logger, *config.Config) (mcpService, error) {
		return nil, errors.New("dial failed")
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected start to panic on service creation failure")
		}
		if !strings.Contains(recovered.(error).Error(), "server creation failed: dial failed") {
			t.Fatalf("unexpected panic payload %v", recovered)
		}
	}()

	dc, ec := testDCAndEC()
	start(dc, ec, testMainConfig())
}

func TestStartPanicsWhenMetricsStartupFails(t *testing.T) {
	t.Cleanup(restoreMainSeams)

	service := &stubMCPService{runtimeErrsChan: make(chan error, 1)}
	metrics := &stubMetricsService{startErr: errors.New("metrics unavailable")}

	newMCPService = func(snx_lib_logging.Logger, *config.Config) (mcpService, error) {
		return service, nil
	}
	newMetricsService = func(snx_lib_logging.Logger, *config.Config) metricsService {
		return metrics
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected start to panic on metrics startup failure")
		}
		if !strings.Contains(recovered.(error).Error(), "metrics server startup failed: metrics unavailable") {
			t.Fatalf("unexpected panic payload %v", recovered)
		}
		if !metrics.started {
			t.Fatal("expected metrics start attempt before panic")
		}
	}()

	dc, ec := testDCAndEC()
	start(dc, ec, testMainConfig())
}

func TestStartCleansUpWhenServiceStartFails(t *testing.T) {
	t.Cleanup(restoreMainSeams)

	service := &stubMCPService{
		startErr:        errors.New("listen failed"),
		runtimeErrsChan: make(chan error, 1),
	}
	metrics := &stubMetricsService{}

	newMCPService = func(snx_lib_logging.Logger, *config.Config) (mcpService, error) {
		return service, nil
	}
	newMetricsService = func(snx_lib_logging.Logger, *config.Config) metricsService {
		return metrics
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected start to panic on service startup failure")
		}
		if !strings.Contains(recovered.(error).Error(), "server startup failed: listen failed") {
			t.Fatalf("unexpected panic payload %v", recovered)
		}
		if !service.shutdownCalled {
			t.Fatal("expected service shutdown during startup failure cleanup")
		}
		if !metrics.stopped {
			t.Fatal("expected metrics stop during startup failure cleanup")
		}
	}()

	dc, ec := testDCAndEC()
	start(dc, ec, testMainConfig())
}

func TestStartStopsServicesAfterRuntimeError(t *testing.T) {
	t.Cleanup(restoreMainSeams)

	service := &stubMCPService{runtimeErrsChan: make(chan error, 1)}
	metrics := &stubMetricsService{}

	newMCPService = func(snx_lib_logging.Logger, *config.Config) (mcpService, error) {
		return service, nil
	}
	newMetricsService = func(snx_lib_logging.Logger, *config.Config) metricsService {
		return metrics
	}

	go func() {
		service.runtimeErrsChan <- errors.New("runtime exploded")
	}()

	dc, ec := testDCAndEC()
	start(dc, ec, testMainConfig())

	if !service.started {
		t.Fatal("expected service to be started")
	}
	if !service.shutdownCalled {
		t.Fatal("expected runtime error path to shut down service")
	}
	if !metrics.stopped {
		t.Fatal("expected runtime error path to stop metrics server")
	}
}
