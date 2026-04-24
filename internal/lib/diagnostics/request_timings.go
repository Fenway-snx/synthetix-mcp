package diagnostics

import (
	"sync"
	"time"

	d "github.com/synesissoftware/Diagnosticism.Go"
)

// Thread-safe structure representing timings information for API requests.
type RequestTimings struct {
	mx sync.RWMutex // protects access to state
	dg d.DOOMGram   // doom-gram that captures timing statistics
	rc uint64       // request count
}

// Accessors

// Obtains the request count.
//
// NOTE: this call operates under the control of an internal RW-lock, and
// thus provides thread-safety as long as no contemporaneous calls are
// made to any of the following unsafe methods:
//   - PushEventUnsafe();
func (rt *RequestTimings) RequestCount() uint64 {

	rt.mx.RLock()
	defer rt.mx.RUnlock()

	return rt.requestCountUnsafe()
}
func (rt *RequestTimings) requestCountUnsafe() uint64 {
	return rt.rc
}

// Obtain a consistent (guarded) sample of the timing information.
func (rt *RequestTimings) SampleTimingInformation() (
	timing_strip string,
	request_count uint64,
) {

	rt.mx.RLock()
	defer rt.mx.RUnlock()

	timing_strip = rt.dg.ToStrip()
	request_count = rt.requestCountUnsafe()

	return
}

// Modifiers

// Pushes an event duration, updating the event count and the timings
// information.
//
// NOTE: this call operates under the control of an internal RW-lock, and
// thus provides thread-safety as long as no contemporaneous calls are
// made to any of the following unsafe methods:
//   - PushEventUnsafe();
func (rt *RequestTimings) PushEventSafe(duration time.Duration) uint64 {

	rt.mx.Lock()
	defer rt.mx.Unlock()

	return rt.PushEventUnsafe(duration)
}

// Pushes an event duration, updating the event count and the timings
// information.
//
// NOTE: this call operates without the control of an internal RW-lock, and
// thus violates thread-safety. It must only be called in circumstances
// that are unequivocally safe. If in doubt, call PushEventSafe().
func (rt *RequestTimings) PushEventUnsafe(duration time.Duration) uint64 {

	rt.dg.PushEventDuration(duration)

	rt.rc++

	return rt.rc
}
