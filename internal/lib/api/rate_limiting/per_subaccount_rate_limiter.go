package ratelimiting

import (
	"context"
	"time"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
)

const perSubAccountKeyPrefix = "ratelimit:order"

type PerSubAccountRateLimits = map[snx_lib_core.SubAccountId]RateLimit

// Per sub-account rate limiter.
type PerSubAccountRateLimiter = RateLimiter[SubAccountId, PerSubAccountRateLimits]

func NewPerSubAccountRateLimiter(
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	window time.Duration,
	generalRateLimit RateLimit,
	specificRateLimits PerSubAccountRateLimits,
) (
	r *PerSubAccountRateLimiter,
	err error,
) {
	return NewRateLimiter[SubAccountId](
		ctx,
		rc,
		perSubAccountKeyPrefix,
		window,
		generalRateLimit,
		specificRateLimits,
	)
}

func NewPerSubAccountRateLimiterFromConfig(
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	cfg *PerSubAccountRateLimiterConfig,
) (
	r *PerSubAccountRateLimiter,
	err error,
) {
	var window time.Duration
	var generalRateLimit RateLimit
	var specificRateLimits PerSubAccountRateLimits

	if cfg != nil {

		window = time.Duration(cfg.WindowMs) * time.Millisecond
		generalRateLimit = RateLimit(cfg.GeneralRateLimit)
		specificRateLimits = cfg.SpecificRateLimits
	} else {

		window = defaultWindow
		generalRateLimit = defaultOrderRateLimit
		specificRateLimits = nil
	}

	return NewPerSubAccountRateLimiter(
		ctx,
		rc,
		window,
		generalRateLimit,
		specificRateLimits,
	)
}
