package workerpool

import (
	"context"
	"errors"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"
)

// Controls worker pool runtime behaviour.
type Config struct {
	NumWorkers      int
	QueueSize       int
	ShutdownTimeout time.Duration
}

// Approximately point-in-time snapshot of worker pool state.
type Metrics struct {
	Failed     uint64
	Processed  uint64
	QueueDepth int64
}

// statistics holds atomic counters for pool instrumentation.
type statistics struct {
	failed     atomic.Uint64
	processed  atomic.Uint64
	queueDepth atomic.Int64
}

// hashSeed is initialised once at program start and remains stable for the
// lifetime of the process (per maphash docs). No sync.Once is needed because
// package-level variable initialisation is inherently synchronised.
var hashSeed = maphash.MakeSeed()

// Generic partitioned worker pool. Each worker owns a dedicated channel;
// callers route jobs to a deterministic worker via GetChannelByID.
// Only manages channels, statistics, and lifecycle coordination — callers
// are responsible for starting their own worker goroutines via
// WorkerChannels + RegisterWorker.
type Pool[J any] struct {
	poolName    string
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	workerChans []chan *J
	statistics
}

var (
	errNumWorkersMustBePositive = errors.New("workerpool: NumWorkers must be > 0")
	errQueueSizeMustBePositive  = errors.New("workerpool: QueueSize must be > 0")
)

// Creates a partitioned worker pool with per-worker channels.
// Returns an error if NumWorkers <= 0 or QueueSize <= 0.
// Workers are NOT started — callers retrieve channels via WorkerChannels()
// and launch their own goroutines, calling RegisterWorker() for each.
func NewPool[J any](
	poolName string,
	ctx context.Context,
	cfg Config,
) (*Pool[J], error) {
	if cfg.NumWorkers <= 0 {
		return nil, errNumWorkersMustBePositive
	}
	if cfg.QueueSize <= 0 {
		return nil, errQueueSizeMustBePositive
	}

	ctx, cancel := context.WithCancel(ctx)

	workerChans := make([]chan *J, cfg.NumWorkers)
	for i := range workerChans {
		workerChans[i] = make(chan *J, cfg.QueueSize)
	}

	return &Pool[J]{
		poolName:    poolName,
		ctx:         ctx,
		cancel:      cancel,
		workerChans: workerChans,
	}, nil
}

// Provides a consistent mapping of an id to a worker channel. The same id
// always maps to the same channel within a process, ensuring per-entity
// ordering. Designed to handle snowflake IDs gracefully.
func (p *Pool[J]) GetChannelByID(id int64) chan *J {
	h := maphash.Comparable(hashSeed, id)
	workerID := int(h % uint64(len(p.workerChans)))
	return p.workerChans[workerID]
}

// Returns the per-worker channel slice. Callers iterate this to launch
// one goroutine per channel.
func (p *Pool[J]) WorkerChannels() []chan *J {
	return p.workerChans
}

// Returns an approximately point-in-time snapshot of pool counters.
func (p *Pool[J]) GetMetrics() Metrics {
	return Metrics{
		Failed:     p.failed.Load(),
		Processed:  p.processed.Load(),
		QueueDepth: p.queueDepth.Load(),
	}
}

// Records a successfully processed job.
func (p *Pool[J]) IncrementProcessed() { p.processed.Add(1) }

// Records n successfully processed jobs (batch flush).
func (p *Pool[J]) IncrementProcessedBy(n uint64) { p.processed.Add(n) }

// Records a dropped or errored job.
func (p *Pool[J]) IncrementFailed() { p.failed.Add(1) }

// Records n dropped or errored jobs (batch flush).
func (p *Pool[J]) IncrementFailedBy(n uint64) { p.failed.Add(n) }

// Increments the current queue depth gauge.
func (p *Pool[J]) IncrementQueueDepth() { p.queueDepth.Add(1) }

// Decrements the current queue depth gauge.
func (p *Pool[J]) DecrementQueueDepth() { p.queueDepth.Add(-1) }

// Returns the pool's display name.
func (p *Pool[J]) PoolName() string { return p.poolName }

// Returns the pool's context (read-only). Workers should select on this
// for shutdown signalling.
func (p *Pool[J]) Context() context.Context { return p.ctx }

// Signals all workers to shut down.
func (p *Pool[J]) Cancel() { p.cancel() }

// Increments the WaitGroup and returns a done callback that decrements it.
// Callers must defer the returned function in their worker goroutine.
func (p *Pool[J]) RegisterWorker() (done func()) {
	p.wg.Add(1)
	return func() { p.wg.Done() }
}

// Blocks until all registered workers have called their done callback.
func (p *Pool[J]) Wait() { p.wg.Wait() }
