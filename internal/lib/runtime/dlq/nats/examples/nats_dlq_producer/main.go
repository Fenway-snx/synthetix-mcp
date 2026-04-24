package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	clasp "github.com/synesissoftware/CLASP.Go"
	libCLImate "github.com/synesissoftware/libCLImate.Go"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_logging_zerolog "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/zerolog"
	snx_lib_runtime_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/dlq"
	snx_lib_runtime_dlq_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/dlq/nats"
)

const (
	applicationName = "nats_dlq_producer"

	replicationFactor = 1 // TODO: determine whether to obtain from config
)

func main() {
	natsURL := nats.DefaultURL

	climate, _ := libCLImate.Init(func(cl *libCLImate.Climate) error {
		cl.Version = []int{0, 0, 1}
		cl.InfoLines = []string{
			"Synthetix DLQ Tools",
			"Posts a JSON string as a dead letter entry via NATS",
			":version:",
			"",
		}

		cl.AddOptionFunc(clasp.Option("--nats-url").SetAlias("-n").SetHelp("NATS server URL"), func(o *clasp.Argument, a *clasp.Specification) {
			natsURL = o.Value
		})

		cl.ValueNames = []string{"JSON string"}
		cl.ValuesConstraint = []int{1}
		cl.ValuesString = "<json-string>"

		return nil
	}, libCLImate.InitFlag_PanicOnFailure)

	result, _ := climate.ParseAndVerify(os.Args, libCLImate.ParseFlag_PanicOnFailure)

	jsonString := result.Values[0].Value

	var letter json.RawMessage
	if err := json.Unmarshal([]byte(jsonString), &letter); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: invalid JSON: %v\n", err)
		os.Exit(1)
	}

	logger := snx_lib_logging_zerolog.NewLogger(os.Stderr)

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: connect to NATS: %v\n", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: create JetStream context: %v\n", err)
		os.Exit(1)
	}

	fallback, _ := snx_lib_runtime_dlq.NewStderrDeliverer()

	ctx := context.Background()

	deliverer, err := snx_lib_runtime_dlq_nats.NewNATSDeliverer(
		logger,
		ctx,
		js,
		replicationFactor,
		fallback,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: create NATS deliverer: %v\n", err)
		os.Exit(1)
	}

	dlq, _ := snx_lib_dlq.NewDLQHandler(
		logger,
		ctx,
		deliverer,
		snx_lib_dlq.Envelope{
			Application: applicationName,
		},
	)

	if err := dlq.Post(letter, snx_lib_dlq.Envelope{}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: post returned: %v\n", err)
	}

	fmt.Fprintln(os.Stderr, "posted")
}
