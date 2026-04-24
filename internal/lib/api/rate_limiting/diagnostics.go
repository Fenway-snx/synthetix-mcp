package ratelimiting

import "sync/atomic"

type RateLimitingDiagnostics struct {
	NumPermitted atomic.Int64 // T.B.C.
	NumRejected  atomic.Int64 // T.B.C.
}
