package middleware

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"

	snx_lib_runtime_halt "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/halt"
)

const haltMessage = "Service is draining for deployment"

// Rejects new requests while preserving in-flight request accounting.
func HaltMiddleware(
	handler *snx_lib_runtime_halt.Handler,
	inFlightRequests *atomic.Int64,
	onCountChanged func(inFlight int64),
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if handler.IsHalting() {
				if onCountChanged != nil {
					onCountChanged(inFlightRequests.Load())
				}
				return c.JSON(http.StatusServiceUnavailable, map[string]string{
					"code":    "SERVICE_DRAINING",
					"message": haltMessage,
				})
			}

			current := inFlightRequests.Add(1)
			if onCountChanged != nil {
				onCountChanged(current)
			}
			defer func() {
				current = inFlightRequests.Add(-1)
				if onCountChanged != nil {
					onCountChanged(current)
				}
			}()

			return next(c)
		}
	}
}

// Waits until the in-flight request count reaches zero.
func WaitForNoInFlightRequests(
	ctx context.Context,
	inFlightRequests *atomic.Int64,
) error {
	return snx_lib_runtime_halt.WaitUntil(ctx, 10*time.Millisecond, func() (bool, error) {
		return inFlightRequests.Load() == 0, nil
	})
}
