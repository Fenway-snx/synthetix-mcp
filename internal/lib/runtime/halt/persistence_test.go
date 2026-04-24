package halt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
)

func Test_NewStatePersistence(t *testing.T) {
	t.Parallel()

	sp := NewStatePersistence(nil, "test-service")
	require.NotNil(t, sp)
	assert.Equal(t, "test-service", sp.serviceId)
	assert.Nil(t, sp.redisClient)
}

func Test_StatePersistence_targetKey(t *testing.T) {
	t.Parallel()

	sp := NewStatePersistence(nil, "my-service")
	assert.Equal(t, TargetStateKeyPrefix+"my-service", sp.targetKey())
}

func Test_StatePersistence_LoadTargetState_NilReceiver(t *testing.T) {
	t.Parallel()

	var sp *StatePersistence
	target, err := sp.LoadTargetState(context.Background())
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, TargetState(""), target)
}

func Test_StatePersistence_LoadTargetState_NilRedisClient(t *testing.T) {
	t.Parallel()

	sp := NewStatePersistence(nil, "test-service")
	target, err := sp.LoadTargetState(context.Background())
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, TargetState(""), target)
}

func Test_StatePersistence_LoadTargetState_InvalidRedisClient(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	require.False(t, invalidRedis.IsValid())

	sp := NewStatePersistence(invalidRedis, "test-service")
	target, err := sp.LoadTargetState(context.Background())
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, TargetState(""), target)
}

func Test_StatePersistence_SaveTargetState_NilReceiver(t *testing.T) {
	t.Parallel()

	var sp *StatePersistence
	err := sp.SaveTargetState(context.Background(), TargetState_Running)
	assert.ErrorIs(t, err, errStatePersistenceRedisUnavailable)
}

func Test_StatePersistence_SaveTargetState_NilRedisClient(t *testing.T) {
	t.Parallel()

	sp := NewStatePersistence(nil, "test-service")
	err := sp.SaveTargetState(context.Background(), TargetState_Running)
	assert.ErrorIs(t, err, errStatePersistenceRedisUnavailable)
}

func Test_StatePersistence_SaveTargetState_InvalidRedisClient(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	sp := NewStatePersistence(invalidRedis, "test-service")
	err := sp.SaveTargetState(context.Background(), TargetState_Running)
	assert.ErrorIs(t, err, errStatePersistenceRedisUnavailable)
}

func Test_StatePersistence_SaveTargetState_InvalidTargetState(t *testing.T) {
	t.Parallel()

	// Nil receiver still exercises the redis-unavailable guard path.
	var sp *StatePersistence
	err := sp.SaveTargetState(context.Background(), TargetState("BOGUS"))
	assert.Error(t, err)
}

func Test_StatePersistence_ClearTargetState_NilReceiver(t *testing.T) {
	t.Parallel()

	var sp *StatePersistence
	err := sp.ClearTargetState(context.Background())
	assert.ErrorIs(t, err, errStatePersistenceRedisUnavailable)
}

func Test_StatePersistence_ClearTargetState_NilRedisClient(t *testing.T) {
	t.Parallel()

	sp := NewStatePersistence(nil, "test-service")
	err := sp.ClearTargetState(context.Background())
	assert.ErrorIs(t, err, errStatePersistenceRedisUnavailable)
}

func Test_StatePersistence_ClearTargetState_InvalidRedisClient(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	sp := NewStatePersistence(invalidRedis, "test-service")
	err := sp.ClearTargetState(context.Background())
	assert.ErrorIs(t, err, errStatePersistenceRedisUnavailable)
}

func Test_TargetStateKeyPrefix_Value(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "snx:admin:target_state:", TargetStateKeyPrefix)
}
