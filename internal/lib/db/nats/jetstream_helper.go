package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// JetStreamHelper provides resilient JetStream operations for all services
type JetStreamHelper struct {
	js     jetstream.JetStream
	logger snx_lib_logging.Logger
}

// NewJetStreamHelper creates a new JetStream helper
func NewJetStreamHelper(
	logger snx_lib_logging.Logger,
	js jetstream.JetStream,
) *JetStreamHelper {
	return &JetStreamHelper{
		js:     js,
		logger: logger,
	}
}

// CreateConsumerWithRetry creates a consumer with retry logic and stream existence check
// This is the recommended way to create consumers in all services
func (h *JetStreamHelper) CreateConsumerWithRetry(
	ctx context.Context,
	streamName string,
	config jetstream.ConsumerConfig,
) (jetstream.Consumer, error) {
	maxRetries := 30 // ~5 minutes with exponential backoff
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	// Get consumer name for logging (prefer Durable over Name)
	consumerName := config.Durable
	if consumerName == "" {
		consumerName = config.Name
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// First check if stream exists
		stream, err := h.js.Stream(ctx, streamName)
		if err != nil {
			if errors.Is(err, jetstream.ErrStreamNotFound) {
				h.logger.Warn("Stream not found, waiting...",
					"stream", streamName,
					"attempt", attempt+1,
					"max_retries", maxRetries,
					"next_retry_in", backoff)
			} else {
				h.logger.Error("Error accessing stream",
					"stream", streamName,
					"error", err)
			}

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled while waiting for stream: %w", ctx.Err())
			case <-time.After(backoff):
				// Exponential backoff
				backoff = backoff * 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
		}

		consumer, err := stream.CreateOrUpdateConsumer(ctx, config)
		if err == nil {
			h.logger.Info("Consumer created successfully",
				"stream", streamName,
				"consumer", consumerName,
				"attempt", attempt+1)
			return consumer, nil
		}

		h.logger.Warn("Failed to create consumer, retrying...",
			"stream", streamName,
			"consumer", consumerName,
			"error", err,
			"attempt", attempt+1)

		// Check for corrupted consumer metadata (err_code=10012)
		// This typically indicates missing or corrupted consumer files
		if isConsumerCorruptedError(err) {
			h.logger.Warn("Detected corrupted consumer metadata (err_code=10012), attempting to delete and recreate",
				"stream", streamName,
				"consumer", consumerName)

			// Try to delete the corrupted consumer
			if deleteErr := stream.DeleteConsumer(ctx, consumerName); deleteErr != nil {
				h.logger.Warn("Failed to delete corrupted consumer, it may not exist",
					"stream", streamName,
					"consumer", consumerName,
					"error", deleteErr)
				// Continue anyway - the consumer might not exist or might be in a bad state
			} else {
				h.logger.Info("Successfully deleted corrupted consumer",
					"stream", streamName,
					"consumer", consumerName)
			}

			// Reset backoff for immediate retry after deletion
			backoff = 1 * time.Second

			// Wait a moment before retry to allow NATS to clean up
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}

		// Wait before retry with consumer-specific backoff
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Update main backoff for next stream check
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return nil, fmt.Errorf("failed to create consumer after %d retries", maxRetries)
}

// isConsumerCorruptedError checks if the error is a corrupted consumer metadata error (err_code=10012)
// This error typically occurs when consumer metadata files are missing or corrupted
func isConsumerCorruptedError(err error) bool {
	// Check for ErrConsumerCreate (err_code=10012) which indicates consumer creation failure
	// typically due to corrupted or missing metadata files
	return errors.Is(err, jetstream.ErrConsumerCreate)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// Don't retry on context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Add other non-retryable errors as needed
	// For now, most errors are retryable
	return true
}
