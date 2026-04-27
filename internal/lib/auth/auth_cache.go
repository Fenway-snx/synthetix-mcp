package auth

import (
	"strings"
	"sync"
	"time"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	// DefaultAuthCacheMaxEntries is the default maximum number of entries in the auth cache.
	DefaultAuthCacheMaxEntries = 50000

	// DefaultNegativeCacheMaxEntries caps negative (refusal) entries independently
	// from the positive cache so that an attacker flooding unique wallet+subaccount
	// pairs cannot evict legitimate positive entries.
	DefaultNegativeCacheMaxEntries = 10000

	defaultDegradedTTL  = 30 * time.Second
	defaultEntryTTL     = 30 * time.Minute
	defaultNegativeTTL  = 30 * time.Second
	defaultTombstoneTTL = 10 * time.Second
)

type authCacheEntry struct {
	authType  AuthType
	cachedAt  time.Time
	revokedAt time.Time // zero = not a tombstone
}

type authCacheKey struct {
	subAccountId  snx_lib_core.SubAccountId
	walletAddress string // always lowercase
}

type refusedAtByAuthCacheKey map[authCacheKey]time.Time

// toLowerFast returns the lowercase version of s without allocating when s
// is already lowercase ASCII — the common case for Ethereum addresses after
// API-boundary normalization.
func toLowerFast(s string) string {
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			return strings.ToLower(s)
		}
	}
	return s
}

// Thread-safe in-memory cache for authorization results.
// Entries expire after entryTTL (default 30m) to keep the cache populated
// with active traders. When the max entries cap is reached, a random entry
// is evicted as a safety valve.
//
// Negative (refusal) results are cached separately with a short TTL
// (default 30s) to prevent auth-refusal DDoS: without negative caching,
// every request for an unauthorized wallet+subaccount pair falls through
// to the verifier, creating an amplification vector.
//
// When invalidation is unavailable, the cache enters "degraded" mode: entries
// use a much shorter TTL (default 30s), bounding staleness while still
// absorbing repeated lookups from the same trader. Restore() flushes stale
// entries and restores the normal TTL.
//
// Revocation tombstones are stored inline in the entries map (revokedAt > 0)
// to prevent a TOCTOU race where an in-flight verifier call returns a stale
// positive result after a revocation event has already been processed.
type AuthCache struct {
	mu           sync.RWMutex
	cachedCount  int // real (non-tombstone) positive entries
	degraded     bool
	entries      map[authCacheKey]*authCacheEntry
	degradedTTL  time.Duration
	entryTTL     time.Duration
	maxEntries   int
	tombstoneTTL time.Duration

	negativeEntries    refusedAtByAuthCacheKey
	negativeCount      int
	maxNegativeEntries int
	negativeTTL        time.Duration
}

// Creates a new authorization cache. maxEntries caps the number of positive
// entries to prevent unbounded memory growth; when the cap is reached a
// random entry is evicted. Entries expire after 30 minutes normally, or
// 30 seconds in degraded mode. A zero or negative maxEntries uses
// DefaultAuthCacheMaxEntries.
func NewAuthCache(maxEntries int) *AuthCache {
	if maxEntries <= 0 {
		maxEntries = DefaultAuthCacheMaxEntries
	}
	return &AuthCache{
		degradedTTL:        defaultDegradedTTL,
		entries:            make(map[authCacheKey]*authCacheEntry),
		entryTTL:           defaultEntryTTL,
		maxEntries:         maxEntries,
		negativeEntries:    make(refusedAtByAuthCacheKey),
		maxNegativeEntries: DefaultNegativeCacheMaxEntries,
		negativeTTL:        defaultNegativeTTL,
		tombstoneTTL:       defaultTombstoneTTL,
	}
}

