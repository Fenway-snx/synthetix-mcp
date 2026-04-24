package halt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_runtime_admin_jetstream_queues "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/admin/jetstream_queues"
	snx_lib_runtime_health_state_manager "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/state_manager"
	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
)

type mockPersistence struct {
	clearErr  error
	loadErr   error
	loadState TargetState
	saveErr   error
}

func (m *mockPersistence) LoadTargetState(context.Context) (TargetState, error) {
	return m.loadState, m.loadErr
}

func (m *mockPersistence) SaveTargetState(context.Context, TargetState) error {
	return m.saveErr
}

func (m *mockPersistence) ClearTargetState(context.Context) error {
	return m.clearErr
}

type oneTimeSaveFailurePersistence struct {
	saveCalls atomic.Int64
}

func (p *oneTimeSaveFailurePersistence) LoadTargetState(context.Context) (TargetState, error) {
	return "", nil
}

func (p *oneTimeSaveFailurePersistence) SaveTargetState(context.Context, TargetState) error {
	if p.saveCalls.Add(1) == 1 {
		return errors.New("transient save failure")
	}
	return nil
}

func (p *oneTimeSaveFailurePersistence) ClearTargetState(context.Context) error {
	return nil
}

// Simulates Redis available for writes but unavailable for clears.
// Saves persist into an in-memory field so LoadTargetState returns the stale key.
type clearFailsPersistence struct {
	state atomic.Value
}

func (p *clearFailsPersistence) LoadTargetState(context.Context) (TargetState, error) {
	v, _ := p.state.Load().(TargetState)
	return v, nil
}

func (p *clearFailsPersistence) SaveTargetState(_ context.Context, target TargetState) error {
	p.state.Store(target)
	return nil
}

func (p *clearFailsPersistence) ClearTargetState(context.Context) error {
	return errStatePersistenceRedisUnavailable
}

func mustNewHandler(
	t *testing.T,
	logger snx_lib_logging.Logger,
	stateManager *snx_lib_runtime_health_state_manager.StateManager,
	cfg Config,
	drainFn DrainFunc,
	resumeFn ResumeFunc,
	shutdownFn ShutdownFunc,
	serviceStateFn ServiceStateFunc,
	persistence statePersistence,
) *Handler {
	t.Helper()

	h, err := NewHandler(logger, stateManager, cfg, drainFn, resumeFn, shutdownFn, serviceStateFn, persistence, nil)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	return h
}

func Test_NewHandler_REJECTS_EMPTY_ADMIN_API_KEY(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "empty string", key: ""},
		{name: "whitespace only", key: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateManager := snx_lib_runtime_health_state_manager.NewStateManager()

			h, err := NewHandler(
				snx_lib_logging_doubles.NewStubLogger(),
				stateManager,
				Config{AdminAPIKey: tt.key, DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
				nil, nil, nil, nil, nil,
				nil,
			)
			require.ErrorIs(t, err, errConfigAdminAPIKeyEmpty)
			assert.Nil(t, h)
		})
	}
}

