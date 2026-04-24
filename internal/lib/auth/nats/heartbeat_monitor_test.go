package authnats

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

func newTestMonitor(timeout time.Duration, onMissed, onRecovered func()) *HeartbeatMonitor {
	m := &HeartbeatMonitor{
		done:        make(chan struct{}),
		onMissed:    onMissed,
		onRecovered: onRecovered,
		timeout:     timeout,
	}
	m.lastSeen.Store(snx_lib_utils_time.Now().UnixMicro())
	return m
}

func Test_HeartbeatMonitor_CallsOnMissedAfterTimeout(t *testing.T) {
	var missed atomic.Int32
	m := newTestMonitor(150*time.Millisecond, func() {
		missed.Add(1)
	}, func() {})

	go m.run(snx_lib_logging_doubles.NewStubLogger())
	defer m.Stop()

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, int32(1), missed.Load(), "onMissed should be called exactly once")
}

func Test_HeartbeatMonitor_HeartbeatsPreventCallback(t *testing.T) {
	var missed atomic.Int32
	m := newTestMonitor(150*time.Millisecond, func() {
		missed.Add(1)
	}, func() {})

	go m.run(snx_lib_logging_doubles.NewStubLogger())
	defer m.Stop()

	for range 5 {
		time.Sleep(40 * time.Millisecond)
		m.lastSeen.Store(snx_lib_utils_time.Now().UnixMicro())
	}

	assert.Equal(t, int32(0), missed.Load(), "onMissed should not be called when heartbeats arrive")
}

func Test_HeartbeatMonitor_RecoveryCallsOnRecovered(t *testing.T) {
	var missed, recovered atomic.Int32
	m := newTestMonitor(100*time.Millisecond, func() {
		missed.Add(1)
	}, func() {
		recovered.Add(1)
	})

	go m.run(snx_lib_logging_doubles.NewStubLogger())
	defer m.Stop()

	// Wait for first miss
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), missed.Load(), "first miss")
	assert.Equal(t, int32(0), recovered.Load(), "no recovery yet")

	// Simulate recovery
	m.lastSeen.Store(snx_lib_utils_time.Now().UnixMicro())
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), recovered.Load(), "onRecovered should be called on recovery")

	// Let it miss again
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(2), missed.Load(), "second miss after recovery")
	assert.Equal(t, int32(1), recovered.Load(), "onRecovered should not be called again until next recovery")
}

func Test_HeartbeatMonitor_ForceMissedRecoveredByHeartbeat(t *testing.T) {
	var missed, recovered atomic.Int32
	m := newTestMonitor(300*time.Millisecond, func() {
		missed.Add(1)
	}, func() {
		recovered.Add(1)
	})

	go m.run(snx_lib_logging_doubles.NewStubLogger())
	defer m.Stop()

	// ForceMissed triggers onMissed immediately, even though heartbeats
	// are healthy (simulates a NATS disconnect handler).
	m.ForceMissed()
	assert.Equal(t, int32(1), missed.Load(), "ForceMissed should call onMissed immediately")

	// Simulate heartbeat arriving after reconnect.
	m.lastSeen.Store(snx_lib_utils_time.Now().UnixMicro())

	// The run loop should pick up the forceDegraded flag and then see
	// healthy heartbeats, triggering onRecovered.
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), recovered.Load(), "onRecovered should be called when heartbeats resume after ForceMissed")
}

func Test_HeartbeatMonitor_ForceMissedWithoutRecovery(t *testing.T) {
	var missed, recovered atomic.Int32
	m := newTestMonitor(150*time.Millisecond, func() {
		missed.Add(1)
	}, func() {
		recovered.Add(1)
	})

	go m.run(snx_lib_logging_doubles.NewStubLogger())
	defer m.Stop()

	// ForceMissed with no subsequent heartbeats — should not recover.
	m.ForceMissed()
	time.Sleep(250 * time.Millisecond)

	assert.Equal(t, int32(1), missed.Load(), "ForceMissed should call onMissed once")
	assert.Equal(t, int32(0), recovered.Load(), "should not recover without heartbeats")
}
