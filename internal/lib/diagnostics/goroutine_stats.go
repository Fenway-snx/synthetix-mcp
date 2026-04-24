package diagnostics

import "runtime"

// A specific form of `ScalarLevelStats` that provides current/max count of
// the process goroutine count.
type GoroutineStats struct {
	stats ScalarLevelStats
}

// Loads, in a thread-safe manner, a copy of the called instance current/max
// counts.
func (gs *GoroutineStats) Load() (max, current int64) {
	return gs.stats.Load()
}

// Updates the instance counts with the number of goroutines that currently
// exist.
func (gs *GoroutineStats) UpdateFromRuntime() {
	n := int64(runtime.NumGoroutine())

	gs.stats.Set(n)
}