func Test_BuildEnvelopeJetStreamQueueDepths(t *testing.T) {
	ctx := context.Background()
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	cfg := Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"}

	t.Run("nil collector omits nested metrics", func(t *testing.T) {
		h, err := NewHandler(snx_lib_logging_doubles.NewStubLogger(), stateManager, cfg, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		env := h.buildEnvelope(ctx)
		assert.Nil(t, env.Metrics.JetStreamQueueDepths)
	})

	t.Run("success populates queues", func(t *testing.T) {
		collector := snx_lib_runtime_admin_jetstream_queues.FuncCollector(func(context.Context) ([]snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepth, error) {
			return []snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepth{
				{Stream: "s", Consumer: "c", Pending: 1, AckPending: 0},
			}, nil
		})
		h, err := NewHandler(snx_lib_logging_doubles.NewStubLogger(), stateManager, cfg, nil, nil, nil, nil, nil, collector)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		env := h.buildEnvelope(ctx)
		require.NotNil(t, env.Metrics.JetStreamQueueDepths)
		assert.False(t, env.Metrics.JetStreamQueueDepths.CollectPartial)
		require.Len(t, env.Metrics.JetStreamQueueDepths.Queues, 1)
		assert.Equal(t, "s", env.Metrics.JetStreamQueueDepths.Queues[0].Stream)
	})

	t.Run("partial failure sets collectPartial", func(t *testing.T) {
		collector := snx_lib_runtime_admin_jetstream_queues.FuncCollector(func(context.Context) ([]snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepth, error) {
			return []snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepth{
				{Stream: "s", Consumer: "c", Pending: 2, AckPending: 0},
			}, errors.New("partial")
		})
		h, err := NewHandler(snx_lib_logging_doubles.NewStubLogger(), stateManager, cfg, nil, nil, nil, nil, nil, collector)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		env := h.buildEnvelope(ctx)
		require.NotNil(t, env.Metrics.JetStreamQueueDepths)
		assert.True(t, env.Metrics.JetStreamQueueDepths.CollectPartial)
		require.Len(t, env.Metrics.JetStreamQueueDepths.Queues, 1)
	})

	t.Run("total failure omits nested metrics", func(t *testing.T) {
		collector := snx_lib_runtime_admin_jetstream_queues.FuncCollector(func(context.Context) ([]snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepth, error) {
			return nil, errors.New("total")
		})
		h, err := NewHandler(snx_lib_logging_doubles.NewStubLogger(), stateManager, cfg, nil, nil, nil, nil, nil, collector)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		env := h.buildEnvelope(ctx)
		assert.Nil(t, env.Metrics.JetStreamQueueDepths)
	})
}

func Test_AdminAuthMiddleware_EMPTY_KEY_REJECTS_ALL_REQUESTS(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not be called when API key is empty")
	})

	handler := AdminAuthMiddleware("", next)

	tests := []struct {
		name       string
		authHeader string
	}{
		{name: "no auth header", authHeader: ""},
		{name: "valid bearer format", authHeader: "Bearer some-token"},
		{name: "empty bearer token", authHeader: "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/admin/state", nil)
			if tt.authHeader != "" {
				request.Header.Set("Authorization", tt.authHeader)
			}
			responseRecorder := httptest.NewRecorder()
			handler.ServeHTTP(responseRecorder, request)
			assert.Equal(t, http.StatusUnauthorized, responseRecorder.Code)
		})
	}
}

func Test_AdminAuthMiddleware(t *testing.T) {
	nextCalled := atomic.Bool{}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled.Store(true)
	})

	handler := AdminAuthMiddleware("secret", next)

	request := httptest.NewRequest(http.MethodGet, "/admin/state", nil)
	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, request)
	assert.Equal(t, http.StatusUnauthorized, responseRecorder.Code)
	assert.False(t, nextCalled.Load())

	request = httptest.NewRequest(http.MethodGet, "/admin/state", nil)
	request.Header.Set("Authorization", "Bearer bad-secret")
	responseRecorder = httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, request)
	assert.Equal(t, http.StatusUnauthorized, responseRecorder.Code)
	assert.False(t, nextCalled.Load())

	request = httptest.NewRequest(http.MethodGet, "/admin/state", nil)
	request.Header.Set("Authorization", "Bearer secret")
	responseRecorder = httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, request)
	assert.Equal(t, http.StatusOK, responseRecorder.Code)
	assert.True(t, nextCalled.Load())
}

func Test_SetStateHandlerErrorResponsesDoNotLeakInternalDetails(t *testing.T) {
	t.Run("resume failure returns generic message", func(t *testing.T) {
		stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
		stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)

		handler := mustNewHandler(t,
			snx_lib_logging_doubles.NewStubLogger(),
			stateManager,
			Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
			nil,
			func() error {
				return errors.New("dial tcp 10.0.3.17:6379: connection refused")
			},
			nil,
			nil,
			nil,
		)
		handler.targetState.Store(TargetState_Idle)

		requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
		responseRecorder := httptest.NewRecorder()
		handler.handleSetState(responseRecorder, request)

		require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
		body := responseRecorder.Body.String()
		assert.Contains(t, body, "state transition failed")
		assert.NotContains(t, body, "dial tcp")
		assert.NotContains(t, body, "10.0.3.17")
		assert.NotContains(t, body, "connection refused")
		assert.NotContains(t, body, "resuming service")
	})

	t.Run("persistence failure returns generic message", func(t *testing.T) {
		stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
		stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

		handler := mustNewHandler(t,
			snx_lib_logging_doubles.NewStubLogger(),
			stateManager,
			Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
			nil,
			nil,
			nil,
			nil,
			&mockPersistence{saveErr: errors.New("dial tcp 10.0.3.17:6379: i/o timeout")},
		)

		requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
		responseRecorder := httptest.NewRecorder()
		handler.handleSetState(responseRecorder, request)

		require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
		body := responseRecorder.Body.String()
		assert.Contains(t, body, "state transition failed")
		assert.NotContains(t, body, "dial tcp")
		assert.NotContains(t, body, "10.0.3.17")
		assert.NotContains(t, body, "persisting target state")
		assert.NotContains(t, body, "redis")
	})

	t.Run("conflict returns safe constant message", func(t *testing.T) {
		stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
		stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_ShuttingDown)

		handler := mustNewHandler(t,
			snx_lib_logging_doubles.NewStubLogger(),
			stateManager,
			Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
			nil, nil, nil, nil, nil,
		)

		requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
		responseRecorder := httptest.NewRecorder()
		handler.handleSetState(responseRecorder, request)

		require.Equal(t, http.StatusConflict, responseRecorder.Code)
		body := responseRecorder.Body.String()
		assert.Contains(t, body, errStateChangeConflict.Error())
	})
}

