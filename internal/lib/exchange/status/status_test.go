package status

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
)

func Test_Build_WhenLocalHalting_ReturnsMaintenance(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: true,
		ReadTimeout:           50 * time.Millisecond,
		RedisClient:           nil,
		ServiceId:             "api",
	})

	assert.False(t, out.AcceptingOrders)
	assert.Equal(t, "MAINTENANCE", out.ExchangeStatus)
	assert.Equal(t, "SERVICE_DRAINING", out.Code)
	assert.Equal(t, "Service is draining for deployment", out.Message)
	assert.Greater(t, out.TimestampMs, int64(0))
}

func Test_Build_WhenLocalHalting_SkipsRedis(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: true,
		ReadTimeout:           0,
		RedisClient:           nil,
		ServiceId:             "",
	})

	assert.False(t, out.AcceptingOrders)
	assert.Equal(t, "MAINTENANCE", out.ExchangeStatus)
	assert.Equal(t, "SERVICE_DRAINING", out.Code)
}

func Test_Build_WhenNilRedis_DegradedWithoutLeaking(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		RedisClient:           nil,
		ServiceId:             "api",
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, "RUNNING", out.ExchangeStatus)
	assert.Equal(t, "STATUS_DEGRADED", out.Code)
	assert.Equal(t, "OK (degraded)", out.Message)
	assert.Greater(t, out.TimestampMs, int64(0))
}

func Test_Build_WhenEmptyServiceId_DegradedWithoutLeaking(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		RedisClient:           nil,
		ServiceId:             "",
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, "RUNNING", out.ExchangeStatus)
	assert.Equal(t, "STATUS_DEGRADED", out.Code)
}

func Test_Build_WhenZeroTimeout_DegradedWithoutLeaking(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           0,
		RedisClient:           invalidRedis,
		ServiceId:             "api",
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, "RUNNING", out.ExchangeStatus)
	assert.Equal(t, "STATUS_DEGRADED", out.Code)
}

func Test_Build_WhenNegativeTimeout_DegradedWithoutLeaking(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           -1 * time.Millisecond,
		RedisClient:           invalidRedis,
		ServiceId:             "api",
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, "RUNNING", out.ExchangeStatus)
	assert.Equal(t, "STATUS_DEGRADED", out.Code)
}

func Test_Build_WhenInvalidRedisClient_DegradedWithoutLeaking(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	require.False(t, invalidRedis.IsValid())

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		RedisClient:           invalidRedis,
		ServiceId:             "api",
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, "RUNNING", out.ExchangeStatus)
	assert.Equal(t, "STATUS_DEGRADED", out.Code)
}

func Test_Build_WhenNotHaltingAndNilRedis_MessageIsNonEmpty(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		RedisClient:           nil,
		ServiceId:             "api",
	})

	assert.NotEmpty(t, out.Message)
}

func Test_readRedisTargetStateWithTimeout_NilClient(t *testing.T) {
	t.Parallel()

	val, ok := readRedisTargetStateWithTimeout(
		context.Background(), nil, "api", 50*time.Millisecond,
	)
	assert.Equal(t, "", val)
	assert.False(t, ok)
}

func Test_readRedisTargetStateWithTimeout_InvalidClient(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	val, ok := readRedisTargetStateWithTimeout(
		context.Background(), invalidRedis, "api", 50*time.Millisecond,
	)
	assert.Equal(t, "", val)
	assert.False(t, ok)
}

func Test_readRedisTargetStateWithTimeout_EmptyServiceId(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	val, ok := readRedisTargetStateWithTimeout(
		context.Background(), invalidRedis, "", 50*time.Millisecond,
	)
	assert.Equal(t, "", val)
	assert.False(t, ok)
}

func Test_readRedisTargetStateWithTimeout_ZeroTimeout(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	val, ok := readRedisTargetStateWithTimeout(
		context.Background(), invalidRedis, "api", 0,
	)
	assert.Equal(t, "", val)
	assert.False(t, ok)
}

func Test_readRedisTargetStateWithTimeout_NegativeTimeout(t *testing.T) {
	t.Parallel()

	invalidRedis := &snx_lib_db_redis.SnxClient{}
	val, ok := readRedisTargetStateWithTimeout(
		context.Background(), invalidRedis, "api", -1*time.Millisecond,
	)
	assert.Equal(t, "", val)
	assert.False(t, ok)
}
