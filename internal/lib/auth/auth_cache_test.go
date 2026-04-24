package auth

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

func Test_AuthCache_LookupMiss(t *testing.T) {
	cache := NewAuthCache(100)

	authType, found := cache.Lookup("0xABCD1234", 1)

	assert.False(t, found)
	assert.Equal(t, AuthTypeNone, authType)
}

func Test_AuthCache_StoreAndLookup(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0x1234567890abcdef1234567890abcdef12345678")

	cache.Store(wallet, 42, AuthTypeOwner)

	authType, found := cache.Lookup(wallet, 42)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)
}

func Test_AuthCache_StoreDelegate(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0xdelegateaddress000000000000000000000000")

	cache.Store(wallet, 99, AuthTypeDelegate)

	authType, found := cache.Lookup(wallet, 99)
	assert.True(t, found)
	assert.Equal(t, AuthTypeDelegate, authType)
}

func Test_AuthCache_CaseInsensitiveWallet(t *testing.T) {
	cache := NewAuthCache(100)

	cache.Store("0xAbCdEf1234567890AbCdEf1234567890AbCdEf12", 1, AuthTypeOwner)

	// Lookup with different casing should still hit
	authType, found := cache.Lookup("0xabcdef1234567890abcdef1234567890abcdef12", 1)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)

	authType, found = cache.Lookup("0xABCDEF1234567890ABCDEF1234567890ABCDEF12", 1)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)
}

func Test_AuthCache_DifferentSubAccountsAreSeparate(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0x1111111111111111111111111111111111111111")

	cache.Store(wallet, 1, AuthTypeOwner)
	cache.Store(wallet, 2, AuthTypeDelegate)

	authType, found := cache.Lookup(wallet, 1)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)

	authType, found = cache.Lookup(wallet, 2)
	assert.True(t, found)
	assert.Equal(t, AuthTypeDelegate, authType)

	_, found = cache.Lookup(wallet, 3)
	assert.False(t, found)
}

func Test_AuthCache_Evict(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0x2222222222222222222222222222222222222222")

	cache.Store(wallet, 10, AuthTypeDelegate)

	_, found := cache.Lookup(wallet, 10)
	require.True(t, found)

	cache.Evict(wallet, 10)

	_, found = cache.Lookup(wallet, 10)
	assert.False(t, found)
}

func Test_AuthCache_EvictNonexistent(t *testing.T) {
	cache := NewAuthCache(100)

	// Should not panic or error
	cache.Evict("0xnonexistent", 999)
	assert.Equal(t, 0, cache.Len())
}

func Test_AuthCache_EvictCaseInsensitive(t *testing.T) {
	cache := NewAuthCache(100)

	cache.Store("0xAbCd", 1, AuthTypeDelegate)
	cache.Evict("0xABCD", 1)

	_, found := cache.Lookup("0xabcd", 1)
	assert.False(t, found)
}

func Test_AuthCache_MaxEntries(t *testing.T) {
	cache := NewAuthCache(3)

	cache.Store("0xAAAA", 1, AuthTypeOwner)
	cache.Store("0xBBBB", 2, AuthTypeOwner)
	cache.Store("0xCCCC", 3, AuthTypeOwner)

	assert.Equal(t, 3, cache.Len())

	// Adding a 4th entry should evict a random entry
	cache.Store("0xDDDD", 4, AuthTypeOwner)

	assert.Equal(t, 3, cache.Len())

	// The new entry should be present
	_, found := cache.Lookup("0xDDDD", 4)
	assert.True(t, found)
}

func Test_AuthCache_MaxEntries_UpdateExistingDoesNotEvict(t *testing.T) {
	cache := NewAuthCache(3)

	cache.Store("0xAAAA", 1, AuthTypeOwner)
	cache.Store("0xBBBB", 2, AuthTypeOwner)
	cache.Store("0xCCCC", 3, AuthTypeOwner)

	// Updating an existing entry should not trigger eviction
	cache.Store("0xAAAA", 1, AuthTypeDelegate)

	assert.Equal(t, 3, cache.Len())

	// All entries should still be present
	authType, found := cache.Lookup("0xAAAA", 1)
	assert.True(t, found)
	assert.Equal(t, AuthTypeDelegate, authType)

	_, found = cache.Lookup("0xBBBB", 2)
	assert.True(t, found)

	_, found = cache.Lookup("0xCCCC", 3)
	assert.True(t, found)
}

