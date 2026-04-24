// Definition of `RateLimiter` and supporting types, which provide
// Rate-limiting for REST and WS API services.
//
// NOTE: the ordering of this file is as follows:
//
// - package;
// - imports;
// - constants;
// - supporting structures and types;
// - main `RateLimiter` structure;
// - (private) initialisation methods;
// - (public) API methods;
// - (private) implementation methods;
//
// Within each section, lexiographical order is required.

package ratelimiting

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	cleanupInterval = 5 * time.Minute  // Period of background task that is used to clean up idle rate limit records.
	idleTimeout     = 10 * time.Minute // Duration of how long a rate limit record can be idle before being removed.
)

// Lua script that atomically implements a token bucket in Redis.
//
// KEYS[1] = bucket key (e.g. "ratelimit:order:42")
// ARGV[1] = max_tokens   (bucket capacity / burst size)
// ARGV[2] = refill_rate   (tokens added per second)
// ARGV[3] = now_ms        (current unix time in milliseconds)
// ARGV[4] = requested     (tokens to consume)
//
// Returns {allowed (0|1), floor(remaining_tokens)}.
const tokenBucketScript = `
local key = KEYS[1]
local max_tokens = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

if tokens == nil or last_refill == nil then
    tokens = max_tokens
    last_refill = now_ms
end

local elapsed = math.max(0, (now_ms - last_refill) / 1000.0)
tokens = math.min(max_tokens, tokens + elapsed * refill_rate)

local allowed = 0
if tokens >= requested then
    tokens = tokens - requested
    allowed = 1
end

redis.call('HMSET', key, 'tokens', tostring(tokens), 'last_refill', tostring(now_ms))
redis.call('PEXPIRE', key, 600000)

return {allowed, math.floor(tokens)}
`

// Types that holds the general or key-specific rate limit to be
// applicable over the given rate-limiter's duration.
type RateLimit int64

// The underlying standard rate-limiting type.
type rateLimiter = rate.Limiter

// Wraps a rate limiter with its last access time for cleanup.
type limiterEntry struct {
	limiter  *rateLimiter
	lastSeen atomic.Int64 // Unix epoch milliseconds
}

func (le *limiterEntry) LastSeen() time.Time {
	lastSeen := le.lastSeen.Load()

	return time.UnixMilli(lastSeen)
}

func (le *limiterEntry) SetLastSeen(eventTime time.Time) {
	le.lastSeen.Store(eventTime.UnixMilli())
}

// Wraps the limiters collection.
type limiters[K ~string | ~int64 | ~uint64, M map[K]RateLimit] struct {
	limiters sync.Map // map[K]*limiterEntry - per K limiters - this is used because allows concurrent Range/Delete
}

func (l *limiters[K, M]) cleanup(
	threshold time.Time,
) {

	l.limiters.Range(func(key, value any) bool {
		le := value.(*limiterEntry)

		lastSeen := le.LastSeen()

		// Remove if idle for longer than threshold
		if lastSeen.Before(threshold) {
			l.limiters.Delete(key)
		}

		return true
	})
}

// Returns or creates a rate limiter for the given K.
func (l *limiters[K, M]) lookupOrCreateLimiter(
	k K,
	window time.Duration,
	rateLimit RateLimit,
	eventTime time.Time,
) *rateLimiter {

	if entry, ok := l.limiters.Load(k); ok {
		le := entry.(*limiterEntry)

		le.SetLastSeen(eventTime)

		return le.limiter
	} else {
		ratePerSecond := rate.Limit(float64(rateLimit) / window.Seconds())

		// Create new limiter with burst equal to limit
		// This allows for bursts up to the limit, then refills at the calculated rate
		limiter := rate.NewLimiter(ratePerSecond, int(rateLimit))

		le := &limiterEntry{
			limiter: limiter,
		}
		le.SetLastSeen(eventTime)

		actual, loaded := l.limiters.LoadOrStore(k, le)
		if loaded {
			return actual.(*limiterEntry).limiter
		}

		return limiter
	}
}

// Returns or creates a rate limiter for the given K.
func (l *limiters[K, M]) removeLimiterIfExists(
	k K,
) {
	l.limiters.Delete(k)
}

type rateLimits[K ~string | ~int64 | ~uint64, M map[K]RateLimit] struct {
	rwmu               sync.RWMutex
	generalRateLimit   RateLimit
	specificRateLimits M // Store of rate-limits for specific Ks

	// NOTE: a later version might take a map of functions to facilitate dynamic changes of rates without requiring system restart
}

// Adds or updates the specific rate-limit for the given key, returning
func (r *rateLimits[K, M]) addSpecificRateLimit(
	k K,
	newRateLimit RateLimit,
) (
	previous RateLimit, // This is an arbitrary value if `!exists`
	exists bool, // Indicates whether a specific rate limit was set for K
) {
	r.rwmu.Lock()
	defer r.rwmu.Unlock()

	previous, exists = r.specificRateLimits[k]

	r.specificRateLimits[k] = newRateLimit

	return
}

