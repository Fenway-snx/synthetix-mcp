package ratelimiting

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func Test_ObserveRateLimitCheck_ALLOWED_INCREMENTS_COUNTER(t *testing.T) {
	service, limiter, action := "rest", "ip", "placeOrders"
	before := testutil.ToFloat64(
		rateLimitChecksTotal.WithLabelValues(service, limiter, action, resultAllowed),
	)

	ObserveRateLimitCheck(service, limiter, action, true, 80, 100)

	after := testutil.ToFloat64(
		rateLimitChecksTotal.WithLabelValues(service, limiter, action, resultAllowed),
	)
	assert.Equal(t, before+1, after)
}

func Test_ObserveRateLimitCheck_REJECTED_INCREMENTS_COUNTER(t *testing.T) {
	service, limiter, action := "ws", "subaccount", "cancelOrders"
	before := testutil.ToFloat64(
		rateLimitChecksTotal.WithLabelValues(service, limiter, action, resultRejected),
	)

	ObserveRateLimitCheck(service, limiter, action, false, 0, 100)

	after := testutil.ToFloat64(
		rateLimitChecksTotal.WithLabelValues(service, limiter, action, resultRejected),
	)
	assert.Equal(t, before+1, after)
}

func Test_ObserveRateLimitCheck_ALLOWED_RECORDS_HISTOGRAM(t *testing.T) {
	service, limiter, action := "rest", "subaccount", "histogramAllowed"
	countBefore := histogramSampleCount(service, limiter, action)

	ObserveRateLimitCheck(service, limiter, action, true, 75, 100)

	countAfter := histogramSampleCount(service, limiter, action)
	assert.Equal(t, countBefore+1, countAfter)
}

func Test_ObserveRateLimitCheck_REJECTED_DOES_NOT_RECORD_HISTOGRAM(t *testing.T) {
	service, limiter, action := "ws", "ip", "histogramRejected"
	countBefore := histogramSampleCount(service, limiter, action)

	ObserveRateLimitCheck(service, limiter, action, false, 0, 100)

	countAfter := histogramSampleCount(service, limiter, action)
	assert.Equal(t, countBefore, countAfter)
}

func Test_ObserveRateLimitCheck_ZERO_LIMIT_SKIPS_HISTOGRAM(t *testing.T) {
	service, limiter, action := "rest", "ip", "histogramZeroLimit"
	countBefore := histogramSampleCount(service, limiter, action)

	ObserveRateLimitCheck(service, limiter, action, true, 0, 0)

	countAfter := histogramSampleCount(service, limiter, action)
	assert.Equal(t, countBefore, countAfter)
}

func Test_ObserveRateLimitCheck_BOTH_RESULT_LABELS(t *testing.T) {
	service, limiter, action := "rest", "subaccount", "bothResults"

	ObserveRateLimitCheck(service, limiter, action, true, 50, 100)
	ObserveRateLimitCheck(service, limiter, action, false, 0, 100)

	allowed := testutil.ToFloat64(
		rateLimitChecksTotal.WithLabelValues(service, limiter, action, resultAllowed),
	)
	rejected := testutil.ToFloat64(
		rateLimitChecksTotal.WithLabelValues(service, limiter, action, resultRejected),
	)
	assert.GreaterOrEqual(t, allowed, float64(1))
	assert.GreaterOrEqual(t, rejected, float64(1))
}

func histogramSampleCount(service, limiter, action string) uint64 {
	var m dto.Metric

	rateLimitBucketRemainingRatio.
		WithLabelValues(service, limiter, action).(prometheus.Metric).
		Write(&m)

	return m.GetHistogram().GetSampleCount()
}
