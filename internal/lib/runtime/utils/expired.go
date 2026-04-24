package utils

import (
	"context"
	"errors"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Reports whether a context's deadline has been exceeded. Only returns true
// for deadline expiration, not for generic cancellation (e.g. client disconnect
// or service shutdown), so that timeout metrics are not inflated by
// cancellations. The wall-clock check closes a narrow race where the deadline
// has passed but Go's internal timer goroutine has not yet fired.
func Expired(ctx context.Context) bool {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return true
	}
	deadline, ok := ctx.Deadline()
	return ok && !snx_lib_utils_time.Now().Before(deadline)
}
