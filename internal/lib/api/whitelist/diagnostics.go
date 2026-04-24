package whitelist

import "sync/atomic"

type WhitelistDiagnostics struct {
	NumPermitted atomic.Int64 // T.B.C.
	NumRejected  atomic.Int64 // T.B.C.
}
