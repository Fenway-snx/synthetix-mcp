package jetstreamqueues

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/nats-io/nats.go/jetstream"
)

// Calls Info on each non-nil consumer handle and builds backlog rows (stream and consumer from server info).
// Nil map entries are skipped (e.g. reservation placeholders before the consumer exists).
func DepthsFromConsumers(
	ctx context.Context,
	consumers map[string]jetstream.Consumer,
) ([]JetStreamQueueDepth, error) {
	if len(consumers) == 0 {
		return nil, nil
	}

	out := make([]JetStreamQueueDepth, 0, len(consumers))
	for key, c := range consumers {
		if c == nil {
			continue
		}
		info, err := c.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("jetstream consumer info for %q: %w", key, err)
		}
		out = append(out, depthFromConsumerInfo(info))
	}
	sortJetStreamQueueDepthsByStreamThenConsumer(out)
	return out, nil
}

// Copies the map under lock, releases it, then builds rows via Info on each handle (same pattern as halt snapshots).
func QueueDepthsFromConsumerMap(ctx context.Context, mu *sync.RWMutex, consumers map[string]jetstream.Consumer) ([]JetStreamQueueDepth, error) {
	mu.RLock()
	snap := make(map[string]jetstream.Consumer, len(consumers))
	for k, v := range consumers {
		snap[k] = v
	}
	mu.RUnlock()

	rows, err := DepthsFromConsumers(ctx, snap)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []JetStreamQueueDepth{}, nil
	}
	return rows, nil
}

// Sorts rows in place by stream then consumer name.
func sortJetStreamQueueDepthsByStreamThenConsumer(rows []JetStreamQueueDepth) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Stream != rows[j].Stream {
			return rows[i].Stream < rows[j].Stream
		}
		return rows[i].Consumer < rows[j].Consumer
	})
}

// Concatenates input slices (nil slices add no rows) and sorts by stream then consumer name.
func MergeSortJetStreamQueueDepths(rows ...[]JetStreamQueueDepth) []JetStreamQueueDepth {
	var total int
	for _, s := range rows {
		total += len(s)
	}
	out := make([]JetStreamQueueDepth, 0, total)
	for _, s := range rows {
		out = append(out, s...)
	}
	sortJetStreamQueueDepthsByStreamThenConsumer(out)
	return out
}

func depthFromConsumerInfo(info *jetstream.ConsumerInfo) JetStreamQueueDepth {
	if info == nil {
		return JetStreamQueueDepth{}
	}
	return JetStreamQueueDepth{
		Stream:     info.Stream,
		Consumer:   consumerNameFromInfo(info),
		Pending:    int64(info.NumPending),
		AckPending: int64(info.NumAckPending),
	}
}

func consumerNameFromInfo(info *jetstream.ConsumerInfo) string {
	if info == nil {
		return ""
	}
	if info.Config.Durable != "" {
		return info.Config.Durable
	}
	if info.Config.Name != "" {
		return info.Config.Name
	}
	return info.Name
}
