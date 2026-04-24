package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	snx_lib_api_ratelimiting "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/rate_limiting"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

// In-process fixed-window rate limiter for the standalone image.
// Matches the semantics of the previous Redis-backed limiter so
// tuning values carry over.
type RateLimiter struct {
	handlerTokenCosts   snx_lib_api_ratelimiting.HandlerTokenCosts
	ipHandlerCosts      snx_lib_api_ratelimiting.HandlerTokenCosts
	ipLimit             int64
	ipWindow            time.Duration
	ipSpecificLimits    snx_lib_api_ratelimiting.PerIPRateLimits
	logger              snx_lib_logging.Logger
	subaccountLimit     int64
	subaccountWindow    time.Duration
	subaccountSpecific  snx_lib_api_ratelimiting.PerSubAccountRateLimits

	mu       sync.Mutex
	counters map[string]*fixedWindowCounter
}

type fixedWindowCounter struct {
	windowStart time.Time
	used        int64
}

func NewRateLimiter(
	_ context.Context,
	logger snx_lib_logging.Logger,
	cfg *config.Config,
) (*RateLimiter, error) {
	if cfg == nil {
		return &RateLimiter{logger: logger, counters: map[string]*fixedWindowCounter{}}, nil
	}

	return &RateLimiter{
		handlerTokenCosts:  cfg.HandlerTokenCosts,
		ipHandlerCosts:     cfg.IPHandlerTokenCosts,
		ipLimit:            int64(cfg.IPRateLimiterConfig.IPRateLimit),
		ipWindow:           time.Duration(cfg.IPRateLimiterConfig.WindowMs) * time.Millisecond,
		ipSpecificLimits:   nil,
		logger:             logger,
		subaccountLimit:    int64(cfg.OrderRateLimiterConfig.GeneralRateLimit),
		subaccountWindow:   time.Duration(cfg.OrderRateLimiterConfig.WindowMs) * time.Millisecond,
		subaccountSpecific: cfg.OrderRateLimiterConfig.SpecificRateLimits,
		counters:           map[string]*fixedWindowCounter{},
	}, nil
}

func (r *RateLimiter) Check(
	ctx context.Context,
	operationName string,
	batchSize int,
	state *session.State,
) error {
	if r == nil {
		return nil
	}

	clientIP := clientIPFromContext(ctx)
	if clientIP != "" && r.ipLimit > 0 {
		ipCost := int64(r.ipHandlerCosts.LookupIPTokenCost(
			snx_lib_api_ratelimiting.RequestAction(operationName),
			batchSize,
		))
		allowed, limit := r.consume("ip:"+clientIP, ipCost, r.ipLimit, r.ipWindow)
		if !allowed {
			return &rateLimitExceededError{
				appliedLimit: int(limit),
				scope:        "ip",
				toolName:     operationName,
			}
		}
	}

	if state == nil || state.AuthMode != session.AuthModeAuthenticated || state.SubAccountID <= 0 || r.subaccountLimit <= 0 {
		return nil
	}

	subCost := int64(r.handlerTokenCosts.LookupTokenCost(
		snx_lib_api_ratelimiting.RequestAction(operationName),
		batchSize,
	))
	key := fmt.Sprintf("sa:%d", state.SubAccountID)
	allowed, limit := r.consume(key, subCost, r.subaccountLimit, r.subaccountWindow)
	if !allowed {
		return &rateLimitExceededError{
			appliedLimit: int(limit),
			scope:        "subaccount",
			toolName:     operationName,
		}
	}
	return nil
}

// Reserves cost tokens for key; returns (allowed, effectiveLimit).
// Resets the bucket when the current window has fully elapsed.
// cost <= 0 is always allowed.
func (r *RateLimiter) consume(key string, cost int64, limit int64, window time.Duration) (bool, int64) {
	if cost <= 0 {
		return true, limit
	}
	if limit <= 0 || window <= 0 {
		return true, limit
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	bucket, ok := r.counters[key]
	if !ok || now.Sub(bucket.windowStart) >= window {
		bucket = &fixedWindowCounter{windowStart: now}
		r.counters[key] = bucket
	}
	if bucket.used+cost > limit {
		return false, limit
	}
	bucket.used += cost
	return true, limit
}
