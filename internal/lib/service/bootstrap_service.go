package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_logging_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/utils"
	snx_lib_logging_zerolog "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/zerolog"
	snx_lib_runtime "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime"
	snx_lib_runtime_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/dlq"
	snx_lib_utils_build "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/build"
)

// Prototype for the service start function.
type ServiceStart[C HasServiceConfigCommon] func(
	dc snx_lib_runtime.DiagnosticContext,
	ec snx_lib_runtime.ExecutionContext,
	cfg *C,
)

// Creates a basic log instance to be used during bootstrap.
func createBootstrapLogger[T HasServiceConfigCommon](
	loadConfig func() (*T, error),
) (
	logger snx_lib_logging.Logger,
	cfg *T,
	err error,
) {

	// Try to load config early to configure the basic logger properly
	var logOutputJSON bool = true
	var logLevel string = "info"
	if cfg, err = loadConfig(); err == nil {
		logOutputJSON = (*cfg).LogOutputJSON()
		logLevel = (*cfg).LogLevel()
	}

	// Create a basic logger that matches the configured format
	logger = snx_lib_logging_zerolog.NewLogger(
		os.Stdout,
		snx_lib_logging_zerolog.WithOutputJSON(logOutputJSON),
		snx_lib_logging_zerolog.WithLevel(snx_lib_logging_zerolog.ParseLogLevel(logLevel)),
	)

	if cfg != nil {
		raw := (*cfg).LogTags()
		if err = snx_lib_logging_utils.ValidateLogTags(raw); err != nil {
			return logger, cfg, err
		}
		if tags := snx_lib_logging_utils.ParseLogTags(raw); len(tags) > 0 {
			logger = logger.With(tags...)
		}
	}

	return
}

func makeDLQ(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	applicationName string,
) (dlq snx_lib_dlq.DeadLetterQueue, err error) {

	var deliverer snx_lib_dlq.DeadLetterDeliverer

	// for _NOW_, we will just use the StderrDeliverer ...

	deliverer, err = snx_lib_runtime_dlq.NewStderrDeliverer()
	if err != nil {
		return
	}

	dlq, _ = snx_lib_dlq.NewDLQHandler(
		logger,
		ctx,
		deliverer,
		snx_lib_dlq.Envelope{
			Application: applicationName,
		},
	)

	return
}

// Bootstraps a service, including instantiating a logger for use during
// bootstrap, then entering running with restart (via
// `RunServiceWithRestart()`).
func BootstrapService[T HasServiceConfigCommon](
	serviceName string,
	loadConfig func() (*T, error),
	serviceFunc ServiceStart[T],
) {
	logger, cfg, err := createBootstrapLogger(loadConfig)

	if err != nil {
		logger.Error("Failed to load configuration", "error", err)

		panic(fmt.Errorf("failed to load config: %w", err))
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Service panic",
				"panic", fmt.Sprintf("%v", r),
				"stack", string(debug.Stack()),
			)
			os.Exit(1)
		}
	}()

	// Create root cancellable context for entire service lifecycle that we
	// can cancel and that listens for the interrupt signal from the OS
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dlq, err := makeDLQ(
		logger,
		ctx,
		serviceName,
	)
	if err != nil {
		logger.Error("Failed to create DLQ", "error", err)

		panic(fmt.Errorf("failed to create DLQ: %w", err))
	}

	deploymentMode := (*cfg).DeploymentMode()

	if !deploymentMode.IsProduction() {
		if s := deploymentMode.CanonicalString(); s != "" {
			logger = logger.With("deployment_mode", s)
		}
	}

	// TODO: remove this once everything bedded down with DLQ
	logger.Info("vcs", "commit", snx_lib_utils_build.BuildCommit())

	if deploymentMode.IsLocal() {
		// emit NEWLINE to stderr each second for observation

		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

		outerLoop:
			for {
				select {
				case <-ctx.Done():

					break outerLoop
				case <-ticker.C:

					fmt.Fprintln(os.Stderr)
				}
			}
		}()
	}

	logger.Info(fmt.Sprintf("Starting %s service...", serviceName),
		"log_level", (*cfg).LogLevel(),
	)

	dcb := snx_lib_runtime.DiagnosticContextBuilder{}

	ecb := snx_lib_runtime.ExecutionContextBuilder{}

	dc := dcb.
		Logger(logger).
		Build()

	ec := ecb.
		ContextAndCancel(ctx, cancel).
		DLQ(dlq).
		Build()

	serviceFunc(
		dc,
		ec,
		cfg,
	)
}
