package time

import "time"

// Obtains the time elapsed since tm.
//
// It is shorthand for `snx_lib_utils_time.Now().Sub(t)`.
//
// Note:
// This must always be used in order that any time provider hooking
// behaviour works consistently.
func Since(tm time.Time) time.Duration {

	return Now().Sub(tm)
}
