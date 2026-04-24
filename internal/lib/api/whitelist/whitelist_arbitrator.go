// Definition of `WhitelistArbitrator` and supporting types, which provide
// whitelist arbitration for REST and WS API services.
//
// NOTE: the ordering of this file is as follows:
//
// - package;
// - imports;
// - constants;
// - supporting structures and types;
// - main `WhitelistArbitrator` structure;
// - (private) initialisation methods;
// - (public) API methods;
// - (private) implementation methods;
//
// Within each section, lexiographical order is required.

package whitelist

import (
	"context"
	"sync/atomic"
	"time"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

type PermissionsMap map[WalletAddress]bool

// Obtains a distinct copy of a permissions map that may be used to grant to
// all wallet addresses.
func GrantAllPermissionsMap() PermissionsMap {
	return PermissionsMap{
		"*": true,
	}
}

// Concurrency safe permissions holder.
type whitelist struct {
	permissions atomic.Value // map[WalletAddress]bool
}

// Determines whether a given address is recognised and is permitted.
func (wl *whitelist) isWalletAllowed(walletAddress WalletAddress) bool {
	m := wl.permissions.Load().(PermissionsMap)

	walletAddress = normalizeWalletAddress(walletAddress)

	// special case where the entire map matches the grant-all permissions map
	if len(m) == 1 {
		aStar, eStar := m["*"]

		if eStar && aStar {
			return true
		}
	}

	allowed, exists := m[walletAddress]

	if exists {
		return allowed
	}

	return false
}

// Atomically update the permissions map.
func (wl *whitelist) updatePermissions(m PermissionsMap) {
	wl.permissions.Store(m)
}

// Provides arbitration of whitelist requests, based on wallet addresses.
//
// The arbitrator makes decisions from a whitelist that it obtains from the
// Redis key (provided in `NewWhitelistArbitrator()`).
type WhitelistArbitrator struct {
	ctx         context.Context
	logger      snx_lib_logging.Logger
	rc          *snx_lib_db_redis.SnxClient
	diagnostics *WhitelistDiagnostics
	whitelist   whitelist
}

// Creates a new instance of `WhitelistArbitrator`.
//
// Parameters:
//   - logger - logger;
//   - ctx - standard execution context, for cancellation of subtasks;
//   - rc - Redis connection;
//   - redisKey - Redis key;
//   - diagnostics - optional diagnostics structure for counting events;
//   - initialPermissions - optional map of initial permissions;
//
// Note:
// If `initialPermissions` contains the key `"*"` then
func NewWhitelistArbitrator(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	redisKey string,
	diagnostics *WhitelistDiagnostics,
	initialPermissions PermissionsMap,
	updateInterval time.Duration,
) (
	arbitrator *WhitelistArbitrator,
	err error,
) {
	if rc == nil {
		err = errRedisClientRequired

		return
	}

	if initialPermissions == nil {
		initialPermissions = make(PermissionsMap)
	}

	initialPermissions = normalizePermissions(initialPermissions)

	var permissions atomic.Value
	permissions.Store(initialPermissions)

	whitelist := whitelist{
		permissions: permissions,
	}

	arbitrator = &WhitelistArbitrator{
		ctx:       ctx,
		logger:    logger,
		rc:        rc,
		whitelist: whitelist,
	}

	// Check if redis connection "IsValid", otherwise we must assume we're in
	// test mode.
	if arbitrator.rc.IsValid() {

		arbitrator.update(
			logger,
			ctx,
			redisKey,
		)

		go arbitrator.runUpdates(
			logger,
			ctx,
			redisKey,
			updateInterval,
		)
	}

	return
}

func (wla *WhitelistArbitrator) CanOrdersBePlacedFor(
	walletAddress WalletAddress,
) (
	r ArbitrationResult,
	err error,
) {

	r = wla.whitelist.isWalletAllowed(walletAddress)

	return
}

func (wla *WhitelistArbitrator) runUpdates(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	redisKey string,
	updateInterval time.Duration,
) {
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

outerLoop:
	for {
		select {
		case <-ctx.Done():
			break outerLoop
		case <-ticker.C:
			wla.update(logger, ctx, redisKey)
		}
	}
}

func (wla *WhitelistArbitrator) update(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	redisKey string,
) {
	data, err := wla.rc.Get(ctx, redisKey).Result()
	if err != nil {

		logger.Error("failed to read whitelist",
			"error", err,
			"key", redisKey,
		)
	} else {

		m, err := parseWalletWhitelist(data)
		if err != nil {

			logger.Error("failed to parse whitelist",
				"error", err,
				"key", redisKey,
			)
		} else {

			wla.whitelist.updatePermissions(m)
		}
	}
}