// Checks the cache for an authorization result.
// Expired entries are lazily removed. The fast path (hit, not expired)
// uses only an RLock; the rare expiry path briefly takes a write lock.
// In degraded mode the shorter degradedTTL is used for expiry checks.
// Tombstone entries (revokedAt > 0) always return a miss.
// walletAddress is normalized to lowercase internally.
func (c *AuthCache) Lookup(
	walletAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
) (authType AuthType, found bool) {
	key := authCacheKey{
		subAccountId:  subAccountId,
		walletAddress: toLowerFast(string(walletAddress)),
	}

	c.mu.RLock()
	entry, ok := c.entries[key]
	ttl := c.activeTTLLocked()
	c.mu.RUnlock()

	if !ok || !entry.revokedAt.IsZero() {
		return AuthTypeNone, false
	}

	if snx_lib_utils_time.Since(entry.cachedAt) >= ttl {
		c.mu.Lock()
		c.deleteIfExpiredLocked(key)
		c.mu.Unlock()
		return AuthTypeNone, false
	}

	return entry.authType, true
}

// Adds or updates a cache entry. Only call with positive authorization
// results — failed lookups must never be cached.
// If the key has an active tombstone (from a recent Evict), the store is
// skipped to prevent caching a stale in-flight verifier result after revocation.
// walletAddress is normalized to lowercase internally.
func (c *AuthCache) Store(
	walletAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
	authType AuthType,
) {
	key := authCacheKey{
		subAccountId:  subAccountId,
		walletAddress: toLowerFast(string(walletAddress)),
	}

	now := snx_lib_utils_time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	existing, exists := c.entries[key]
	if exists && !existing.revokedAt.IsZero() {
		if now.Sub(existing.revokedAt) < c.tombstoneTTL {
			return
		}
	}

	isNew := !exists || !existing.revokedAt.IsZero()
	if isNew && c.cachedCount >= c.maxEntries {
		c.evictRandomLocked()
	}

	c.entries[key] = &authCacheEntry{
		authType: authType,
		cachedAt: now,
	}
	if isNew {
		c.cachedCount++
	}

	if _, hadNegative := c.negativeEntries[key]; hadNegative {
		delete(c.negativeEntries, key)
		c.negativeCount--
	}
}

