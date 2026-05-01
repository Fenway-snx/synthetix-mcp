package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/server"
)

type stubMCPService struct {
	startErr        error
	shutdownErr     error
	started         bool
	shutdownCalled  bool
	runtimeErrsChan chan error
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

func testMainConfig() *config.Config {
	return &config.Config{
		Environment:     "test",
		ServerAddress:   ":8090",
		ServerName:      "synthetix-mcp",
		ServerVersion:   "0.1.0",
		ShutdownTimeout: time.Second,
		LogLevel:        "debug",
		LogOutputJSON:   true,
	}
}

func restoreMainSeams() {
	newMCPService = func(logger snx_lib_logging.Logger, cfg *config.Config) (mcpService, error) {
		return server.New(logger, cfg)
	}
}

func TestStartReturnsErrorWhenServiceCreationFails(t *testing.T) {
	t.Cleanup(restoreMainSeams)

	newMCPService = func(snx_lib_logging.Logger, *config.Config) (mcpService, error) {
		return nil, errors.New("dial failed")
	}

	err := start(snx_lib_logging_doubles.NewStubLogger(), testMainConfig())
	if err == nil {
		t.Fatal("expected start to return service creation failure")
	}
	if !strings.Contains(err.Error(), "server creation failed: dial failed") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestStartCleansUpWhenServiceStartFails(t *testing.T) {
	t.Cleanup(restoreMainSeams)

	service := &stubMCPService{
		startErr:        errors.New("listen failed"),
		runtimeErrsChan: make(chan error, 1),
	}

	newMCPService = func(snx_lib_logging.Logger, *config.Config) (mcpService, error) {
		return service, nil
	}

	err := start(snx_lib_logging_doubles.NewStubLogger(), testMainConfig())
	if err == nil {
		t.Fatal("expected start to return service startup failure")
	}
	if !strings.Contains(err.Error(), "server startup failed: listen failed") {
		t.Fatalf("unexpected error %v", err)
	}
	if !service.shutdownCalled {
		t.Fatal("expected service shutdown during startup failure cleanup")
	}
}

func TestStartStopsServicesAfterRuntimeError(t *testing.T) {
	t.Cleanup(restoreMainSeams)

	service := &stubMCPService{runtimeErrsChan: make(chan error, 1)}

	newMCPService = func(snx_lib_logging.Logger, *config.Config) (mcpService, error) {
		return service, nil
	}

	go func() {
		service.runtimeErrsChan <- errors.New("runtime exploded")
	}()

	err := start(snx_lib_logging_doubles.NewStubLogger(), testMainConfig())
	if err == nil || !strings.Contains(err.Error(), "server runtime error: runtime exploded") {
		t.Fatalf("expected runtime error, got %v", err)
	}

	if !service.started {
		t.Fatal("expected service to be started")
	}
	if !service.shutdownCalled {
		t.Fatal("expected runtime error path to shut down service")
	}
}

func TestLoadConfigNoBrokerFlagDisablesBroker(t *testing.T) {
	t.Setenv("SNXMCP_AGENT_BROKER_ENABLED", "")

	cfg, err := loadConfig([]string{"--no-broker"})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.AgentBroker.Enabled {
		t.Fatal("expected --no-broker to disable agent broker")
	}
}
