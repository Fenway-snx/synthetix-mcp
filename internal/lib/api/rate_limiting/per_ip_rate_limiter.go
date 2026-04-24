package ratelimiting

import (
	"context"
	"time"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
)

const perIPKeyPrefix = "ratelimit:ip"

type PerIPRateLimits = map[string]RateLimit

// Per IP-address rate limiter.
type PerIPRateLimiter = RateLimiter[string, PerIPRateLimits]

func NewPerIPRateLimiter(
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	window time.Duration,
	generalRateLimit RateLimit,
	specificRateLimits PerIPRateLimits,
) (
	r *PerIPRateLimiter,
	err error,
) {
	return NewRateLimiter[string](
		ctx,
		rc,
		perIPKeyPrefix,
		window,
		generalRateLimit,
		specificRateLimits,
	)
}

func NewPerIPRateLimiterFromConfig(
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	cfg *PerIPRateLimiterConfig,
) (
	r *PerIPRateLimiter,
	err error,
) {
	var window time.Duration
	var ipRateLimit RateLimit

	if cfg != nil {

		window = time.Duration(cfg.WindowMs) * time.Millisecond
		ipRateLimit = cfg.IPRateLimit
	} else {

		window = defaultWindow
		ipRateLimit = defaultIPRateLimit
	}

	return NewPerIPRateLimiter(
		ctx,
		rc,
		window,
		ipRateLimit,
		nil,
	)
}
