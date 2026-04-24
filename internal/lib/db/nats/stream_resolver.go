package nats

import (
	"strings"
	"sync"

	"github.com/nats-io/nats.go/jetstream"
)

type matchRule struct {
	pattern string
	stream  string
}

// Maps NATS subjects to JetStream stream names (e.g. "snx-v1-EXECUTION_EVENTS").
// Built once from stream configs at initialization. Exact subjects are
// pre-populated into the cache; pattern rules (prefix/suffix/contains) are
// immutable slices checked on cache miss. Safe for concurrent use.
type SubjectToStreamNameResolver struct {
	prefixes []matchRule
	contains []matchRule

	cache sync.Map // subject -> stream name (string, "" if no stream)
}

// Builds a resolver from all known stream configurations.
// Static streams populate from config builders in stream.go.
// Dynamic per-symbol streams (ORDERS, ORDERBOOK_JOURNAL) use pattern matching.
// Safe for concurrent use after construction.
func NewSubjectToStreamNameResolver() *SubjectToStreamNameResolver {
	r := &SubjectToStreamNameResolver{}

	r.registerFromConfig(CreateExecutionEventsStreamConfig(1))
	r.registerFromConfig(CreateLiquidationEventsStreamConfig(1))
	r.registerFromConfig(CreateAccountLifecycleStreamConfig(1))
	r.registerFromConfig(CreateFundingEventsStreamConfig(1))
	r.registerFromConfig(CreateADLRankingsStreamConfig(1))
	r.registerFromConfig(CreateAccountsTreasuryStreamConfig(1))
	r.registerFromConfig(CreateRelayerTxnQueueStreamConfig(1))
	r.registerFromConfig(CreateOrderBookJournalStreamConfig(1))
	r.registerFromConfig(CreateOrdersStreamConfig(1))

	r.contains = []matchRule{
		{pattern: "journal.orderbook", stream: StreamName_OrderBookJournal.String()},
		{pattern: "traded.event", stream: StreamName_OrderBookJournal.String()},
	}

	return r
}

func (r *SubjectToStreamNameResolver) registerFromConfig(cfg jetstream.StreamConfig) {
	for _, subj := range cfg.Subjects {
		if strings.ContainsAny(subj, "*>") {
			prefix := strings.TrimRight(subj, ".*>")
			r.prefixes = append(r.prefixes, matchRule{pattern: prefix, stream: cfg.Name})
		} else {
			r.cache.Store(subj, cfg.Name)
		}
	}
}

// Returns the full JetStream stream name for the given subject, or ""
// if the subject does not belong to any known stream (e.g. core NATS
// price feeds, recovery subjects). Results are cached per unique subject
// string, except for prefix matches (e.g. _INBOX.*) which contain unique
// IDs and would grow the cache unboundedly.
func (r *SubjectToStreamNameResolver) Resolve(subject string) string {
	if v, ok := r.cache.Load(subject); ok {
		return v.(string)
	}

	stream, shouldCache := r.resolveByPattern(subject)
	if shouldCache {
		r.cache.Store(subject, stream)
	}
	return stream
}

// Note: these patterns make assumptions about our code and should be checked before general use.
func (r *SubjectToStreamNameResolver) resolveByPattern(subject string) (string, bool) {
	// _INBOX subjects contain unique IDs per request — never cache.
	if strings.HasPrefix(subject, "_INBOX") {
		return "INBOX", false
	}

	for i := range r.prefixes {
		if strings.HasPrefix(subject, r.prefixes[i].pattern) {
			return r.prefixes[i].stream, true
		}
	}

	for i := range r.contains {
		if strings.Contains(subject, r.contains[i].pattern) {
			return r.contains[i].stream, true
		}
	}

	return "", true
}
