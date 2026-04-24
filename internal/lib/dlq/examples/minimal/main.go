package main

import (
	"context"
	"fmt"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_runtime_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/dlq"
)

const (
	applicationName = "DLQ Minimal Example"
	systemName      = ""
	subsystemName   = ""
)

// Create a DLQ.
func createDLQ(
	ctx context.Context,
	logger snx_lib_logging.Logger,
) (dlq snx_lib_dlq.DeadLetterQueue) {

	deliverer, err := snx_lib_runtime_dlq.NewStderrDeliverer()
	if err != nil {
		panic(fmt.Errorf("failed to initialise the deliverer: %w", err))
	}

	dlq, _ = snx_lib_dlq.NewDLQHandler(
		logger,
		ctx,
		deliverer,
		snx_lib_dlq.Envelope{
			Application: applicationName,
			System:      systemName,
			Subsystem:   subsystemName,
		},
	)

	return
}

func main() {
	logger := snx_lib_logging.NewNoOpLogger()

	dlq := createDLQ(
		context.Background(),
		logger,
	)

	start(dlq)
}

func start(dlq snx_lib_dlq.DeadLetterQueue) {

	// fmt.Fprintf(os.Stderr, "before dumping a letter to the DLQ ...\n")
	// fmt.Fprintln(os.Stderr)

	// letter := "help!"
	letter := map[string]any{
		"plea": "help!",
		"func": func() {},
	}

	dlq.Post(letter, snx_lib_dlq.Envelope{})

	// fmt.Fprintln(os.Stderr)
	// fmt.Fprintf(os.Stderr, "after dumping a letter to the DLQ ...\n")
}
