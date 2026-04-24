package ratelimiting

import "time"

const (
	defaultIPRateLimit    = 1_000           // Default per-IP token budget per second, chosen as max IP limits allowed for any IP connection.
	defaultOrderRateLimit = 100             // Default per-SID token budget per second, chosen as lowest common denominator for max 50 reads/s and 20 writes/s.
	defaultWindow         = 1 * time.Second // Default time window for rate limiting.

	// DefaultHandlerTokenCost is the fallback token cost for any handler action
	// not explicitly listed in the HandlerTokenCosts map.
	DefaultHandlerTokenCost = 1

	// DefaultIPHandlerTokenCost is the fallback token cost for any handler
	// action not explicitly listed in the IP HandlerTokenCosts map.
	DefaultIPHandlerTokenCost = 1
)