func Test_DrainErrorInEnvelopeDoesNotLeakInternalDetails(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			return errors.New("write tcp 10.0.3.17:6379->10.0.3.18:52014: broken pipe")
		},
		nil,
		nil,
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)
	require.Equal(t, http.StatusOK, responseRecorder.Code)

	require.Eventually(
		t,
		func() bool { return handler.getDrainError() != "" },
		time.Second,
		10*time.Millisecond,
	)

	drainError := handler.getDrainError()
	assert.Equal(t, "drain operation failed", drainError)
	assert.NotContains(t, drainError, "write tcp")
	assert.NotContains(t, drainError, "10.0.3.17")
	assert.NotContains(t, drainError, "broken pipe")

	envelope := handler.buildEnvelope(context.Background())
	assert.Equal(t, "drain operation failed", envelope.Error)
}

func Test_ResumeReturnsErrorWhenClearTargetStateFails(t *testing.T) {
	t.Run("from Idle", func(t *testing.T) {
		stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
		stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)

		handler := mustNewHandler(t,
			snx_lib_logging_doubles.NewStubLogger(),
			stateManager,
			Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
			nil,
			func() error { return nil },
			nil,
			nil,
			&mockPersistence{clearErr: errStatePersistenceRedisUnavailable},
		)
		handler.targetState.Store(TargetState_Idle)

		requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
		responseRecorder := httptest.NewRecorder()
		handler.handleSetState(responseRecorder, request)

		require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	})

	t.Run("from Draining", func(t *testing.T) {
		stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
		stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Draining)

		handler := mustNewHandler(t,
			snx_lib_logging_doubles.NewStubLogger(),
			stateManager,
			Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
			func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
			func() error { return nil },
			nil,
			nil,
			&mockPersistence{clearErr: errStatePersistenceRedisUnavailable},
		)
		handler.targetState.Store(TargetState_Idle)

		requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
		responseRecorder := httptest.NewRecorder()
		handler.handleSetState(responseRecorder, request)

		require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	})
}

func Test_StaleHaltKeyAfterFailedClearCausesReHaltOnRestart(t *testing.T) {
	persistence := &clearFailsPersistence{}

	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		func() error { return nil },
		nil,
		nil,
		persistence,
	)

	// Halt to IDLE — save succeeds, key written to persistence.
	idleBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	idleRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(idleBody))
	idleResponse := httptest.NewRecorder()
	handler.handleSetState(idleResponse, idleRequest)
	require.Equal(t, http.StatusOK, idleResponse.Code)

	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Idle },
		time.Second,
		10*time.Millisecond,
	)

	// Resume to RUNNING — clear fails, stale key survives.
	runBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	runRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(runBody))
	runResponse := httptest.NewRecorder()
	handler.handleSetState(runResponse, runRequest)
	require.Equal(t, http.StatusInternalServerError, runResponse.Code)

	// Verify: stale key survived in persistence.
	staleTarget, loadErr := persistence.LoadTargetState(context.Background())
	require.NoError(t, loadErr)
	assert.Equal(t, TargetState_Idle, staleTarget)

	// Simulate restart: new handler with the same persistence.
	restartStateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	restartStateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	restartHandler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		restartStateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil,
		nil,
		nil,
		persistence,
	)

	err = restartHandler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// The stale key causes the restarted service to auto-halt.
	require.Eventually(
		t,
		func() bool {
			return restartStateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Idle
		},
		time.Second,
		10*time.Millisecond,
	)
	assert.Equal(t, TargetState_Idle, restartHandler.getTargetState())
}

func Test_HaltWithRedisUnavailableSaveIsNonFatal(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil,
		nil,
		nil,
		&mockPersistence{saveErr: errStatePersistenceRedisUnavailable},
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)

	require.Equal(t, http.StatusOK, responseRecorder.Code)
	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Idle },
		time.Second,
		10*time.Millisecond,
	)
}

