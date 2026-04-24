package diagnostics

import "sync/atomic"

// T.B.C.
type ScalarLevelStats struct {
	max     int64
	current int64
}

// Loads, in a thread-safe manner, a copy of the called instance current/max
// counts.
func (sls *ScalarLevelStats) Load() (max, current int64) {
	max = atomic.LoadInt64(&sls.max)
	current = atomic.LoadInt64(&sls.current)

	return
}

// Decrements the current count.
//
// Preconditions:
//   - current value > 0;
func (sls *ScalarLevelStats) Dec() {
	atomic.AddInt64(&sls.current, -1)
}

// Increments the current count and, possibly, the maximum count.
//
// Note:
// This function _will_ change the current value and _can_ change the max
// value in the case that the new current exceeds the contemporaneously
// known max value. As such, this provide eventual consistency.
func (sls *ScalarLevelStats) Inc() {
	newCurrent := atomic.AddInt64(&sls.current, +1)

	sls.updateMaxFromCurrent(newCurrent)
}

// Sets the current level and, potentially, adjusts the max level.
//
// Note:
// This function _will_ change the current value and _can_ change the max
// value in the case that the new current exceeds the contemporaneously
// known max value. As such, this provide eventual consistency.
func (sls *ScalarLevelStats) Set(level int64) {
	atomic.StoreInt64(&sls.current, level)

	newCurrent := level

	sls.updateMaxFromCurrent(newCurrent)
}

func (sls *ScalarLevelStats) updateMaxFromCurrent(newCurrent int64) {

	// NOTE: when updating sls.max, we face a race condition, because other
	// task(s) could be doing the same thing, so we do a hot loop. The
	// likelihood of this running more than even once is very small.

	for {

		// Over time, sls.current rises and falls, but we concern outselves only
		// with the value as we know it (newCurrent).
		//
		// During times of contention, which operates minutely, we may be
		// attempting to update sls.max when current is dropping, because oher
		// tasks may have completed. However, a lower newCurrent in the purview
		// of another task does not invalidate the intended sls.max obtained
		// from the value of newCurrent in this task.
		//
		// The question is: How do we update with our "new max" (which is our
		// current) without overwriting a higher max from another task?

		curr_max := atomic.LoadInt64(&sls.max)

		if curr_max >= newCurrent {

			// Another task has updated GE than we wish to do, so we exit.
			break
		} else {

			// We are now in a potential race to update with 0+ other task(s), so
			// we effect an atomic update iff the current value of sls.max has not
			// changed since we obtained it above, and with which we compared to
			// our current.

			if atomic.CompareAndSwapInt64(&sls.max, curr_max, newCurrent) {

				// We successfully updated when the current max had not changed.
				break
			} else {

				// Another task has updated sls.max between our taking its value and
				// attempting its update, so we need to loop around and make another
				// attempt.
			}
		}
	}
}
