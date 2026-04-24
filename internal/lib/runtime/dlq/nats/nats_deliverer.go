package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

const (
	publishTimeout = 5 * time.Second
)

type natsDeliverer struct {
	// diagnostics state
	logger snx_lib_logging.Logger
	// execution state
	ctx context.Context
	// NATS config/state
	subject string
	js      jetstream.JetStream
	// fallback state
	fallback snx_lib_dlq.DeadLetterDeliverer
}

var _ snx_lib_dlq.DeadLetterDeliverer = (*natsDeliverer)(nil)

func (nd *natsDeliverer) OnPost(envelope snx_lib_dlq.Envelope, envelopeJSONString string) error {
	ctx, cancel := context.WithTimeout(nd.ctx, publishTimeout)
	defer cancel()

	msg := &nats.Msg{
		Subject: nd.subject,
		Data:    []byte(envelopeJSONString),
	}

	ack, err := nd.js.PublishMsg(ctx, msg)
	if err != nil {

		// the following on-failure logic is abstruse by necessity, because DLQ
		// is the "reporter of last resort" so we make all possible attempts to
		// write and then report all contributory failures regardless of whether
		// we were able to make any report.

		if nd.fallback != nil {

			// : fallback available

			nd.logger.Error("Failed to publish dead letter to NATS, falling back",
				"error", err,
				"subject", nd.subject,
			)

			errF := nd.fallback.OnPost(envelope, envelopeJSONString)
			if errF != nil {

				nd.logger.Error("Failed to publish dead letter to NATS fallback",
					"error", errF,
					"subject", nd.subject,
				)

				// double `"%w"` intentional here, in this very rare circumstance
				return fmt.Errorf("failed to publish dead letter to NATS: %w; failed to publish dead letter to fallback: %w", err, errF)
			} else {

				return fmt.Errorf("failed to publish dead letter to NATS: %w; published dead letter to fallback", err)
			}
		} else {

			// : no fallback available

			nd.logger.Error("Failed to publish dead letter to NATS",
				"error", err,
				"subject", nd.subject,
			)

			return fmt.Errorf("failed to publish dead letter to NATS: %w", err)
		}
	}

	nd.logger.Warn("Dead letter posted",
		"sequence", ack.Sequence,
		"stream", ack.Stream,
		"subject", nd.subject,
	)

	return nil
}

// Creates a new instance of a type that implements [DeadLetterDeliverer] by
// publishing envelopes to a dedicated NATS JetStream DLQ stream. If a
// JetStream publish fails at runtime, the deliverer falls back to the given
// fallback deliverer (if specified).
//
// The constructor ensures the DLQ stream exists via [CreateOrUpdateStream]
// (idempotent; safe when multiple services start concurrently).
//
// Parameters:
//   - logger - Logger. May not be `nil`;
//   - ctx - Service-lifetime context for shutdown propagation. May not be
//     `nil`;
//   - js - JetStream. May not be `nil`;
//   - fallback - Optional instance of [DeadLetterDeliverer] to be used as a
//     fallback in case of failure to publish to NATS;
//
// Preconditions (checked at runtime; typed nils are not detected):
//   - `logger != nil`;
//   - `ctx != nil`;
//   - `js != nil`;
func NewNATSDeliverer(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	js jetstream.JetStream,
	replicationFactor int,
	fallback snx_lib_dlq.DeadLetterDeliverer,
) (snx_lib_dlq.DeadLetterDeliverer, error) {
	// precondition enforcement(s)

	if logger == nil {
		panic("VIOLATION: parameter `logger` may not be `nil`")
	}
	if ctx == nil {
		panic("VIOLATION: parameter `ctx` may not be `nil`")
	}
	if js == nil {
		panic("VIOLATION: parameter `js` may not be `nil`")
	}

	// NOTE: we do _NOT_ use a stub for fallback, because require clarity in
	// diagnostic messages around success/failure of NATS and/or of fallback
	// require

	// set up the stream, with a given time limit

	setupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(setupCtx, snx_lib_db_nats.CreateDLQStreamConfig(replicationFactor)); err != nil {
		return nil, fmt.Errorf("failed to create DLQ stream: %w", err)
	}

	return &natsDeliverer{
		logger:   logger,
		ctx:      ctx,
		js:       js,
		subject:  snx_lib_db_nats.SystemDLQPosted.String(),
		fallback: fallback,
	}, nil
}
