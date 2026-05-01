package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_logging_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/utils"
	snx_lib_logging_zerolog "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/zerolog"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/server"
)

type mcpService interface {
	Start() error
	Shutdown(context.Context) error
	Errors() <-chan error
}

var newMCPService = func(logger snx_lib_logging.Logger, cfg *config.Config) (mcpService, error) {
	return server.New(logger, cfg)
}

func main() {
	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	logger, err := newLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("MCP service panic",
				"panic", fmt.Sprintf("%v", r),
				"stack", string(debug.Stack()),
			)
			os.Exit(1)
		}
	}()
	if err := start(logger, cfg); err != nil {
		logger.Error("MCP service stopped with error", "error", err)
		os.Exit(1)
	}
}

func loadConfig(args []string) (*config.Config, error) {
	fs := flag.NewFlagSet("synthetix-mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	noBroker := fs.Bool("no-broker", false, "start without self-hosted broker tools; read-only and signed_* tools remain available")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *noBroker {
		if err := os.Setenv("SNXMCP_AGENT_BROKER_ENABLED", "false"); err != nil {
			return nil, fmt.Errorf("set no-broker env override: %w", err)
		}
	}
	return config.Load()
}

func newLogger(cfg *config.Config) (snx_lib_logging.Logger, error) {
	logger := snx_lib_logging_zerolog.NewLogger(
		os.Stdout,
		snx_lib_logging_zerolog.WithOutputJSON(cfg.LogOutputJSON),
		snx_lib_logging_zerolog.WithLevel(snx_lib_logging_zerolog.ParseLogLevel(cfg.LogLevel)),
	)
	if cfg.LogTags == "" {
		return logger, nil
	}
	if err := snx_lib_logging_utils.ValidateLogTags(cfg.LogTags); err != nil {
		return nil, err
	}
	return logger.With(snx_lib_logging_utils.ParseLogTags(cfg.LogTags)...), nil
}

func start(logger snx_lib_logging.Logger, cfg *config.Config) error {
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
		return fmt.Errorf("server creation failed: %w", err)
	}

	if err := svc.Start(); err != nil {
		logger.Error("Failed to start MCP server", "error", err)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if shutdownErr := svc.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("Failed to stop MCP server during startup cleanup", "error", shutdownErr)
		}
		return fmt.Errorf("server startup failed: %w", err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	var shutdownReason string
	var runtimeErr error
	select {
	case err := <-svc.Errors():
		logger.Error("MCP server runtime error", "error", err)
		shutdownReason = "runtime_error"
		runtimeErr = err
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
	if runtimeErr != nil {
		return fmt.Errorf("server runtime error: %w", runtimeErr)
	}
	return nil
}