func (r *rateLimits[K, M]) getSpecificRateLimit(
	k K,
) (
	previous RateLimit, // This is an arbitrary value if `!exists`
	exists bool, // Indicates whether a specific rate limit is set for K
) {
	r.rwmu.RLock()
	defer r.rwmu.RUnlock()

	previous, exists = r.specificRateLimits[k]

	return
}

// Obtains the K-specific rate limit, if specified, or the
// general rate limit otherwise;
func (r *rateLimits[K, M]) lookupRateLimitForKey(k K) RateLimit {
	r.rwmu.RLock()
	defer r.rwmu.RUnlock()

	if specificRateLimit, exists := r.specificRateLimits[k]; exists {
		return specificRateLimit
	} else {
		return r.generalRateLimit
	}
}

func (r *rateLimits[K, M]) removeSpecificRateLimit(
	k K,
) (
	previous RateLimit, // This is an arbitrary value if `!exists`
	exists bool, // Indicates whether a specific rate limit was set for K
) {
	r.rwmu.Lock()
	defer r.rwmu.Unlock()

	previous, exists = r.specificRateLimits[k]

	delete(r.specificRateLimits, k)

	return
}

// A general purpose rate limiter.
//
// Note:
// Sadly, due to the limitations of Go's generics, we have to specify both
// key type K and the type M of the map itself.
type RateLimiter[K ~string | ~int64 | ~uint64, M map[K]RateLimit] struct {
	ctx        context.Context
	limiters   limiters[K, M] // in-memory limiters (used when redisClient is nil)
	window     time.Duration
	rateLimits rateLimits[K, M]

	redisClient *snx_lib_db_redis.SnxClient // when non-nil, token bucket state is stored in Redis
	keyPrefix   string                      // Redis key prefix, e.g. "ratelimit:order"
}

// Attempts to create an order rate limiter instance, based on the given
// parameters.
//
// Parameters:
//   - ctx - standard execution context, for cancellation of subtasks;
//   - rc - optional Redis client. When non-nil the token bucket state is
//     stored in Redis (shared across service instances). When nil, an
//     in-memory token bucket is used instead;
//   - keyPrefix - Redis key prefix (e.g. "ratelimit:order"). Ignored when
//     rc is nil;
//   - window - a positive time duration that specifies over what period the
//     limit algorithm will measure;
//   - generalRateLimit - the rate limit that will be applied in the general
//     case. May not be negative. If zero, no limit will be applied in the
//     general case;
//   - specificRateLimits - a mapping containing specific rate limit for
//     given Ks. None may be negative. When zero, no limit will be applied
//     to that K;
func NewRateLimiter[K ~string | ~int64 | ~uint64, M map[K]RateLimit](
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	keyPrefix string,
	window time.Duration,
	generalRateLimit RateLimit,
	specificRateLimits M,
) (
	r *RateLimiter[K, M],
	err error,
) {

	if window <= 0 {
		return nil, errRateLimitDurationMustBePositive
	}

	if generalRateLimit < 0 {
		return nil, errRateLimitsMayNotBeNagative
	}

	for _, v := range specificRateLimits {
		if v < 0 {
			return nil, errRateLimitsMayNotBeNagative
		}
	}

	if specificRateLimits == nil {
		specificRateLimits = make(M)
	}

	// Fallback: if a Redis client was provided but its underlying connection
	// is not valid (e.g. nil ClusterClient in tests), treat it as absent so
	// the rate limiter falls back to the in-memory token bucket instead of
	// panicking on a nil dereference.
	if rc != nil && !rc.IsValid() {
		rc = nil
	}

	r = &RateLimiter[K, M]{
		ctx:    ctx,
		window: window,
		rateLimits: rateLimits[K, M]{
			generalRateLimit:   generalRateLimit,
			specificRateLimits: specificRateLimits,
		},
		redisClient: rc,
		keyPrefix:   keyPrefix,
	}

	// When Redis is used, TTL-based expiry handles cleanup;
	// otherwise start the in-memory token bucket as a backup if the Redis
	// token bucket is no longer valid.
	if rc == nil {
		go r.cleanupLoop(ctx)
	}

	return
}

// Checks if a K has exceeded its order rate limit.
//
// When Redis backs the limiter, a non-nil error means the check could not be
// completed. Callers may log and fail open (allow the request) or fail closed;
// REST and WebSocket handlers currently fail open on Redis errors.
//
// Returns:
// - allowed boolean;
// - (approximate) remaining count of available tokens;
// - the rate limit applied for k;
// - error;
func (r *RateLimiter[K, M]) CheckOrderLimit(
	ctx context.Context,
	k K,
	orderCount int,
) (
	bool, // allowed
	int, // availableTokens (approximate available tokens)
	RateLimit, // limit
	error,
) {
	rateLimit := r.rateLimits.lookupRateLimitForKey(k)

	if rateLimit == 0 {
		return true, 0, 0, nil
	}

	if r.redisClient != nil {
		return r.checkOrderLimitRedis(ctx, k, orderCount, rateLimit)
	}

	return r.checkOrderLimitInMemory(k, orderCount, rateLimit)
}

