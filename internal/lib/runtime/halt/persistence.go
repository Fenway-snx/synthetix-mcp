package halt

import (
	"context"
	"errors"
	"fmt"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
)

// TargetStateKeyPrefix is the Redis key prefix for admin-set target state.
// Exported so that read-only consumers (e.g. lib/exchange/status) use the
// same key the write path uses, keeping them in sync.
const TargetStateKeyPrefix = "snx:admin:target_state:"

var (
	errStatePersistenceRedisUnavailable = errors.New("state persistence redis client is unavailable")
)

// Stores admin target lifecycle state (IDLE/STOPPED) in Redis.
// Keys have no time-to-live; they remain until ClearTargetState runs on
// RUNNING.
type StatePersistence struct {
	redisClient *snx_lib_db_redis.SnxClient
	serviceId   string
}

func NewStatePersistence(
	redisClient *snx_lib_db_redis.SnxClient,
	serviceId string,
) *StatePersistence {
	return &StatePersistence{
		redisClient: redisClient,
		serviceId:   serviceId,
	}
}

func (p *StatePersistence) LoadTargetState(ctx context.Context) (TargetState, error) {
	if p == nil || p.redisClient == nil || !p.redisClient.IsValid() {
		return "", nil
	}

	value, err := p.redisClient.Get(ctx, p.targetKey()).Result()
	if err != nil {
		if errors.Is(err, snx_lib_db_redis.Nil) {
			return "", nil
		}
		return "", fmt.Errorf("loading target state from redis: %w", err)
	}

	target := TargetState(value)
	if !isValidTargetState(target) {
		return "", nil
	}

	return target, nil
}

func (p *StatePersistence) SaveTargetState(ctx context.Context, target TargetState) error {
	if p == nil || p.redisClient == nil || !p.redisClient.IsValid() {
		return errStatePersistenceRedisUnavailable
	}

	if !isValidTargetState(target) {
		return fmt.Errorf("invalid target state: %s", target)
	}

	// Zero expiration: key persists until explicit delete on resume (RUNNING).
	if err := p.redisClient.Set(ctx, p.targetKey(), string(target), 0).Err(); err != nil {
		return fmt.Errorf("saving target state to redis: %w", err)
	}

	return nil
}

func (p *StatePersistence) ClearTargetState(ctx context.Context) error {
	if p == nil || p.redisClient == nil || !p.redisClient.IsValid() {
		return errStatePersistenceRedisUnavailable
	}

	if err := p.redisClient.Del(ctx, p.targetKey()).Err(); err != nil {
		return fmt.Errorf("clearing target state from redis: %w", err)
	}

	return nil
}

func (p *StatePersistence) targetKey() string {
	return TargetStateKeyPrefix + p.serviceId
}