func Test_ResumePersistenceFailureKeepsConsistentInMemoryState(t *testing.T) {
	t.Run("from Draining", func(t *testing.T) {
		stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
		stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

		releaseDrain := make(chan struct{})
		handler := mustNewHandler(t,
			snx_lib_logging_doubles.NewStubLogger(),
			stateManager,
			Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
			func(ctx context.Context) error {
				select {
				case <-releaseDrain:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
			func() error { return nil },
			nil,
			nil,
			&mockPersistence{clearErr: errors.New("redis connection reset")},
		)

		// Halt to IDLE.
		idleBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		idleRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(idleBody))
		idleResponse := httptest.NewRecorder()
		handler.handleSetState(idleResponse, idleRequest)
		require.Equal(t, http.StatusOK, idleResponse.Code)

		require.Eventually(
			t,
			func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Draining },
			time.Second,
			10*time.Millisecond,
		)

		// Resume to RUNNING — clear fails, but service is running.
		runBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		runRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(runBody))
		runResponse := httptest.NewRecorder()
		handler.handleSetState(runResponse, runRequest)

		require.Equal(t, http.StatusInternalServerError, runResponse.Code)
		// Service IS running — state must reflect that.
		assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
		assert.Equal(t, TargetState_Running, handler.getTargetState())

		// Envelope must be self-consistent: status and targetState both say RUNNING.
		envelope := handler.buildEnvelope(context.Background())
		assert.Equal(t, "RUNNING", envelope.Metadata.Status)
		assert.Equal(t, "RUNNING", envelope.Metadata.TargetState)

		close(releaseDrain)
	})

	t.Run("from Idle", func(t *testing.T) {
		stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
		stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)

		handler := mustNewHandler(t,
			snx_lib_logging_doubles.NewStubLogger(),
			stateManager,
			Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
			nil,
			func() error { return nil },
			nil,
			nil,
			&mockPersistence{clearErr: errors.New("redis connection reset")},
		)
		handler.targetState.Store(TargetState_Idle)

		runBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		runRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(runBody))
		runResponse := httptest.NewRecorder()
		handler.handleSetState(runResponse, runRequest)

		require.Equal(t, http.StatusInternalServerError, runResponse.Code)
		// Service IS running — state must reflect that.
		assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
		assert.Equal(t, TargetState_Running, handler.getTargetState())

		// Envelope must be self-consistent.
		envelope := handler.buildEnvelope(context.Background())
		assert.Equal(t, "RUNNING", envelope.Metadata.Status)
		assert.Equal(t, "RUNNING", envelope.Metadata.TargetState)
	})
}

func Test_SetStateHandlerRunningToIdle(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	releaseDrain := make(chan struct{})
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			select {
			case <-releaseDrain:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		nil,
		nil,
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	handler.handleSetState(responseRecorder, request)
	require.Equal(t, http.StatusOK, responseRecorder.Code)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())

	close(releaseDrain)
	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Idle },
		time.Second,
		10*time.Millisecond,
	)
}

func Test_SetStateHandlerRunningToStopped(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	shutdownCalled := atomic.Bool{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil,
		func() { shutdownCalled.Store(true) },
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)

	require.Equal(t, http.StatusOK, responseRecorder.Code)
	require.Eventually(
		t,
		func() bool {
			return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_ShuttingDown
		},
		time.Second,
		10*time.Millisecond,
	)
	assert.True(t, shutdownCalled.Load())
}

func Test_SetStateHandlerDrainingToRunningCancelsDrain(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	resumeCalls := atomic.Int64{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		func() error {
			resumeCalls.Add(1)
			return nil
		},
		nil,
		nil,
		nil,
	)

	idleBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	idleRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(idleBody))
	idleResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(idleResponseRecorder, idleRequest)
	require.Equal(t, http.StatusOK, idleResponseRecorder.Code)
	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Draining },
		time.Second,
		10*time.Millisecond,
	)

	runningBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	runningRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(runningBody))
	runningResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(runningResponseRecorder, runningRequest)
	require.Equal(t, http.StatusOK, runningResponseRecorder.Code)

	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Healthy },
		time.Second,
		10*time.Millisecond,
	)
	assert.EqualValues(t, 1, resumeCalls.Load())
}

func Test_SetStateHandlerRejectsWhenStopped(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_ShuttingDown)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)

	assert.Equal(t, http.StatusConflict, responseRecorder.Code)
}

