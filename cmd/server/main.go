package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_metrics "github.com/Fenway-snx/synthetix-mcp/internal/lib/metrics"
	snx_lib_runtime "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime"
	snx_lib_service "github.com/Fenway-snx/synthetix-mcp/internal/lib/service"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/server"
)

const ServiceNameTag = "MCP"

type metricsService interface {
	Start() error
	Stop(context.Context) error
}

type mcpService interface {
	Start() error
	Shutdown(context.Context) error
	Errors() <-chan error
}

var newMetricsService = func(logger snx_lib_logging.Logger, cfg *config.Config) metricsService {
	return snx_lib_metrics.NewServerWithPprof(
		logger,
		cfg.Metrics.Port,
		cfg.Metrics.PprofEnabled,
		cfg.Metrics.BlockProfileRate,
		cfg.Metrics.MutexProfileFraction,
	)
}

var newMCPService = func(logger snx_lib_logging.Logger, cfg *config.Config) (mcpService, error) {
	return server.New(logger, cfg)
}

func main() {
	snx_lib_service.BootstrapService(
		ServiceNameTag,
		config.Load,
		start,
	)
}

func start(
	dc snx_lib_runtime.DiagnosticContext,
	ec snx_lib_runtime.ExecutionContext,
	cfg *config.Config,
) {
	logger := dc.Logger()
	logger.Info(
		"Starting MCP service",
		"environment", cfg.Environment,
		"server_address", cfg.ServerAddress,
		"server_name", cfg.ServerName,
		"server_version", cfg.ServerVersion,
	)

	svc, err := newMCPService(logger, cfg)
	if err != nil {
		logger.Error("Failed to create MCP server", "error", err)
		panic(fmt.Errorf("server creation failed: %w", err))
	}

	metricsServer := newMetricsService(logger, cfg)
	if err := metricsServer.Start(); err != nil {
		logger.Error("Failed to start metrics server", "error", err)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if shutdownErr := svc.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("Failed to stop MCP server during startup cleanup", "error", shutdownErr)
		}
		if stopErr := metricsServer.Stop(shutdownCtx); stopErr != nil {
			logger.Error("Failed to stop metrics server during startup cleanup", "error", stopErr)
		}
		panic(fmt.Errorf("metrics server startup failed: %w", err))
	}

	if err := svc.Start(); err != nil {
		logger.Error("Failed to start MCP server", "error", err)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if shutdownErr := svc.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("Failed to stop MCP server during startup cleanup", "error", shutdownErr)
		}
		if stopErr := metricsServer.Stop(shutdownCtx); stopErr != nil {
			logger.Error("Failed to stop metrics server during startup cleanup", "error", stopErr)
		}
		panic(fmt.Errorf("server startup failed: %w", err))
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	var shutdownReason string
	select {
	case err := <-svc.Errors():
		logger.Error("MCP server runtime error", "error", err)
		shutdownReason = "runtime_error"
	case sig := <-signalChan:
		logger.Info("Received shutdown signal", "signal", sig.String())
		shutdownReason = "signal"
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	logger.Info("Stopping MCP service", "reason", shutdownReason)
	if err := svc.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to stop MCP server cleanly", "error", err)
	}
	if err := metricsServer.Stop(shutdownCtx); err != nil {
		logger.Error("Failed to stop metrics server cleanly", "error", err)
	}
}
