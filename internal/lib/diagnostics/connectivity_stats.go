package diagnostics

import (
	"sync/atomic"
)

// State for connectivity statistics, although can be used for other counted
// quantities that have a start and a stop, or an addition and a removal.
type ConnectivityStats struct {
	total   int64
	max     int64
	current int64
}

// Loads, in a thread-safe manner, a copy of the called instance.
func (cs *ConnectivityStats) Load() (total, max, current int64) {
	total = atomic.LoadInt64(&cs.total)
	max = atomic.LoadInt64(&cs.max)
	current = atomic.LoadInt64(&cs.current)

	return
}

// Called by a containing instance when a subscription is established.
func (cs *ConnectivityStats) OnSubscribe() {
	atomic.AddInt64(&cs.total, 1)
	new_current := atomic.AddInt64(&cs.current, 1)

	// NOTE: when updating cs.max, we face a race condition, because other
	// task(s) could be doing the same thing, so we do a hot loop. The
	// likelihood of this running more than even once is very small.

	for {

		// Over time, cs.current rises and falls, but we concern outselves only
		// with the value as we know it (new_current).
		//
		// During times of contention, which operates minutely, we may be
		// attempting to update cs.max when current is dropping, because oher
		// tasks may have completed. However, a lower new_current in the purview
		// of another task does not invalidate the intended cs.max obtained from
		// the value of new_current in this task.
		//
		// The question is: How do we update with our "new max" (which is our
		// current) without overwriting a higher max from another task?

		curr_max := atomic.LoadInt64(&cs.max)

		if curr_max >= new_current {

			// Another task has updated GE than we wish to do, so we exit.
			break
		} else {

			// We are now in a potential race to update with 0+ other task(s), so
			// we effect an atomic update iff the current value of cs.max has not
			// changed since we obtained it above, and with which we compared to
			// our current.

			if atomic.CompareAndSwapInt64(&cs.max, curr_max, new_current) {

				// We successfully updated when the current max had not changed.
				break
			} else {

				// Another task has updated cs.max between our taking its value and
				// attempting its update, so we need to loop around and make another
				// attempt.
			}
		}
	}
}

// Called by a containing instance when a subscription is disestablished.
func (cs *ConnectivityStats) OnUnsubscribe() {
	atomic.AddInt64(&cs.current, -1)
}