func Test_SetStateHandlerDrainingToRunningResumeFailureKeepsTargetAndDrain(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	releaseDrain := make(chan struct{})
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			select {
			case <-releaseDrain:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		func() error {
			return errors.New("resume failed")
		},
		nil,
		nil,
		nil,
	)

	idleBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	idleRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(idleBody))
	idleResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(idleResponseRecorder, idleRequest)
	require.Equal(t, http.StatusOK, idleResponseRecorder.Code)
	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Draining },
		time.Second,
		10*time.Millisecond,
	)
	assert.Equal(t, TargetState_Idle, handler.getTargetState())

	runningBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	runningRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(runningBody))
	runningResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(runningResponseRecorder, runningRequest)
	require.Equal(t, http.StatusInternalServerError, runningResponseRecorder.Code)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Idle, handler.getTargetState())

	close(releaseDrain)
	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Idle },
		time.Second,
		10*time.Millisecond,
	)
}

func Test_SetStateHandlerIdleToRunningResumeFailureKeepsTargetAndState(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil,
		func() error {
			return errors.New("resume failed")
		},
		nil,
		nil,
		nil,
	)
	handler.targetState.Store(TargetState_Idle)

	runningBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Running})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	runningRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(runningBody))
	runningResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(runningResponseRecorder, runningRequest)
	require.Equal(t, http.StatusInternalServerError, runningResponseRecorder.Code)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Idle, handler.getTargetState())
}

func Test_SetStateHandlerRunningToStoppedDrainFailureDoesNotShutdown(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	shutdownCalled := atomic.Bool{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			return errors.New("drain failed")
		},
		nil,
		func() { shutdownCalled.Store(true) },
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)
	require.Equal(t, http.StatusOK, responseRecorder.Code)

	require.Eventually(
		t,
		func() bool { return handler.getDrainError() != "" },
		time.Second,
		10*time.Millisecond,
	)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Stopped, handler.getTargetState())
	assert.False(t, shutdownCalled.Load())
}

func Test_SetStateHandlerDrainingStoppedRetriesAfterDrainFailure(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	drainCalls := atomic.Int64{}
	shutdownCalled := atomic.Bool{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			if drainCalls.Add(1) == 1 {
				return errors.New("first drain failed")
			}
			return nil
		},
		nil,
		func() { shutdownCalled.Store(true) },
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	firstRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	firstResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(firstResponseRecorder, firstRequest)
	require.Equal(t, http.StatusOK, firstResponseRecorder.Code)
	require.Eventually(
		t,
		func() bool { return handler.getDrainError() != "" },
		time.Second,
		10*time.Millisecond,
	)
	assert.EqualValues(t, 1, drainCalls.Load())
	assert.False(t, shutdownCalled.Load())
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())

	secondRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	secondResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(secondResponseRecorder, secondRequest)
	require.Equal(t, http.StatusOK, secondResponseRecorder.Code)

	require.Eventually(
		t,
		func() bool {
			return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_ShuttingDown
		},
		time.Second,
		10*time.Millisecond,
	)
	assert.EqualValues(t, 2, drainCalls.Load())
	assert.True(t, shutdownCalled.Load())
}

func Test_SetStateHandlerRunningToStoppedPersistenceFailureDoesNotStartDrain(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	drainCalls := atomic.Int64{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			drainCalls.Add(1)
			return nil
		},
		nil,
		nil,
		nil,
		&mockPersistence{saveErr: errors.New("redis write failed")},
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)

	require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Running, handler.getTargetState())
	assert.EqualValues(t, 0, drainCalls.Load())
}

func Test_SetStateHandlerDrainingStoppedPersistenceFailureDoesNotRetryDrain(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Draining)

	drainCalls := atomic.Int64{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			drainCalls.Add(1)
			return nil
		},
		nil,
		nil,
		nil,
		&mockPersistence{saveErr: errors.New("redis write failed")},
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)

	require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Running, handler.getTargetState())
	assert.EqualValues(t, 0, drainCalls.Load())
}

func Test_SetStateHandlerIdleToStoppedPersistenceFailureKeepsIdleAndNoShutdown(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)

	shutdownCalled := atomic.Bool{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil,
		nil,
		func() { shutdownCalled.Store(true) },
		nil,
		&mockPersistence{saveErr: errors.New("redis write failed")},
	)
	handler.targetState.Store(TargetState_Idle)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)

	require.Equal(t, http.StatusInternalServerError, responseRecorder.Code)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Idle, handler.getTargetState())
	assert.False(t, shutdownCalled.Load())
}

