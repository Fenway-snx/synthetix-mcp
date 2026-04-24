package status

import (
	"context"
	"testing"
	"time"

	go_redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_runtime_halt "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/halt"
)

// unreachableRedisClient returns an SnxClient that passes IsValid() but
// connects to an unreachable address, causing every command to fail with
// a connection error (not redis.Nil).
func unreachableRedisClient() *snx_lib_db_redis.SnxClient {
	rdb := go_redis.NewClusterClient(&go_redis.ClusterOptions{
		Addrs: []string{"127.0.0.1:1"},
	})
	return &snx_lib_db_redis.SnxClient{ClusterClient: rdb}
}

// stubReader returns a targetStateReaderFunc that always returns the
// given value and ok.
func stubReader(
	value string,
	ok bool,
) targetStateReaderFunc {
	return func(context.Context, *snx_lib_db_redis.SnxClient, string, time.Duration) (string, bool) {
		return value, ok
	}
}

// --- readRedisTargetStateWithTimeout (real implementation) ---

func Test_readRedisTargetStateWithTimeout_ConnectionError(t *testing.T) {
	t.Parallel()

	rc := unreachableRedisClient()
	defer rc.Close()

	val, ok := readRedisTargetStateWithTimeout(
		context.Background(), rc, "api", 100*time.Millisecond,
	)
	assert.Equal(t, "", val)
	assert.False(t, ok)
}

func Test_readRedisTargetStateWithTimeout_CancelledContext(t *testing.T) {
	t.Parallel()

	rc := unreachableRedisClient()
	defer rc.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	val, ok := readRedisTargetStateWithTimeout(
		ctx, rc, "api", 100*time.Millisecond,
	)
	assert.Equal(t, "", val)
	assert.False(t, ok)
}

// --- Build via injected reader (covers all decision branches) ---

func Test_Build_WhenRedisReturnsEmpty_ReturnsRunningOK(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		ServiceId:             "api",
		readTargetState:       stubReader("", true),
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusRunning, out.ExchangeStatus)
	assert.Empty(t, out.Code)
	assert.Equal(t, "OK", out.Message)
	assert.Greater(t, out.TimestampMs, int64(0))
}

func Test_Build_WhenRedisReturnsRUNNING_ReturnsRunningOK(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		ServiceId:             "api",
		readTargetState:       stubReader(string(snx_lib_runtime_halt.TargetState_Running), true),
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusRunning, out.ExchangeStatus)
	assert.Empty(t, out.Code)
	assert.Equal(t, "OK", out.Message)
}

func Test_Build_WhenRedisReturnsIDLE_ReturnsMaintenance(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		ServiceId:             "api",
		readTargetState:       stubReader(string(snx_lib_runtime_halt.TargetState_Idle), true),
	})

	assert.False(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusMaintenance, out.ExchangeStatus)
	assert.Equal(t, codeServiceDraining, out.Code)
	assert.Equal(t, "Service is draining for deployment", out.Message)
	assert.Greater(t, out.TimestampMs, int64(0))
}

func Test_Build_WhenRedisReturnsSTOPPED_ReturnsMaintenance(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		ServiceId:             "api",
		readTargetState:       stubReader(string(snx_lib_runtime_halt.TargetState_Stopped), true),
	})

	assert.False(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusMaintenance, out.ExchangeStatus)
	assert.Equal(t, codeServiceDraining, out.Code)
}

func Test_Build_WhenRedisReadFails_ReturnsDegraded(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		ServiceId:             "api",
		readTargetState:       stubReader("", false),
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusRunning, out.ExchangeStatus)
	assert.Equal(t, codeStatusDegraded, out.Code)
	assert.Equal(t, "OK (degraded)", out.Message)
}

func Test_Build_WhenRedisUnreachable_ReturnsDegraded(t *testing.T) {
	t.Parallel()

	rc := unreachableRedisClient()
	defer rc.Close()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           100 * time.Millisecond,
		RedisClient:           rc,
		ServiceId:             "api",
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusRunning, out.ExchangeStatus)
	assert.Equal(t, codeStatusDegraded, out.Code)
	assert.Equal(t, "OK (degraded)", out.Message)
	assert.Greater(t, out.TimestampMs, int64(0))
}

func Test_Build_WhenCancelledContext_ReturnsDegraded(t *testing.T) {
	t.Parallel()

	rc := unreachableRedisClient()
	defer rc.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := Build(ctx, Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           100 * time.Millisecond,
		RedisClient:           rc,
		ServiceId:             "api",
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusRunning, out.ExchangeStatus)
	assert.Equal(t, codeStatusDegraded, out.Code)
}

func Test_Build_HaltingTakesPriorityOverRedis(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: true,
		ReadTimeout:           50 * time.Millisecond,
		ServiceId:             "api",
		readTargetState:       stubReader(string(snx_lib_runtime_halt.TargetState_Running), true),
	})

	assert.False(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusMaintenance, out.ExchangeStatus)
	assert.Equal(t, codeServiceDraining, out.Code)
}

func Test_Build_NilReaderFallsBackToRealImplementation(t *testing.T) {
	t.Parallel()

	out := Build(context.Background(), Inputs{
		LocalServiceIsHalting: false,
		ReadTimeout:           50 * time.Millisecond,
		RedisClient:           nil,
		ServiceId:             "api",
		readTargetState:       nil,
	})

	assert.True(t, out.AcceptingOrders)
	assert.Equal(t, exchangeStatusRunning, out.ExchangeStatus)
	assert.Equal(t, codeStatusDegraded, out.Code)
}
