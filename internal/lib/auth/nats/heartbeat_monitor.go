package authnats

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	DefaultHeartbeatTimeout  = 60 * time.Second
	HeartbeatPublishInterval = 20 * time.Second
)

// Monitors a periodic NATS heartbeat from the subaccount service (the
// publisher of delegation revocation events). If no heartbeat arrives
// within the configured timeout, onMissed is called to degrade the auth
// cache. When heartbeats resume after a miss, onRecovered is called to
// restore normal operation.
//
// External callers (e.g. NATS disconnect handlers) can call ForceMissed()
// to immediately degrade and synchronize with the monitor's run loop, so
// that recovery is handled correctly when heartbeats resume.
type HeartbeatMonitor struct {
	lastSeen      atomic.Int64 // unix microseconds
	forceDegraded atomic.Bool  // set by ForceMissed, consumed by run loop
	sub           *nats.Subscription
	done          chan struct{}
	onMissed      func()
	onRecovered   func()
	stopOnce      sync.Once
	timeout       time.Duration
}

// Subscribes to the auth heartbeat subject and starts a background
// checker. If no heartbeat arrives within timeout, onMissed is called
// once. When heartbeats resume after a miss, onRecovered is called once.
// The cycle repeats on subsequent timeouts.
func NewHeartbeatMonitor(
	logger snx_lib_logging.Logger,
	conn *nats.Conn,
	timeout time.Duration,
	onMissed func(),
	onRecovered func(),
) (*HeartbeatMonitor, error) {
	m := &HeartbeatMonitor{
		done:        make(chan struct{}),
		onMissed:    onMissed,
		onRecovered: onRecovered,
		timeout:     timeout,
	}
	m.lastSeen.Store(snx_lib_utils_time.Now().UnixMicro())

	subject := snx_lib_db_nats.SubaccountHeartbeatAuth.String()
	sub, err := conn.Subscribe(subject, func(_ *nats.Msg) {
		m.lastSeen.Store(snx_lib_utils_time.Now().UnixMicro())
	})
	if err != nil {
		return nil, fmt.Errorf("subscribing to auth heartbeat: %w", err)
	}
	m.sub = sub

	go m.run(logger)
	return m, nil
}

// Immediately calls onMissed and signals the run loop to treat the
// monitor as being in the missed state. Use this from NATS disconnect
// handlers to eagerly degrade without waiting for the heartbeat timeout.
// When heartbeats resume, the run loop will detect recovery and call
// onRecovered normally.
func (m *HeartbeatMonitor) ForceMissed() {
	m.lastSeen.Store(0)
	m.forceDegraded.Store(true)
	m.onMissed()
}

func (m *HeartbeatMonitor) run(logger snx_lib_logging.Logger) {
	ticker := time.NewTicker(m.timeout / 3)
	defer ticker.Stop()

	missed := false
	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			if m.forceDegraded.CompareAndSwap(true, false) {
				missed = true
			}

			nowMicros := snx_lib_utils_time.Now().UnixMicro()
			age := time.Duration(nowMicros-m.lastSeen.Load()) * time.Microsecond
			if age >= m.timeout {
				if !missed {
					logger.Warn("Auth publisher heartbeat lost, degrading cache", "age", age)
					m.onMissed()
					missed = true
				}
			} else if missed {
				logger.Info("Auth publisher heartbeat recovered, restoring cache")
				m.onRecovered()
				missed = false
			}
		}
	}
}

// Drains the NATS subscription and stops the checker goroutine.
// Safe to call multiple times.
func (m *HeartbeatMonitor) Stop() {
	m.stopOnce.Do(func() {
		close(m.done)
		if m.sub != nil {
			_ = m.sub.Drain()
		}
	})
}
