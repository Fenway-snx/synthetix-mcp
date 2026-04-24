// Package authredis provides a Redis-backed implementation of
// snx_lib_auth.NonceStore. It lives in a sub-package so production builds of
// lib/auth stay free of github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis —
// only services that actually run on Redis pull in that dependency.
package authredis

import (
	"context"
	"fmt"
	"time"

	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
)

// NonceStore implements snx_lib_auth.NonceStore on top of the shared cluster
// client. Reservations are atomic SETNX with a fixed TTL.
type NonceStore struct {
	rc     *snx_lib_db_redis.SnxClient
	prefix string
}

func NewNonceStore(rc *snx_lib_db_redis.SnxClient) snx_lib_auth.NonceStore {
	return &NonceStore{
		rc:     rc,
		prefix: "ws:auth:nonce:",
	}
}

func (r *NonceStore) IsNonceUsed(address string, nonce snx_lib_auth.Nonce) (bool, error) {
	ctx := context.Background()
	key := r.getNonceKey(address, nonce)

	exists := r.rc.Exists(ctx, key)
	if exists.Err() != nil {
		return false, fmt.Errorf("failed to check nonce existence: %w", exists.Err())
	}

	return exists.Val() > 0, nil
}

// Atomically reserves a nonce; returns true on first reservation, false if the
// nonce was already taken (replay).
func (r *NonceStore) ReserveNonce(address string, nonce snx_lib_auth.Nonce) (bool, error) {
	ctx := context.Background()
	key := r.getNonceKey(address, nonce)

	result := r.rc.SetNX(ctx, key, "used", 10*time.Minute)
	if result.Err() != nil {
		return false, fmt.Errorf("failed to reserve nonce: %w", result.Err())
	}

	return result.Val(), nil
}

// No-op; Redis TTL handles expiration.
func (r *NonceStore) CleanupExpiredNonces(maxAge time.Duration) error {
	return nil
}

func (r *NonceStore) getNonceKey(address string, nonce snx_lib_auth.Nonce) string {
	return fmt.Sprintf("%s%s:%s", r.prefix, address, nonce)
}