func Test_AuthCache_Len(t *testing.T) {
	cache := NewAuthCache(100)
	assert.Equal(t, 0, cache.Len())

	cache.Store("0xAAAA", 1, AuthTypeOwner)
	assert.Equal(t, 1, cache.Len())

	cache.Store("0xBBBB", 2, AuthTypeOwner)
	assert.Equal(t, 2, cache.Len())

	cache.Evict("0xAAAA", 1)
	assert.Equal(t, 1, cache.Len())
}

func Test_AuthCache_Clear(t *testing.T) {
	cache := NewAuthCache(100)

	cache.Store("0xAAAA", 1, AuthTypeOwner)
	cache.Store("0xBBBB", 2, AuthTypeDelegate)
	cache.Store("0xCCCC", 3, AuthTypeOwner)
	require.Equal(t, 3, cache.Len())

	cache.clearAll()

	assert.Equal(t, 0, cache.Len())

	_, found := cache.Lookup("0xAAAA", 1)
	assert.False(t, found)

	_, found = cache.Lookup("0xBBBB", 2)
	assert.False(t, found)

	// Cache still works after clear — new entries can be stored
	cache.Store("0xDDDD", 4, AuthTypeOwner)
	authType, found := cache.Lookup("0xDDDD", 4)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)
}

func Test_AuthCache_TombstoneBlocksStore(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0x3333333333333333333333333333333333333333")

	cache.Store(wallet, 10, AuthTypeDelegate)
	_, found := cache.Lookup(wallet, 10)
	require.True(t, found)

	cache.Evict(wallet, 10)
	_, found = cache.Lookup(wallet, 10)
	require.False(t, found)

	// Store should be blocked by the tombstone
	cache.Store(wallet, 10, AuthTypeDelegate)
	_, found = cache.Lookup(wallet, 10)
	assert.False(t, found, "store should be blocked by tombstone")
	assert.Equal(t, 0, cache.Len())
}

func Test_AuthCache_TombstoneExpires(t *testing.T) {
	cache := NewAuthCache(100)
	cache.tombstoneTTL = 1 * time.Millisecond // very short TTL for testing
	wallet := snx_lib_api_types.WalletAddress("0x4444444444444444444444444444444444444444")

	cache.Evict(wallet, 20)

	// Wait for tombstone to expire
	time.Sleep(5 * time.Millisecond)

	cache.Store(wallet, 20, AuthTypeDelegate)
	_, found := cache.Lookup(wallet, 20)
	assert.True(t, found, "store should succeed after tombstone expires")
}

func Test_AuthCache_EvictWithoutPriorEntry_CreatesTombstone(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0x5555555555555555555555555555555555555555")

	// Evict a key that was never stored (the TOCTOU scenario:
	// NATS event arrives before the gRPC response is cached)
	cache.Evict(wallet, 30)

	// Now try to store — should be blocked by tombstone
	cache.Store(wallet, 30, AuthTypeDelegate)
	_, found := cache.Lookup(wallet, 30)
	assert.False(t, found, "store should be blocked by tombstone even if key was never cached")
}

func Test_AuthCache_Degrade_ClearsEntriesAndUsesShortTTL(t *testing.T) {
	cache := NewAuthCache(100)
	cache.degradedTTL = 1 * time.Millisecond
	wallet := snx_lib_api_types.WalletAddress("0x8888888888888888888888888888888888888888")

	cache.Store(wallet, 1, AuthTypeOwner)
	_, found := cache.Lookup(wallet, 1)
	require.True(t, found)

	cache.Degrade()

	// Existing entries should be cleared
	_, found = cache.Lookup(wallet, 1)
	assert.False(t, found, "entries should be cleared on degrade")

	// Store still works in degraded mode
	cache.Store(wallet, 1, AuthTypeOwner)
	_, found = cache.Lookup(wallet, 1)
	assert.True(t, found, "store should still work in degraded mode")

	// But entry expires quickly due to degradedTTL
	time.Sleep(5 * time.Millisecond)
	_, found = cache.Lookup(wallet, 1)
	assert.False(t, found, "entry should expire quickly in degraded mode")
}

func Test_AuthCache_Restore_ClearsAndResumesNormalTTL(t *testing.T) {
	cache := NewAuthCache(100)
	cache.degradedTTL = 1 * time.Millisecond
	cache.entryTTL = 1 * time.Hour
	wallet := snx_lib_api_types.WalletAddress("0x8888888888888888888888888888888888888888")

	cache.Degrade()

	cache.Store(wallet, 1, AuthTypeOwner)
	require.Equal(t, 1, cache.Len())

	cache.Restore()

	// Entries from degraded mode are flushed
	_, found := cache.Lookup(wallet, 1)
	assert.False(t, found, "entries should be cleared on restore")

	// New entries use normal TTL again
	cache.Store(wallet, 1, AuthTypeOwner)
	time.Sleep(5 * time.Millisecond)

	_, found = cache.Lookup(wallet, 1)
	assert.True(t, found, "entry should survive with normal TTL after restore")
}

