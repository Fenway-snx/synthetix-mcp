package ratelimiting

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	resultAllowed  = "allowed"
	resultRejected = "rejected"
)

var ratioHistogramBuckets = []float64{
	0, 0.05, 0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 1.0,
}

var (
	rateLimitChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limit_checks_total",
			Help: "Total number of rate-limit checks partitioned by outcome",
		},
		[]string{"service", "limiter", "action", "result"},
	)

	rateLimitBucketRemainingRatio = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_rate_limit_bucket_remaining_ratio",
			Help:    "Ratio of remaining tokens to bucket capacity observed on allowed checks",
			Buckets: ratioHistogramBuckets,
		},
		[]string{"service", "limiter", "action"},
	)
)

// Observes a rate-limit check result. When allowed and limit > 0,
// also records the remaining-ratio histogram.
func ObserveRateLimitCheck(
	service string,
	limiter string,
	action string,
	allowed bool,
	availableTokens int,
	limit RateLimit,
) {
	result := resultRejected
	if allowed {
		result = resultAllowed
	}

	rateLimitChecksTotal.
		WithLabelValues(service, limiter, action, result).
		Inc()

	if allowed && limit > 0 {
		ratio := float64(availableTokens) / float64(limit)

		rateLimitBucketRemainingRatio.
			WithLabelValues(service, limiter, action).
			Observe(ratio)
	}
}
