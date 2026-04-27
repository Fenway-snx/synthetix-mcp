package time

import "time"

// Interface implemented by types that can provide a notionally "current"
// time.
//
// The primary purpose of abstracting current time retrieval behind this
// interface is to facilitate deterministic testing of time-related
// functionality.
type TimeProvider interface {
	Now() time.Time
}