func Test_AuthCache_Degrade_ConcurrentSafety(t *testing.T) {
	cache := NewAuthCache(10000)

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent stores
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			wallet := snx_lib_api_types.WalletAddress(fmt.Sprintf("0x%040x", i))
			cache.Store(wallet, snx_lib_core.SubAccountId(i), AuthTypeDelegate)
		}
	}()

	// Concurrent lookups
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			wallet := snx_lib_api_types.WalletAddress(fmt.Sprintf("0x%040x", i))
			cache.Lookup(wallet, snx_lib_core.SubAccountId(i))
		}
	}()

	// Degrade mid-flight
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Microsecond)
		cache.Degrade()
	}()

	// Restore mid-flight
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Microsecond)
		cache.Restore()
	}()

	wg.Wait()
}

func Test_AuthCache_EntryExpires(t *testing.T) {
	cache := NewAuthCache(100)
	cache.entryTTL = 1 * time.Millisecond
	wallet := snx_lib_api_types.WalletAddress("0x6666666666666666666666666666666666666666")

	cache.Store(wallet, 1, AuthTypeOwner)
	_, found := cache.Lookup(wallet, 1)
	require.True(t, found)

	time.Sleep(5 * time.Millisecond)

	_, found = cache.Lookup(wallet, 1)
	assert.False(t, found, "entry should have expired")
	assert.Equal(t, 0, cache.Len(), "expired entry should be removed")
}

func Test_AuthCache_EntryNotExpiredWithinTTL(t *testing.T) {
	cache := NewAuthCache(100)
	cache.entryTTL = 1 * time.Hour
	wallet := snx_lib_api_types.WalletAddress("0x7777777777777777777777777777777777777777")

	cache.Store(wallet, 1, AuthTypeOwner)

	authType, found := cache.Lookup(wallet, 1)
	assert.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)
}

func Test_AuthCache_StoreRefreshesTTL(t *testing.T) {
	cache := NewAuthCache(100)
	cache.entryTTL = 50 * time.Millisecond
	wallet := snx_lib_api_types.WalletAddress("0x8888888888888888888888888888888888888888")

	cache.Store(wallet, 1, AuthTypeOwner)
	time.Sleep(30 * time.Millisecond)

	// Re-store refreshes the cachedAt timestamp
	cache.Store(wallet, 1, AuthTypeOwner)
	time.Sleep(30 * time.Millisecond)

	// 60ms total, but only 30ms since last store — should still be valid
	_, found := cache.Lookup(wallet, 1)
	assert.True(t, found, "re-stored entry should have refreshed TTL")
}

func Test_ToLowerFast_MixedCase(t *testing.T) {
	result := toLowerFast("0xAbCdEf1234567890AbCdEf1234567890AbCdEf12")
	assert.Equal(t, "0xabcdef1234567890abcdef1234567890abcdef12", result)
}

func Test_ToLowerFast_Empty(t *testing.T) {
	result := toLowerFast("")
	assert.Equal(t, "", result)
}

func Test_ToLowerFast_AllUppercase(t *testing.T) {
	result := toLowerFast("ABCDEF")
	assert.Equal(t, "abcdef", result)
}

func Test_AuthCache_StoreOverwritesAuthType(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0x1111111111111111111111111111111111111111")

	cache.Store(wallet, 1, AuthTypeOwner)
	authType, found := cache.Lookup(wallet, 1)
	require.True(t, found)
	assert.Equal(t, AuthTypeOwner, authType)

	cache.Store(wallet, 1, AuthTypeDelegate)
	authType, found = cache.Lookup(wallet, 1)
	require.True(t, found)
	assert.Equal(t, AuthTypeDelegate, authType)

	assert.Equal(t, 1, cache.Len())
}

func Test_AuthCache_MaxEntries_EvictsOneEntry(t *testing.T) {
	cache := NewAuthCache(3)

	cache.Store("0xAAAA", 1, AuthTypeOwner)
	cache.Store("0xBBBB", 2, AuthTypeOwner)
	cache.Store("0xCCCC", 3, AuthTypeOwner)

	cache.Store("0xDDDD", 4, AuthTypeOwner)

	assert.Equal(t, 3, cache.Len(), "cache should remain at max capacity")

	_, found := cache.Lookup("0xDDDD", 4)
	assert.True(t, found, "new entry should be present")
}

