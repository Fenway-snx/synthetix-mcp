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
	// Default maximum positive authorization entries.
	DefaultAuthCacheMaxEntries = 50000

	// Caps refusals separately so noisy misses cannot evict valid entries.
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

// Lowercases only when uppercase ASCII is present.
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
// Positive and refusal results use separate caps and TTLs.
// Tombstones block stale in-flight verifier results after revocation.
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

// Creates an authorization cache with bounded positive and refusal entries.
// A zero or negative cap uses the package default.
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

// Checks for a positive authorization result.
// Expired entries are lazily removed; tombstones always miss.
// Wallet addresses are normalized to lowercase internally.
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

// Adds or updates a positive authorization entry.
// Active tombstones skip writes to block stale in-flight verifier results.
// Wallet addresses are normalized to lowercase internally.
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

// Replaces a positive entry with a short-lived tombstone.
// Wallet addresses are normalized to lowercase internally.
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

// Removes all positive, tombstone, and refusal entries.
func (c *AuthCache) clearAll() {
	c.mu.Lock()
	clear(c.entries)
	c.cachedCount = 0
	clear(c.negativeEntries)
	c.negativeCount = 0
	c.mu.Unlock()
}

// Switches to short-TTL mode and flushes existing entries.
// This bounds staleness when invalidation is unavailable.
func (c *AuthCache) Degrade() {
	c.mu.Lock()
	c.degraded = true
	clear(c.entries)
	c.cachedCount = 0
	clear(c.negativeEntries)
	c.negativeCount = 0
	c.mu.Unlock()
}

// Restores normal TTLs and flushes degraded-mode entries.
func (c *AuthCache) Restore() {
	c.mu.Lock()
	c.degraded = false
	clear(c.entries)
	c.cachedCount = 0
	clear(c.negativeEntries)
	c.negativeCount = 0
	c.mu.Unlock()
}

// Returns the current number of positive entries.
func (c *AuthCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cachedCount
}

// Returns the active positive-entry TTL.
// Caller must hold c.mu (read or write).
func (c *AuthCache) activeTTLLocked() time.Duration {
	if c.degraded {
		return c.degradedTTL
	}
	return c.entryTTL
}

// Deletes an expired positive entry after rechecking under write lock.
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

// Deletes an expired refusal entry after rechecking under write lock.
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

// Removes an arbitrary positive entry; map iteration is randomized.
// Revocation tombstones stay in place.
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

// Checks for a recent authorization refusal.
// Expired entries are lazily removed.
// Wallet addresses are normalized to lowercase internally.
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

// Stores a definitive authorization refusal.
// Do not call for transient verifier errors.
// Wallet addresses are normalized to lowercase internally.
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

// Returns the current number of refusal entries.
func (c *AuthCache) negativeLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.negativeCount
}

// Removes an arbitrary refusal entry.
// Caller must hold c.mu write lock.
func (c *AuthCache) evictRandomNegativeLocked() {
	for key := range c.negativeEntries {
		delete(c.negativeEntries, key)
		c.negativeCount--
		return
	}
}
