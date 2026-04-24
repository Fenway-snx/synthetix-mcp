package jetstreamqueues

import (
	"context"
	"errors"
)

// Runs several collectors in sequence, merges backlog rows sorted by stream then consumer, and returns
// errors.Join of all part errors (nil if every part succeeded). Failed parts contribute no rows; successful
// parts are still merged so callers get partial data during incidents. Nil parts are skipped.
type CompositeJetStreamQueueDepthCollector struct {
	parts []JetStreamQueueDepthCollector
}

// Builds from parts, omitting nil interface values. Do not pass typed-nil pointers (e.g. (*T)(nil))
// unless that concrete type's backlog collection method handles a nil receiver.
func NewCompositeJetStreamQueueDepthCollector(parts ...JetStreamQueueDepthCollector) *CompositeJetStreamQueueDepthCollector {
	var filtered []JetStreamQueueDepthCollector
	for _, p := range parts {
		if p != nil {
			filtered = append(filtered, p)
		}
	}
	return &CompositeJetStreamQueueDepthCollector{parts: filtered}
}

// Runs each part in order, merges sorted rows from successful parts, and returns errors.Join of every
// part error (or nil if all parts succeeded).
func (c *CompositeJetStreamQueueDepthCollector) CollectJetStreamQueueDepths(ctx context.Context) ([]JetStreamQueueDepth, error) {
	if c == nil || len(c.parts) == 0 {
		return []JetStreamQueueDepth{}, nil
	}
	var slices [][]JetStreamQueueDepth
	var errs []error
	for _, p := range c.parts {
		rows, err := p.CollectJetStreamQueueDepths(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		slices = append(slices, rows)
	}
	merged := MergeSortJetStreamQueueDepths(slices...)
	if merged == nil {
		merged = []JetStreamQueueDepth{}
	}
	return merged, errors.Join(errs...)
}