func Test_AuthCache_TombstoneMixedExpiry(t *testing.T) {
	cache := NewAuthCache(100)
	cache.tombstoneTTL = 50 * time.Millisecond

	// Create an old tombstone
	cache.Evict("0xAAAA", 1)
	time.Sleep(60 * time.Millisecond) // let it expire

	// Create a fresh tombstone
	cache.Evict("0xBBBB", 2)

	// Expired tombstone should allow store
	cache.Store("0xAAAA", 1, AuthTypeDelegate)
	_, found := cache.Lookup("0xAAAA", 1)
	assert.True(t, found, "expired tombstone should not block store")

	// Active tombstone should block store
	cache.Store("0xBBBB", 2, AuthTypeDelegate)
	_, found = cache.Lookup("0xBBBB", 2)
	assert.False(t, found, "active tombstone should block store")
}

func Test_AuthCache_ConcurrentEvictAndStore(t *testing.T) {
	cache := NewAuthCache(10000)
	wallet := snx_lib_api_types.WalletAddress("0x9999999999999999999999999999999999999999")

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent stores
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			cache.Store(wallet, snx_lib_core.SubAccountId(i%10), AuthTypeDelegate)
		}
	}()

	// Concurrent evictions
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			cache.Evict(wallet, snx_lib_core.SubAccountId(i%10))
		}
	}()

	// Concurrent lookups
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			cache.Lookup(wallet, snx_lib_core.SubAccountId(i%10))
		}
	}()

	wg.Wait()
}

func Test_AuthCache_StoreRefusalAndLookupRefusal(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0xaaaa000000000000000000000000000000000000")

	assert.False(t, cache.LookupRefusal(wallet, 1), "empty cache should return false")

	cache.StoreRefusal(wallet, 1)
	assert.True(t, cache.LookupRefusal(wallet, 1))
	assert.Equal(t, 1, cache.negativeLen())
	assert.Equal(t, 0, cache.Len(), "negative entries should not affect positive Len()")
}

func Test_AuthCache_StoreRefusal_CaseInsensitive(t *testing.T) {
	cache := NewAuthCache(100)

	cache.StoreRefusal("0xAbCd", 1)
	assert.True(t, cache.LookupRefusal("0xabcd", 1))
	assert.True(t, cache.LookupRefusal("0xABCD", 1))
}

func Test_AuthCache_StoreRefusal_Expires(t *testing.T) {
	cache := NewAuthCache(100)
	cache.negativeTTL = 1 * time.Millisecond
	wallet := snx_lib_api_types.WalletAddress("0xbbbb000000000000000000000000000000000000")

	cache.StoreRefusal(wallet, 1)
	assert.True(t, cache.LookupRefusal(wallet, 1))

	time.Sleep(5 * time.Millisecond)

	assert.False(t, cache.LookupRefusal(wallet, 1), "expired negative entry should return false")
	assert.Equal(t, 0, cache.negativeLen(), "expired entry should be lazily removed")
}

func Test_AuthCache_StoreRefusal_MaxEntries(t *testing.T) {
	cache := NewAuthCache(100)
	cache.maxNegativeEntries = 3

	cache.StoreRefusal("0xAAAA", 1)
	cache.StoreRefusal("0xBBBB", 2)
	cache.StoreRefusal("0xCCCC", 3)
	assert.Equal(t, 3, cache.negativeLen())

	cache.StoreRefusal("0xDDDD", 4)
	assert.Equal(t, 3, cache.negativeLen(), "should evict one to stay at max")

	assert.True(t, cache.LookupRefusal("0xDDDD", 4), "new entry should be present")
}

func Test_AuthCache_StoreRefusal_UpdateDoesNotIncrement(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0xcccc000000000000000000000000000000000000")

	cache.StoreRefusal(wallet, 1)
	cache.StoreRefusal(wallet, 1)
	assert.Equal(t, 1, cache.negativeLen(), "duplicate StoreRefusal should not inflate count")
}

func Test_AuthCache_Store_ClearsNegativeEntry(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0xdddd000000000000000000000000000000000000")

	cache.StoreRefusal(wallet, 1)
	assert.Equal(t, 1, cache.negativeLen())

	cache.Store(wallet, 1, AuthTypeOwner)
	assert.Equal(t, 0, cache.negativeLen(), "positive store should clear negative entry")
	assert.Equal(t, 1, cache.Len())

	_, found := cache.Lookup(wallet, 1)
	assert.True(t, found, "positive entry should be present after clearing negative")
	assert.False(t, cache.LookupRefusal(wallet, 1), "negative entry should be gone")
}

