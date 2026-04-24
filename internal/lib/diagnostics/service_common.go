package diagnostics

import (
	"time"
)

// Common diagnostics state for services.
type ServiceCommon struct {
	HeartbeatCommon
	GoroutineStats
}

// -------------------------------------
// `HeartbeatCommon` methods
// -------------------------------------

// Invokes `HeartbeatCommon#GetHeartbeatInfo()` on `HeartbeatCommon`
// field.
func (sc *ServiceCommon) GetHeartbeatInfo() (count uint64, strip string) {
	return sc.HeartbeatCommon.GetCommonInfo()
}

// Invokes `HeartbeatCommon#OnHeartbeatCompletion()` on `HeartbeatCommon`
// field.
func (sc *ServiceCommon) OnHeartbeatCompletion(tm_start time.Time) {
	sc.HeartbeatCommon.OnHeartbeatCompletion(tm_start)
}

// -------------------------------------
// `GoroutineStats` methods
// -------------------------------------

// Invokes `ScalarLevelStats#Load()` on `GoroutineStats` field.
func (sc *ServiceCommon) LoadGoroutineCounts() (max, current int64) {
	return sc.GoroutineStats.Load()
}

// Obtains the current goroutine count from the Go runtime and invokes
// `ScalarLevelStats#Set()` on on `GoroutineStats` field.
func (sc *ServiceCommon) UpdateGoroutineCount() {
	sc.GoroutineStats.UpdateFromRuntime()
}

// Combined functionality of first `#UpdateGoroutineCount()` and then
// `#LoadGoroutineCounts()`.
func (sc *ServiceCommon) UpdateGoroutineCountAndLoad() (max, current int64) {
	sc.GoroutineStats.UpdateFromRuntime()

	return sc.GoroutineStats.Load()
}
