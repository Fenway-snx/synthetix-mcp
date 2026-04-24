package nats

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Client-side buffer operations (async JetStream enqueue, core NATS publish).
// Concentrated in 1μs-100μs where these operations typically land,
// with tail coverage up to 1s for degradation.
var localBuckets = []float64{
	0.000001, 0.0000025, 0.000005, 0.00001, 0.000025, 0.00005,
	0.0001, 0.00025, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.25, 1.0,
}

// Network round-trip operations (sync JetStream publish, request-reply).
// Concentrated in 100μs-5ms where NATS ACK round-trips typically land.
var networkBuckets = []float64{
	0.0001, 0.00025, 0.0005, 0.00075, 0.001, 0.0015,
	0.002, 0.003, 0.005, 0.01, 0.025, 0.05, 0.1, 0.5, 2.5,
}

var (
	jsPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nats_jetstream_sync_publish_duration_seconds",
			Help:    "Sync JetStream publish latency (call to ACK round-trip)",
			Buckets: networkBuckets,
		},
		[]string{"stream"},
	)

	jsPublishErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_jetstream_sync_publish_errors_total",
			Help: "Sync JetStream publish failures",
		},
		[]string{"stream"},
	)

	jsAsyncEnqueueDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nats_jetstream_async_enqueue_duration_seconds",
			Help:    "Async JetStream publish enqueue latency",
			Buckets: localBuckets,
		},
		[]string{"stream"},
	)

	jsAsyncEnqueueErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_jetstream_async_enqueue_errors_total",
			Help: "Async JetStream publish enqueue failures",
		},
		[]string{"stream"},
	)
)

var (
	corePublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nats_core_publish_duration_seconds",
			Help:    "Core NATS publish latency",
			Buckets: localBuckets,
		},
		[]string{"stream"},
	)

	corePublishErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_core_publish_errors_total",
			Help: "Core NATS publish failures",
		},
		[]string{"stream"},
	)

	coreRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nats_core_request_duration_seconds",
			Help:    "Core NATS request-reply round-trip latency",
			Buckets: networkBuckets,
		},
		[]string{"stream"},
	)

	coreRequestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_core_request_errors_total",
			Help: "Core NATS request-reply failures",
		},
		[]string{"stream"},
	)
)