// Replaces a cache entry with a tombstone to prevent re-caching from an
// in-flight verifier call.
// walletAddress is normalized to lowercase internally.
func (c *AuthCache) Evict(
	walletAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
) {
	key := authCacheKey{
		subAccountId:  subAccountId,
		walletAddress: toLowerFast(string(walletAddress)),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := snx_lib_utils_time.Now()
	existing, hadReal := c.entries[key]
	if hadReal && existing.revokedAt.IsZero() {
		c.cachedCount--
	}
	c.entries[key] = &authCacheEntry{revokedAt: now}
}

// clearAll removes all entries (including tombstones and negative entries).
func (c *AuthCache) clearAll() {
	c.mu.Lock()
	clear(c.entries)
	c.cachedCount = 0
	clear(c.negativeEntries)
	c.negativeCount = 0
	c.mu.Unlock()
}

// Switches the cache to degraded mode: entries use degradedTTL (default
// 30s) instead of the normal entryTTL (30m), and all current entries
// (including negative entries) are flushed. This bounds staleness when
// invalidation is unavailable while still absorbing repeated verifier calls
// from active traders.
func (c *AuthCache) Degrade() {
	c.mu.Lock()
	c.degraded = true
	clear(c.entries)
	c.cachedCount = 0
	clear(c.negativeEntries)
	c.negativeCount = 0
	c.mu.Unlock()
}

// Restores the cache to normal mode. All entries
// (including negative entries) cached during degraded mode are flushed so
// that the next lookup goes to the verifier.
func (c *AuthCache) Restore() {
	c.mu.Lock()
	c.degraded = false
	clear(c.entries)
	c.cachedCount = 0
	clear(c.negativeEntries)
	c.negativeCount = 0
	c.mu.Unlock()
}

// Returns the current number of cached (non-tombstone) entries.
func (c *AuthCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cachedCount
}

// activeTTLLocked returns degradedTTL when in degraded mode, entryTTL otherwise.
// Caller must hold c.mu (read or write).
func (c *AuthCache) activeTTLLocked() time.Duration {
	if c.degraded {
		return c.degradedTTL
	}
	return c.entryTTL
}

// deleteIfExpiredLocked re-checks an entry under write lock and deletes it
// if it is still a real (non-tombstone) entry whose age exceeds the active TTL.
// Caller must hold c.mu write lock.
func (c *AuthCache) deleteIfExpiredLocked(key authCacheKey) {
	e, exists := c.entries[key]
	if !exists || !e.revokedAt.IsZero() {
		return
	}
	if snx_lib_utils_time.Since(e.cachedAt) >= c.activeTTLLocked() {
		delete(c.entries, key)
		c.cachedCount--
	}
}

// deleteNegativeIfExpiredLocked re-checks a negative entry under write lock
// and deletes it if its age exceeds the negative TTL.
// Caller must hold c.mu write lock.
func (c *AuthCache) deleteNegativeIfExpiredLocked(key authCacheKey) {
	refusedAt, exists := c.negativeEntries[key]
	if !exists {
		return
	}
	if snx_lib_utils_time.Since(refusedAt) >= c.negativeTTL {
		delete(c.negativeEntries, key)
		c.negativeCount--
	}
}

// evictRandomLocked removes an arbitrary non-tombstone entry from the cache.
// Go's map iteration order is randomized, giving us O(1) amortized eviction
// with near-identical hit rates to LRU at large cache sizes. Tombstone
// entries are skipped since they serve as short-lived revocation guards.
// Caller must hold c.mu write lock.
func (c *AuthCache) evictRandomLocked() {
	for key, entry := range c.entries {
		if entry.revokedAt.IsZero() {
			delete(c.entries, key)
			c.cachedCount--
			return
		}
	}
}

// Checks the negative cache for a recent authorization refusal.
// Returns true if the wallet+subaccount pair was recently refused, meaning
// the caller can reject without hitting the verifier.
// Expired entries are lazily removed. The fast path (hit, not expired)
// uses only an RLock; the rare expiry path briefly takes a write lock.
// walletAddress is normalized to lowercase internally.
func (c *AuthCache) LookupRefusal(
	walletAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
) bool {
	key := authCacheKey{
		subAccountId:  subAccountId,
		walletAddress: toLowerFast(string(walletAddress)),
	}

	c.mu.RLock()
	refusedAt, ok := c.negativeEntries[key]
	c.mu.RUnlock()

	if !ok {
		return false
	}

	if snx_lib_utils_time.Since(refusedAt) >= c.negativeTTL {
		c.mu.Lock()
		c.deleteNegativeIfExpiredLocked(key)
		c.mu.Unlock()
		return false
	}

	return true
}

// Stores a negative (refusal) cache entry. Call when the origin confirms
// that a wallet has no authorization for a subaccount. Do NOT call on
// transient verifier errors, only on definitive refusals.
// walletAddress is normalized to lowercase internally.
func (c *AuthCache) StoreRefusal(
	walletAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
) {
	key := authCacheKey{
		subAccountId:  subAccountId,
		walletAddress: toLowerFast(string(walletAddress)),
	}

	now := snx_lib_utils_time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.negativeEntries[key]
	if !exists && c.negativeCount >= c.maxNegativeEntries {
		c.evictRandomNegativeLocked()
	}

	c.negativeEntries[key] = now
	if !exists {
		c.negativeCount++
	}
}

// Returns the current number of negative (refusal) cache entries.
func (c *AuthCache) negativeLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.negativeCount
}

// evictRandomNegativeLocked removes an arbitrary negative entry.
// Caller must hold c.mu write lock.
func (c *AuthCache) evictRandomNegativeLocked() {
	for key := range c.negativeEntries {
		delete(c.negativeEntries, key)
		c.negativeCount--
		return
	}
}
