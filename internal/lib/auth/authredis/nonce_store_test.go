package authredis

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func newTestClient(t *testing.T) *snx_lib_db_redis.SnxClient {
	t.Helper()
	cfg := snx_lib_db_redis.Config{
		Addr:     "localhost:6379",
		Password: "",
		PoolSize: 10,
	}
	hostnameMap := map[string]string{"redis": "localhost"}
	rdb, err := snx_lib_db_redis.NewClientWithHostnameMapping(snx_lib_logging_doubles.NewStubLogger(), cfg, hostnameMap)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	return rdb
}

func Test_RedisNonceStore_BasicOperations(t *testing.T) {
	rdb := newTestClient(t)
	defer rdb.Close()

	store := NewNonceStore(rdb)

	ctx := context.Background()
	testAddress := "0x1234567890123456789012345678901234567890"
	testNonce := snx_lib_auth.Nonce(12345)

	defer func() {
		key := "ws:auth:nonce:" + testAddress + ":12345"
		rdb.Del(ctx, key)
	}()

	t.Run("Fresh nonce should not be used", func(t *testing.T) {
		isUsed, err := store.IsNonceUsed(testAddress, testNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.False(t, isUsed, "Fresh nonce should not be marked as used")
	})

	t.Run("Reserve nonce", func(t *testing.T) {
		reserved, err := store.ReserveNonce(testAddress, testNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, reserved, "Nonce should be successfully reserved")
	})

	t.Run("Reserve nonce atomically", func(t *testing.T) {
		reserved, err := store.ReserveNonce(testAddress, testNonce+1)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, reserved, "First reservation should succeed")

		reserved, err = store.ReserveNonce(testAddress, testNonce+1)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.False(t, reserved, "Second reservation should fail")

		key := "ws:auth:nonce:" + testAddress + ":12346"
		rdb.Del(ctx, key)
	})

	t.Run("Used nonce should be detected", func(t *testing.T) {
		isUsed, err := store.IsNonceUsed(testAddress, testNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, isUsed, "Nonce should be marked as used")
	})

	t.Run("Different nonce should not be used", func(t *testing.T) {
		differentNonce := testNonce + 1
		isUsed, err := store.IsNonceUsed(testAddress, differentNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.False(t, isUsed, "Different nonce should not be marked as used")
	})

	t.Run("Different address should not be used", func(t *testing.T) {
		differentAddress := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdef"
		isUsed, err := store.IsNonceUsed(differentAddress, testNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.False(t, isUsed, "Same nonce for different address should not be marked as used")
	})

	t.Run("CleanupExpiredNonces should not error", func(t *testing.T) {
		err := store.CleanupExpiredNonces(time.Hour)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})
}

func Test_RedisNonceStore_EdgeCases(t *testing.T) {
	rdb := newTestClient(t)
	defer rdb.Close()

	store := NewNonceStore(rdb)

	t.Run("Negative nonce", func(t *testing.T) {
		testAddress := "0x1234567890123456789012345678901234567890"
		negativeNonce := snx_lib_auth.Nonce(-123)

		isUsed, err := store.IsNonceUsed(testAddress, negativeNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.False(t, isUsed)

		reserved, err := store.ReserveNonce(testAddress, negativeNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, reserved, "Negative nonce should be successfully reserved")

		isUsed, err = store.IsNonceUsed(testAddress, negativeNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, isUsed)

		ctx := context.Background()
		key := "ws:auth:nonce:" + testAddress + ":-123"
		rdb.Del(ctx, key)
	})

	t.Run("Zero nonce", func(t *testing.T) {
		testAddress := "0x1234567890123456789012345678901234567890"
		zeroNonce := snx_lib_auth.Nonce(0)

		isUsed, err := store.IsNonceUsed(testAddress, zeroNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.False(t, isUsed)

		reserved, err := store.ReserveNonce(testAddress, zeroNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, reserved, "Zero nonce should be successfully reserved")

		isUsed, err = store.IsNonceUsed(testAddress, zeroNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, isUsed)

		ctx := context.Background()
		key := "ws:auth:nonce:" + testAddress + ":0"
		rdb.Del(ctx, key)
	})

	t.Run("Large nonce", func(t *testing.T) {
		testAddress := "0x1234567890123456789012345678901234567890"
		largeNonce := snx_lib_auth.Nonce(9223372036854775807)

		isUsed, err := store.IsNonceUsed(testAddress, largeNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.False(t, isUsed)

		reserved, err := store.ReserveNonce(testAddress, largeNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, reserved, "Large nonce should be successfully reserved")

		isUsed, err = store.IsNonceUsed(testAddress, largeNonce)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.True(t, isUsed)

		ctx := context.Background()
		key := "ws:auth:nonce:" + testAddress + ":9223372036854775807"
		rdb.Del(ctx, key)
	})
}

func Test_RedisNonceStore_getNonceKey(t *testing.T) {
	f1 := func(address string, nonce snx_lib_auth.Nonce) string {
		return fmt.Sprintf("%s:%s", address, strconv.FormatInt(int64(nonce), 10))
	}

	f2 := func(address string, nonce snx_lib_auth.Nonce) string {
		return fmt.Sprintf("%s:%d", address, int64(nonce))
	}

	f3 := func(address string, nonce snx_lib_auth.Nonce) string {
		return fmt.Sprintf("%s:%s", address, nonce.String())
	}

	f4 := func(address string, nonce snx_lib_auth.Nonce) string {
		return fmt.Sprintf("%s:%s", address, nonce)
	}

	expected := "abc:123"
	address := "abc"
	nonce := snx_lib_auth.Nonce(123)

	assert.Equal(t, expected, f1(address, nonce))
	assert.Equal(t, expected, f2(address, nonce))
	assert.Equal(t, expected, f3(address, nonce))
	assert.Equal(t, expected, f4(address, nonce))
}
