package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_runtime_halt "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/halt"
	snx_lib_runtime_health_state_manager "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/state_manager"
	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
)

func Test_HaltMiddleware_REJECTS_WHEN_HALTING(t *testing.T) {
	e := echo.New()
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Draining)
	handler, err := snx_lib_runtime_halt.NewHandler(
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		snx_lib_runtime_halt.Config{
			AdminAPIKey: "test-key",
			ServiceId:   "test-service",
			Version:     "test",
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	inFlight := &atomic.Int64{}
	mw := HaltMiddleware(handler, inFlight, nil)
	next := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/trade", nil)
	responseRecorder := httptest.NewRecorder()
	context := e.NewContext(request, responseRecorder)
	require.NoError(t, next(context))

	assert.Equal(t, http.StatusServiceUnavailable, responseRecorder.Code)
	assert.EqualValues(t, 0, inFlight.Load())
}

func Test_HaltMiddleware_TRACKS_IN_FLIGHT_REQUESTS(t *testing.T) {
	e := echo.New()
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)
	handler, err := snx_lib_runtime_halt.NewHandler(
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		snx_lib_runtime_halt.Config{
			AdminAPIKey: "test-key",
			ServiceId:   "test-service",
			Version:     "test",
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	inFlight := &atomic.Int64{}
	requestStarted := make(chan struct{}, 1)
	releaseRequest := make(chan struct{})
	mw := HaltMiddleware(handler, inFlight, nil)
	next := mw(func(c echo.Context) error {
		requestStarted <- struct{}{}
		<-releaseRequest
		return c.NoContent(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/trade", nil)
	responseRecorder := httptest.NewRecorder()
	context := e.NewContext(request, responseRecorder)

	done := make(chan error, 1)
	go func() {
		done <- next(context)
	}()

	<-requestStarted
	assert.EqualValues(t, 1, inFlight.Load())
	close(releaseRequest)
	require.NoError(t, <-done)
	assert.EqualValues(t, 0, inFlight.Load())
}

func Test_WaitForNoInFlightRequests(t *testing.T) {
	inFlight := &atomic.Int64{}
	inFlight.Store(1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- WaitForNoInFlightRequests(ctx, inFlight)
	}()

	time.Sleep(20 * time.Millisecond)
	inFlight.Store(0)
	require.NoError(t, <-done)
}

func Test_WaitForNoInFlightRequests_CONTEXT_CANCEL(t *testing.T) {
	inFlight := &atomic.Int64{}
	inFlight.Store(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WaitForNoInFlightRequests(ctx, inFlight)
	require.Error(t, err)
}