func Test_SetStateHandlerIdleToStoppedPersistenceFailureThenRetrySucceeds(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)

	shutdownCalled := atomic.Bool{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil,
		nil,
		func() { shutdownCalled.Store(true) },
		nil,
		&oneTimeSaveFailurePersistence{},
	)
	handler.targetState.Store(TargetState_Idle)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	firstRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	firstResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(firstResponseRecorder, firstRequest)
	require.Equal(t, http.StatusInternalServerError, firstResponseRecorder.Code)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, stateManager.GetServiceState())
	assert.False(t, shutdownCalled.Load())

	secondRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	secondResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(secondResponseRecorder, secondRequest)
	require.Equal(t, http.StatusOK, secondResponseRecorder.Code)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_ShuttingDown, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Stopped, handler.getTargetState())
	assert.True(t, shutdownCalled.Load())
}

func Test_SetStateHandlerIdleToStoppedShutdownCallbackCanReenter(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)

	done := make(chan struct{}, 1)
	var handler *Handler
	shutdownFn := func() {
		_, _, _ = handler.applyTargetState(
			context.Background(),
			TargetState_Running,
			time.Second,
			"shutdown-callback",
			true,
		)
		done <- struct{}{}
	}

	handler = mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil,
		nil,
		shutdownFn,
		nil,
		nil,
	)
	handler.targetState.Store(TargetState_Idle)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)
	require.Equal(t, http.StatusOK, responseRecorder.Code)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown callback blocked (possible mutex deadlock)")
	}
}

func Test_RunDrainStoppedShutdownCallbackCanReenter(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	done := make(chan struct{}, 1)
	var handler *Handler
	shutdownFn := func() {
		_, _, _ = handler.applyTargetState(
			context.Background(),
			TargetState_Running,
			time.Second,
			"shutdown-callback",
			true,
		)
		done <- struct{}{}
	}

	handler = mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil,
		shutdownFn,
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Stopped})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)
	require.Equal(t, http.StatusOK, responseRecorder.Code)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown callback blocked in runDrain (possible mutex deadlock)")
	}
}

func Test_RunDrainCancelsContextOnSuccess(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()

	cancelCalls := atomic.Int64{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil,
		nil,
		nil,
		nil,
	)

	handler.mu.Lock()
	handler.drainSeq = 1
	handler.drainCancel = func() { cancelCalls.Add(1) }
	handler.mu.Unlock()

	handler.runDrain(context.Background(), 1, TargetState_Idle)

	assert.EqualValues(t, 1, cancelCalls.Load())
	assert.Nil(t, handler.drainCancel)
}

func Test_RunDrainCancelsContextOnError(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()

	cancelCalls := atomic.Int64{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return errors.New("drain failed") },
		nil,
		nil,
		nil,
		nil,
	)

	handler.mu.Lock()
	handler.drainSeq = 1
	handler.drainCancel = func() { cancelCalls.Add(1) }
	handler.mu.Unlock()

	handler.runDrain(context.Background(), 1, TargetState_Idle)

	assert.EqualValues(t, 1, cancelCalls.Load())
	assert.Nil(t, handler.drainCancel)
}

func Test_DrainTimeoutWithNonCooperativeDrainFunction(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	drainCalls := atomic.Int64{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: 50 * time.Millisecond, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			call := drainCalls.Add(1)
			if call == 1 {
				time.Sleep(200 * time.Millisecond)
				return ctx.Err()
			}
			return nil
		},
		nil,
		nil,
		nil,
		nil,
	)

	requestBody, err := json.Marshal(StateChangeRequest{Target: TargetState_Idle})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	request := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()
	handler.handleSetState(responseRecorder, request)
	require.Equal(t, http.StatusOK, responseRecorder.Code)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())

	require.Eventually(t,
		func() bool { return handler.getDrainError() != "" },
		time.Second,
		10*time.Millisecond,
	)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Idle, handler.getTargetState())
	assert.Equal(t, responseDrainOperationFailed, handler.getDrainError())

	retryRequest := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(requestBody))
	retryResponseRecorder := httptest.NewRecorder()
	handler.handleSetState(retryResponseRecorder, retryRequest)
	require.Equal(t, http.StatusOK, retryResponseRecorder.Code)

	require.Eventually(t,
		func() bool {
			return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Idle
		},
		time.Second,
		10*time.Millisecond,
	)
	assert.EqualValues(t, 2, drainCalls.Load())
	assert.Empty(t, handler.getDrainError())
}

func Test_CheckPersistedState_nil_PERSISTENCE_RETURNS_nil(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil, nil, nil, nil, nil,
	)

	err := handler.CheckPersistedState(context.Background())
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
}

