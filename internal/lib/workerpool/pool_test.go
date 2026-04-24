package workerpool

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	snowflakeNodeBits     = 10
	snowflakeSequenceBits = 12
	snowflakeShift        = snowflakeNodeBits + snowflakeSequenceBits // 22
)

func snowflakeID(timestamp uint64) int64 {
	return int64(timestamp << snowflakeShift)
}

func newTestPool(numWorkers int) *Pool[struct{}] {
	pool, _ := NewPool[struct{}](
		"test-pool",
		context.Background(),
		Config{
			NumWorkers: numWorkers,
			QueueSize:  1,
		},
	)

	return pool
}

func Test_GetChannelByID_ReturnsNonNil(t *testing.T) {
	maxSnowflakeTS := uint64(1)<<41 - 1

	tests := []struct {
		name       string
		numWorkers int
		id         int64
	}{
		{"single_worker", 1, snowflakeID(42)},
		{"zero_id", 4, 0},
		{"max_int64", 7, math.MaxInt64},
		{"large_timestamp", 7, snowflakeID(maxSnowflakeTS)},
		{"prime_workers", 31, snowflakeID(1000)},
		{"power_of_two_workers", 16, snowflakeID(1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newTestPool(tt.numWorkers)
			defer pool.Cancel()
			require.NotNil(t, pool.GetChannelByID(tt.id))
		})
	}
}

// Verifies that snowflake IDs distribute within 10% of the expected count
// for each worker, across both prime and power-of-two worker counts.
func Test_GetChannelByID_Distribution(t *testing.T) {
	workerCounts := []int{2, 3, 5, 7, 8, 10, 16, 31, 32, 64}

	for _, numWorkers := range workerCounts {
		t.Run(fmt.Sprintf("workers=%d", numWorkers), func(t *testing.T) {
			pool := newTestPool(numWorkers)
			defer pool.Cancel()

			counts := make(map[chan *struct{}]int)
			const numIDs = 100_000
			for ts := uint64(1); ts <= numIDs; ts++ {
				counts[pool.GetChannelByID(snowflakeID(ts))]++
			}

			assert.Equal(t, numWorkers, len(counts),
				"expected %d distinct workers, got %d", numWorkers, len(counts))

			expected := float64(numIDs) / float64(numWorkers)
			lo := expected * 0.9
			hi := expected * 1.1
			for ch, c := range counts {
				assert.GreaterOrEqual(t, float64(c), lo,
					"channel %v received %d jobs, expected at least %.0f", ch, c, lo,
				)
				assert.LessOrEqual(t, float64(c), hi,
					"channel %v received %d jobs, expected at most %.0f", ch, c, hi,
				)
			}
		})
	}
}

func Test_GetChannelByID_Deterministic(t *testing.T) {
	pool := newTestPool(16)
	defer pool.Cancel()

	testIDs := []int64{0, 1, 42, math.MaxInt64, snowflakeID(999), snowflakeID(123456)}

	for _, id := range testIDs {
		expected := pool.GetChannelByID(id)
		for i := 0; i < 100; i++ {
			assert.Equal(t, expected, pool.GetChannelByID(id),
				"ID %d mapped to different channels on call %d", id, i,
			)
		}
	}
}

func Test_NewPool_ReturnsErrorOnInvalidConfig(t *testing.T) {
	t.Run("zero_workers", func(t *testing.T) {
		pool, err := NewPool[struct{}]("test", context.Background(), Config{NumWorkers: 0, QueueSize: 1})
		assert.Nil(t, pool)
		assert.ErrorIs(t, err, errNumWorkersMustBePositive)
	})

	t.Run("negative_workers", func(t *testing.T) {
		pool, err := NewPool[struct{}]("test", context.Background(), Config{NumWorkers: -1, QueueSize: 1})
		assert.Nil(t, pool)
		assert.ErrorIs(t, err, errNumWorkersMustBePositive)
	})

	t.Run("zero_queue_size", func(t *testing.T) {
		pool, err := NewPool[struct{}]("test", context.Background(), Config{NumWorkers: 4, QueueSize: 0})
		assert.Nil(t, pool)
		assert.ErrorIs(t, err, errQueueSizeMustBePositive)
	})
}

func Test_Pool_WorkerChannels(t *testing.T) {
	pool := newTestPool(8)
	defer pool.Cancel()

	channels := pool.WorkerChannels()
	assert.Len(t, channels, 8)

	// Each channel should be unique
	seen := make(map[chan *struct{}]bool)
	for _, ch := range channels {
		assert.False(t, seen[ch], "duplicate channel found")
		seen[ch] = true
	}
}

func Test_Pool_Metrics(t *testing.T) {
	pool := newTestPool(4)
	defer pool.Cancel()

	// Initial metrics should be zero
	m := pool.GetMetrics()
	assert.Equal(t, uint64(0), m.Processed)
	assert.Equal(t, uint64(0), m.Failed)
	assert.Equal(t, int64(0), m.QueueDepth)

	// Increment and verify
	pool.IncrementProcessed()
	pool.IncrementProcessed()
	pool.IncrementFailed()
	pool.IncrementQueueDepth()
	pool.IncrementQueueDepth()
	pool.DecrementQueueDepth()

	m = pool.GetMetrics()
	assert.Equal(t, uint64(2), m.Processed)
	assert.Equal(t, uint64(1), m.Failed)
	assert.Equal(t, int64(1), m.QueueDepth)
}

func Test_Pool_RegisterWorkerAndWait(t *testing.T) {
	pool := newTestPool(2)
	defer pool.Cancel()

	done := make(chan struct{})

	workerDone := pool.RegisterWorker()
	go func() {
		defer workerDone()
		// Simulate work
	}()

	go func() {
		pool.Wait()
		close(done)
	}()

	// Wait should return after the worker's done callback fires
	<-done
}

func Test_Pool_PoolName(t *testing.T) {
	pool, _ := NewPool[int](
		"my-pool",
		context.Background(),
		Config{NumWorkers: 1, QueueSize: 1},
	)
	defer pool.Cancel()

	assert.Equal(t, "my-pool", pool.PoolName())
}

func Test_Pool_ContextCancellation(t *testing.T) {
	pool := newTestPool(2)

	// Context should not be done initially
	select {
	case <-pool.Context().Done():
		t.Fatal("context should not be done")
	default:
	}

	pool.Cancel()

	// Context should be done after cancel
	select {
	case <-pool.Context().Done():
		// expected
	default:
		t.Fatal("context should be done after cancel")
	}
}