func Test_AuthCache_Store_TombstoneDoesNotClearNegative(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0xeeee000000000000000000000000000000000000")

	cache.StoreRefusal(wallet, 1)
	cache.Evict(wallet, 1)

	// Tombstone should block the store; negative entry should remain
	cache.Store(wallet, 1, AuthTypeDelegate)
	assert.True(t, cache.LookupRefusal(wallet, 1),
		"negative entry should survive when tombstone blocks positive store",
	)
}

func Test_AuthCache_Degrade_ClearsNegativeEntries(t *testing.T) {
	cache := NewAuthCache(100)

	cache.StoreRefusal("0xAAAA", 1)
	cache.StoreRefusal("0xBBBB", 2)
	assert.Equal(t, 2, cache.negativeLen())

	cache.Degrade()

	assert.Equal(t, 0, cache.negativeLen(), "degrade should clear negative entries")
	assert.False(t, cache.LookupRefusal("0xAAAA", 1))
}

func Test_AuthCache_Restore_ClearsNegativeEntries(t *testing.T) {
	cache := NewAuthCache(100)

	cache.StoreRefusal("0xAAAA", 1)
	assert.Equal(t, 1, cache.negativeLen())

	cache.Restore()

	assert.Equal(t, 0, cache.negativeLen(), "restore should clear negative entries")
}

func Test_AuthCache_ClearAll_ClearsNegativeEntries(t *testing.T) {
	cache := NewAuthCache(100)

	cache.StoreRefusal("0xAAAA", 1)
	cache.Store("0xBBBB", 2, AuthTypeOwner)

	cache.clearAll()

	assert.Equal(t, 0, cache.Len())
	assert.Equal(t, 0, cache.negativeLen())
}

func Test_AuthCache_DifferentSubAccountsSeparateNegative(t *testing.T) {
	cache := NewAuthCache(100)
	wallet := snx_lib_api_types.WalletAddress("0x1111111111111111111111111111111111111111")

	cache.StoreRefusal(wallet, 1)
	cache.StoreRefusal(wallet, 2)

	assert.True(t, cache.LookupRefusal(wallet, 1))
	assert.True(t, cache.LookupRefusal(wallet, 2))
	assert.False(t, cache.LookupRefusal(wallet, 3))
}

func Test_AuthCache_ConcurrentRefusalAccess(t *testing.T) {
	cache := NewAuthCache(10000)
	wallet := snx_lib_api_types.WalletAddress("0x9999999999999999999999999999999999999999")

	var wg sync.WaitGroup
	iterations := 1000

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			cache.StoreRefusal(wallet, snx_lib_core.SubAccountId(i%10))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			cache.LookupRefusal(wallet, snx_lib_core.SubAccountId(i%10))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range iterations {
			cache.Store(wallet, snx_lib_core.SubAccountId(i%10), AuthTypeDelegate)
		}
	}()

	wg.Wait()
}

func Test_AuthCache_ConcurrentAccess(t *testing.T) {
	cache := NewAuthCache(10000)

	var wg sync.WaitGroup
	concurrency := 100
	opsPerGoroutine := 100

	// Concurrent writes
	for i := range concurrency {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range opsPerGoroutine {
				wallet := snx_lib_api_types.WalletAddress(fmt.Sprintf("0x%040x", id*opsPerGoroutine+j))
				cache.Store(wallet, snx_lib_core.SubAccountId(id), AuthTypeOwner)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := range concurrency {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range opsPerGoroutine {
				wallet := snx_lib_api_types.WalletAddress(fmt.Sprintf("0x%040x", id*opsPerGoroutine+j))
				cache.Lookup(wallet, snx_lib_core.SubAccountId(id))
			}
		}(i)
	}
	wg.Wait()

	// Concurrent mixed reads, writes, evictions, and negative cache operations
	for i := range concurrency {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range opsPerGoroutine {
				wallet := snx_lib_api_types.WalletAddress(fmt.Sprintf("0x%040x", id*opsPerGoroutine+j))
				switch j % 5 {
				case 0:
					cache.Store(wallet, snx_lib_core.SubAccountId(id), AuthTypeDelegate)
				case 1:
					cache.Lookup(wallet, snx_lib_core.SubAccountId(id))
				case 2:
					cache.Evict(wallet, snx_lib_core.SubAccountId(id))
				case 3:
					cache.StoreRefusal(wallet, snx_lib_core.SubAccountId(id))
				case 4:
					cache.LookupRefusal(wallet, snx_lib_core.SubAccountId(id))
				}
			}
		}(i)
	}
	wg.Wait()
}
