package jetstreamqueues

import "context"

// Implemented by services that expose consumer backlog for admin metrics (e.g. /admin/state).
type JetStreamQueueDepthCollector interface {
	CollectJetStreamQueueDepths(ctx context.Context) ([]JetStreamQueueDepth, error)
}

// Wraps a collect function where a collector interface value is required (e.g. in tests).
type FuncCollector func(context.Context) ([]JetStreamQueueDepth, error)

func (f FuncCollector) CollectJetStreamQueueDepths(ctx context.Context) ([]JetStreamQueueDepth, error) {
	return f(ctx)
}

// Single consumer backlog snapshot in the admin JSON shape.
type JetStreamQueueDepth struct {
	Stream     string `json:"stream"`
	Consumer   string `json:"consumer"`
	Pending    int64  `json:"pending"`
	AckPending int64  `json:"ack_pending"`
}