func Test_CheckPersistedState_EMPTY_OR_RUNNING_IS_NO_OP(t *testing.T) {
	tests := []struct {
		name      string
		loadState TargetState
	}{
		{name: "empty persisted state", loadState: ""},
		{name: "RUNNING persisted state", loadState: TargetState_Running},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
			stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

			handler := mustNewHandler(t,
				snx_lib_logging_doubles.NewStubLogger(),
				stateManager,
				Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
				nil, nil, nil, nil,
				&mockPersistence{loadState: tt.loadState},
			)

			err := handler.CheckPersistedState(context.Background())
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
			assert.Equal(t, TargetState_Running, handler.getTargetState())
		})
	}
}

func Test_CheckPersistedState_LOAD_ERROR_RETURNS_ERROR(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil, nil, nil, nil,
		&mockPersistence{loadErr: errors.New("redis connection failed")},
	)

	err := handler.CheckPersistedState(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading persisted target state")
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
}

func Test_CheckPersistedState_IDLE_TRIGGERS_DRAIN(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	err := handler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Eventually(
		t,
		func() bool { return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_Idle },
		time.Second,
		10*time.Millisecond,
	)
	assert.Equal(t, TargetState_Idle, handler.getTargetState())
}

func Test_CheckPersistedState_STOPPED_TRIGGERS_DRAIN_AND_SHUTDOWN(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	shutdownCalled := atomic.Bool{}
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil,
		func() { shutdownCalled.Store(true) },
		nil,
		&mockPersistence{loadState: TargetState_Stopped},
	)

	err := handler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// runDrain sets ShuttingDown under mu then unlocks before calling shutdownFn; wait for both.
	require.Eventually(
		t,
		func() bool {
			return stateManager.GetServiceState() == snx_lib_runtime_health_types.ServiceState_ShuttingDown &&
				shutdownCalled.Load()
		},
		time.Second,
		10*time.Millisecond,
	)
	assert.Equal(t, TargetState_Stopped, handler.getTargetState())
}

func Test_CheckPersistedState_RETURNS_CONFLICT_WHEN_SHUTTING_DOWN(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_ShuttingDown)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil, nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	err := handler.CheckPersistedState(context.Background())
	assert.ErrorIs(t, err, errStateChangeConflict)
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_ShuttingDown, stateManager.GetServiceState())
}

func Test_CheckPersistedState_IDLE_BLOCKS_UNTIL_DRAIN_COMPLETES(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	drainStarted := make(chan struct{})
	drainRelease := make(chan struct{})

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: 5 * time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			close(drainStarted)
			<-drainRelease
			return nil
		},
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	done := make(chan error, 1)
	go func() {
		done <- handler.CheckPersistedState(context.Background())
	}()

	<-drainStarted

	select {
	case <-done:
		t.Fatal("CheckPersistedState returned before drain completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(drainRelease)

	select {
	case err := <-done:
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	case <-time.After(time.Second):
		t.Fatal("CheckPersistedState did not return after drain completed")
	}

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, stateManager.GetServiceState())
}

func Test_CheckPersistedState_STOPPED_BLOCKS_UNTIL_DRAIN_AND_SHUTDOWN(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	drainStarted := make(chan struct{})
	drainRelease := make(chan struct{})
	shutdownCalled := atomic.Bool{}

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: 5 * time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			close(drainStarted)
			<-drainRelease
			return nil
		},
		nil,
		func() { shutdownCalled.Store(true) },
		nil,
		&mockPersistence{loadState: TargetState_Stopped},
	)

	done := make(chan error, 1)
	go func() {
		done <- handler.CheckPersistedState(context.Background())
	}()

	<-drainStarted

	select {
	case <-done:
		t.Fatal("CheckPersistedState returned before drain completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(drainRelease)

	select {
	case err := <-done:
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	case <-time.After(time.Second):
		t.Fatal("CheckPersistedState did not return after drain completed")
	}

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_ShuttingDown, stateManager.GetServiceState())
	assert.True(t, shutdownCalled.Load())
}

func Test_CheckPersistedState_DRAIN_ERROR_STILL_UNBLOCKS(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error {
			return errors.New("drain failed")
		},
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	err := handler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState())
	assert.Equal(t, responseDrainOperationFailed, handler.getDrainError())
}

func Test_CheckPersistedState_FROM_STARTING_STATE(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Starting)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	err := handler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Idle, handler.getTargetState())
}

func Test_CheckPersistedState_FROM_UNHEALTHY_STATE(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Unhealthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	err := handler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Idle, handler.getTargetState())
}