// checkOrderLimitRedis executes the token bucket Lua script against Redis.
func (r *RateLimiter[K, M]) checkOrderLimitRedis(
	ctx context.Context,
	k K,
	orderCount int,
	rateLimit RateLimit,
) (
	bool, // allowed
	int, // availableTokens
	RateLimit, // limit
	error,
) {
	key := fmt.Sprintf("%s:%v", r.keyPrefix, k)
	maxTokens := int64(rateLimit)
	refillRate := float64(rateLimit) / r.window.Seconds()
	nowMs := snx_lib_utils_time.Now().UnixMilli()

	result, err := r.redisClient.Eval(
		ctx,
		tokenBucketScript,
		[]string{key},
		maxTokens,
		refillRate,
		nowMs,
		orderCount,
	).Int64Slice()
	if err != nil {
		return false, 0, rateLimit, fmt.Errorf("redis rate limit eval failed: %w", err)
	}

	allowed := result[0] == 1
	availableTokens := int(result[1])

	return allowed, availableTokens, rateLimit, nil
}

// checkOrderLimitInMemory uses the local golang.org/x/time/rate limiter.
func (r *RateLimiter[K, M]) checkOrderLimitInMemory(
	k K,
	orderCount int,
	rateLimit RateLimit,
) (
	bool, // allowed
	int, // availableTokens
	RateLimit, // limit
	error,
) {
	now := snx_lib_utils_time.Now()

	limiter := r.limiters.lookupOrCreateLimiter(k, r.window, rateLimit, now)

	// Check if we can allow this many orders
	allowed := limiter.AllowN(snx_lib_utils_time.Now(), orderCount)

	// Get approximate count of available tokens
	availableTokens := int(limiter.Tokens())

	return allowed, availableTokens, rateLimit, nil
}

// Adds or updates the specific rate-limit for the given key, returning
func (r *RateLimiter[K, M]) AddSpecificRateLimit(
	k K,
	newRateLimit RateLimit,
) (
	previous RateLimit, // This is an arbitrary value if `!exists`
	exists bool, // Indicates whether a specific rate limit was set for K
	err error,
) {
	if newRateLimit < 0 {
		err = errRateLimitsMayNotBeNagative

		return
	}

	previous, exists = r.rateLimits.addSpecificRateLimit(k, newRateLimit)

	// Remove the existing bucket so it gets re-created with the new limit.
	r.limiters.removeLimiterIfExists(k)
	r.deleteRedisBucket(k)

	return
}

func (r *RateLimiter[K, M]) GetSpecificRateLimit(
	k K,
) (
	previous RateLimit, // This is an arbitrary value if `!exists`
	exists bool, // Indicates whether a specific rate limit is set for K
) {
	previous, exists = r.rateLimits.getSpecificRateLimit(k)

	return
}

func (r *RateLimiter[K, M]) RemoveSpecificRateLimit(
	k K,
) (
	previous RateLimit, // This is an arbitrary value if `!exists`
	exists bool, // Indicates whether a specific rate limit was set for K
) {
	previous, exists = r.rateLimits.removeSpecificRateLimit(k)

	// Remove the existing bucket so it gets re-created with the general limit.
	r.limiters.removeLimiterIfExists(k)
	r.deleteRedisBucket(k)

	return
}

// The window.
func (r *RateLimiter[K, M]) Window() time.Duration {
	return r.window
}

// runs periodically to remove idle rate limiters
func (r *RateLimiter[K, M]) cleanupLoop(
	ctx context.Context,
) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

outerLoop:
	for {
		select {
		case <-ctx.Done():
			break outerLoop
		case <-ticker.C:
			r.cleanup()
		}
	}
}

// removes rate limiters that haven't been used in IdleTimeout
func (r *RateLimiter[K, M]) cleanup() {
	now := snx_lib_utils_time.Now()
	threshold := now.Add(-idleTimeout)

	r.limiters.cleanup(threshold)
}

// deleteRedisBucket removes a token bucket key from Redis.
// This is a best-effort operation; errors are silently ignored because
// the bucket will simply be re-created on the next CheckOrderLimit call.
func (r *RateLimiter[K, M]) deleteRedisBucket(k K) {
	if r.redisClient == nil {
		return
	}

	key := fmt.Sprintf("%s:%v", r.keyPrefix, k)
	r.redisClient.Del(r.ctx, key)
}