func Test_CheckPersistedState_FROM_UNKNOWN_STATE(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	err := handler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Idle, handler.getTargetState())
}

func Test_WaitForDrain_RETURNS_IMMEDIATELY_WHEN_NO_DRAIN(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil, nil, nil, nil, nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := handler.WaitForDrain(ctx)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_WaitForDrain_RESPECTS_CONTEXT_CANCELLATION(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: 10 * time.Second, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := handler.CheckPersistedState(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "waiting for drain to complete")
}

func Test_IsHalting_TRUE_FOR_SHUTTING_DOWN(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	shutdownCalled := make(chan struct{})
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(context.Context) error { return nil },
		nil,
		func() { close(shutdownCalled) },
		nil,
		&mockPersistence{loadState: TargetState_Stopped},
	)

	err := handler.CheckPersistedState(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	<-shutdownCalled

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_ShuttingDown, stateManager.GetServiceState())
	assert.True(t, handler.IsHalting(), "IsHalting must return true when state is ShuttingDown")
}

func Test_WaitForDrain_RETURNS_NIL_WHEN_DONE_CLOSED_BEFORE_CTX_CHECK(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil, nil, nil, nil, nil,
	)

	doneCh := make(chan struct{})
	close(doneCh)

	handler.mu.Lock()
	handler.drainDone = doneCh
	handler.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := handler.WaitForDrain(ctx)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CancelDrain_CLOSES_DRAIN_DONE_CHANNEL(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	drainStarted := make(chan struct{})
	drainBlock := make(chan struct{})

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: 5 * time.Second, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			close(drainStarted)
			<-drainBlock
			return nil
		},
		nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- handler.CheckPersistedState(context.Background())
	}()

	<-drainStarted

	setStateBody := StateChangeRequest{Target: TargetState_Running}
	body, _ := json.Marshal(setStateBody)
	req := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()
	handler.stateRouteHandler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	close(drainBlock)

	select {
	case err := <-waitDone:
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	case <-time.After(time.Second):
		t.Fatal("WaitForDrain did not unblock after cancel via resume")
	}

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
}

func Test_RunDrain_TARGET_CHANGED_TO_RUNNING_DURING_DRAIN(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)

	drainStarted := make(chan struct{})
	drainRelease := make(chan struct{})

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: 5 * time.Second, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			close(drainStarted)
			<-drainRelease
			return nil
		},
		func() error { return nil },
		nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- handler.CheckPersistedState(context.Background())
	}()

	<-drainStarted

	setStateBody := StateChangeRequest{Target: TargetState_Running}
	body, _ := json.Marshal(setStateBody)
	req := httptest.NewRequest(http.MethodPut, "/admin/state", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()
	handler.stateRouteHandler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	close(drainRelease)

	select {
	case err := <-waitDone:
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	case <-time.After(time.Second):
		t.Fatal("WaitForDrain did not unblock after target changed to RUNNING during drain")
	}

	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Healthy, stateManager.GetServiceState())
	assert.Equal(t, TargetState_Running, handler.getTargetState())
}

func Test_CheckPersistedState_APPLY_ERROR_RETURNS_WITHOUT_WAITING(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_ShuttingDown)

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil, nil, nil, nil,
		&mockPersistence{loadState: TargetState_Idle},
	)

	err := handler.CheckPersistedState(context.Background())
	assert.ErrorIs(t, err, errStateChangeConflict)
}

func Test_RunDrain_STALE_SEQUENCE_IGNORED(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Draining)

	drainCalls := atomic.Int64{}

	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		func(ctx context.Context) error {
			drainCalls.Add(1)
			return nil
		},
		nil, nil, nil, nil,
	)

	handler.targetState.Store(TargetState_Idle)
	handler.runDrain(context.Background(), 999, TargetState_Idle)

	assert.EqualValues(t, 1, drainCalls.Load())
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, stateManager.GetServiceState(),
		"stale drain should not have changed state to Idle")
}

func Test_WaitForDrain_CANCELLED_CTX_WITH_CLOSED_DONE(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	handler := mustNewHandler(t,
		snx_lib_logging_doubles.NewStubLogger(),
		stateManager,
		Config{AdminAPIKey: "test-key", DrainTimeout: time.Second, ServiceId: "test-service", Version: "test"},
		nil, nil, nil, nil, nil,
	)

	doneCh := make(chan struct{})
	handler.mu.Lock()
	handler.drainDone = doneCh
	handler.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	close(doneCh)

	err := handler.WaitForDrain(ctx)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}
