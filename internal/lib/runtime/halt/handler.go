package halt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_runtime_admin "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/admin"
	snx_lib_runtime_admin_jetstream_queues "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/admin/jetstream_queues"
	snx_lib_runtime_health_state_manager "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/state_manager"
	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	adminStatusDraining = "DRAINING"
	adminStatusError    = "ERROR"
	adminStatusIdle     = "IDLE"
	adminStatusRunning  = "RUNNING"
	adminStatusStarting = "STARTING"
	adminStatusStopped  = "STOPPED"
	adminStatusUnknown  = "UNKNOWN"

	defaultDrainTimeout          = 60 * time.Second
	responseDrainOperationFailed = "drain operation failed"
	responseStateTransitionError = "state transition failed"
	stateRoutePattern            = "/admin/state"

	errJetStreamQueuesCollect = "failed to collect jetstream queue depths"
)

var (
	errConfigAdminAPIKeyEmpty = errors.New("halt.Config: AdminAPIKey must not be empty — emergency halt would be unreachable")
	errStateChangeConflict    = errors.New("cannot change state while service is stopped")
	errStateChangeInvalidBody = errors.New("invalid state change request body")
	errStateChangeUnsupported = errors.New("unsupported state transition")
)

const maxCallerLength = 128

type transitionResult struct {
	err            error
	shouldShutdown bool
	statusCode     int
}

type DrainFunc func(ctx context.Context) error
type ResumeFunc func() error
type ShutdownFunc func()
type ServiceStateFunc func() map[string]any

type Config struct {
	AdminAPIKey  string
	DrainTimeout time.Duration
	ServiceId    string
	Version      string
}

type statePersistence interface {
	LoadTargetState(ctx context.Context) (TargetState, error)
	SaveTargetState(ctx context.Context, target TargetState) error
	ClearTargetState(ctx context.Context) error
}

type Handler struct {
	cfg            Config
	drainFn        DrainFunc
	drainStartUs   atomic.Int64
	drainErr       atomic.Value
	logger         snx_lib_logging.Logger
	mu             sync.Mutex
	persistence    statePersistence
	resumeFn       ResumeFunc
	serviceStateFn ServiceStateFunc
	shutdownFn     ShutdownFunc
	startTime      time.Time
	stateManager   *snx_lib_runtime_health_state_manager.StateManager
	targetState    atomic.Value

	drainCancel context.CancelFunc
	drainDone   chan struct{}
	drainSeq    uint64

	jetStreamQueues snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepthCollector
}

func NewHandler(
	logger snx_lib_logging.Logger,
	stateManager *snx_lib_runtime_health_state_manager.StateManager,
	cfg Config,
	drainFn DrainFunc,
	resumeFn ResumeFunc,
	shutdownFn ShutdownFunc,
	serviceStateFn ServiceStateFunc,
	persistence statePersistence,
	jetStreamQueues snx_lib_runtime_admin_jetstream_queues.JetStreamQueueDepthCollector,
) (*Handler, error) {
	if strings.TrimSpace(cfg.AdminAPIKey) == "" {
		return nil, errConfigAdminAPIKeyEmpty
	}

	if cfg.DrainTimeout <= 0 {
		cfg.DrainTimeout = defaultDrainTimeout
	}
	if drainFn == nil {
		drainFn = func(context.Context) error { return nil }
	}
	if resumeFn == nil {
		resumeFn = func() error { return nil }
	}
	if shutdownFn == nil {
		shutdownFn = func() {}
	}
	if serviceStateFn == nil {
		serviceStateFn = func() map[string]any { return map[string]any{} }
	}

	handler := &Handler{
		cfg:             cfg,
		drainFn:         drainFn,
		logger:          logger,
		persistence:     persistence,
		resumeFn:        resumeFn,
		serviceStateFn:  serviceStateFn,
		shutdownFn:      shutdownFn,
		startTime:       snx_lib_utils_time.Now(),
		stateManager:    stateManager,
		jetStreamQueues: jetStreamQueues,
	}

	handler.targetState.Store(TargetState_Running)
	handler.drainErr.Store("")

	return handler, nil
}

func (h *Handler) RegisterRoutes(server *snx_lib_runtime_admin.AdminHTTPServer) {
	authenticatedStateHandler := AdminAuthMiddleware(h.cfg.AdminAPIKey, h.stateRouteHandler())
	server.RegisterRoute(stateRoutePattern, authenticatedStateHandler)
}

// Applies a target state transition (IDLE, RUNNING, STOPPED) from an admin PUT request.
// Validates the request body, resolves the drain timeout, and delegates to applyTargetState.
func (h *Handler) handleSetState(w http.ResponseWriter, r *http.Request) {
	var request StateChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, errStateChangeInvalidBody.Error(), http.StatusBadRequest)
		return
	}
	if !isValidTargetState(request.Target) {
		http.Error(w, "invalid target state", http.StatusBadRequest)
		return
	}

	timeout := h.cfg.DrainTimeout
	if request.TimeoutSeconds > 0 {
		timeout = time.Duration(request.TimeoutSeconds) * time.Second
	}

	caller := callerFromRequest(r)

	h.logger.Debug("admin state change requested",
		"action", "SET_STATE",
		"caller", caller,
		"serviceId", h.cfg.ServiceId,
		"targetState", request.Target,
		"timeoutSeconds", int(timeout.Seconds()),
		"timestampUs", snx_lib_utils_time.Now().UnixMicro(),
	)

	envelope, statusCode, err := h.applyTargetState(r.Context(), request.Target, timeout, caller, false)
	if err != nil {
		h.logger.Error("admin state change failed",
			"caller", caller,
			"error", err,
			"serviceId", h.cfg.ServiceId,
			"statusCode", statusCode,
			"targetState", request.Target,
			"timestampUs", snx_lib_utils_time.Now().UnixMicro(),
		)
		http.Error(w, sanitizeErrorForResponse(err), statusCode)
		return
	}

	h.writeJSONResponse(w, envelope)
}

// Returns the current service state envelope as JSON for an admin GET request.
func (h *Handler) handleGetState(w http.ResponseWriter, r *http.Request) {
	envelope := h.buildEnvelope(r.Context())
	status := h.currentAdminStatus()
	target := h.getTargetState()

	h.logger.Debug("admin state read",
		"action", "GET_STATE",
		"caller", callerFromRequest(r),
		"serviceId", h.cfg.ServiceId,
		"status", status,
		"targetState", target,
		"timestampUs", snx_lib_utils_time.Now().UnixMicro(),
	)

	h.writeJSONResponse(w, envelope)
}

func (h *Handler) writeJSONResponse(w http.ResponseWriter, envelope StateEnvelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(envelope); encodeErr != nil {
		h.logger.Error("failed encoding state response", "error", encodeErr)
	}
}

func (h *Handler) IsHalting() bool {
	switch h.stateManager.GetServiceState() {
	case snx_lib_runtime_health_types.ServiceState_Draining,
		snx_lib_runtime_health_types.ServiceState_Idle,
		snx_lib_runtime_health_types.ServiceState_ShuttingDown:
		return true
	default:
		return false
	}
}

// Reads the Redis-persisted target state and, when it is IDLE or STOPPED,
// applies the transition and blocks until drain completes (or ctx expires).
// Returns nil when persistence is unconfigured or the persisted target is
// RUNNING / absent.
func (h *Handler) CheckPersistedState(ctx context.Context) error {
	if h.persistence == nil {
		return nil
	}

	target, err := h.persistence.LoadTargetState(ctx)
	if err != nil {
		return fmt.Errorf("loading persisted target state: %w", err)
	}
	if target == "" || target == TargetState_Running {
		return nil
	}

	_, _, applyErr := h.applyTargetState(
		ctx,
		target,
		h.cfg.DrainTimeout,
		"startup-persistence",
		true,
	)
	if applyErr != nil {
		return applyErr
	}

	if target == TargetState_Idle || target == TargetState_Stopped {
		return h.WaitForDrain(ctx)
	}
	return nil
}

// Blocks until the in-flight drain goroutine finishes or ctx is cancelled.
// Returns nil immediately when no drain is active. Drain completion takes
// priority over context cancellation to avoid a race when shutdownFn
// cancels the same context shortly after closing drainDone.
func (h *Handler) WaitForDrain(ctx context.Context) error {
	h.mu.Lock()
	done := h.drainDone
	h.mu.Unlock()

	if done == nil {
		return nil
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		select {
		case <-done:
			return nil
		default:
			return fmt.Errorf("waiting for drain to complete: %w", ctx.Err())
		}
	}
}

func (h *Handler) applyTargetState(
	ctx context.Context,
	target TargetState,
	timeout time.Duration,
	caller string,
	skipPersistence bool,
) (StateEnvelope, int, error) {
	h.mu.Lock()

	previousState := h.currentAdminStatus()
	currentState := h.stateManager.GetServiceState()
	if currentState == snx_lib_runtime_health_types.ServiceState_ShuttingDown {
		h.mu.Unlock()
		return StateEnvelope{}, http.StatusConflict, errStateChangeConflict
	}

	var result transitionResult

	switch currentState {
	case snx_lib_runtime_health_types.ServiceState_Healthy,
		snx_lib_runtime_health_types.ServiceState_Starting,
		snx_lib_runtime_health_types.ServiceState_Unhealthy,
		snx_lib_runtime_health_types.ServiceState_Unknown:
		result = h.transitionFromHealthyLocked(ctx, target, timeout, skipPersistence)
	case snx_lib_runtime_health_types.ServiceState_Draining:
		result = h.transitionFromDrainingLocked(ctx, target, timeout, skipPersistence)
	case snx_lib_runtime_health_types.ServiceState_Idle:
		result = h.transitionFromIdleLocked(ctx, target, skipPersistence)
	default:
		result = transitionResult{statusCode: http.StatusConflict, err: errStateChangeUnsupported}
	}

	statusCode := http.StatusOK
	if result.statusCode != 0 {
		statusCode = result.statusCode
	}
	if result.err != nil && statusCode == http.StatusOK {
		statusCode = http.StatusInternalServerError
	}

	newState := h.currentAdminStatus()
	h.mu.Unlock()

	envelope := h.buildEnvelope(ctx)

	if result.shouldShutdown {
		h.shutdownFn()
	}

	h.logger.Info("admin state transition",
		"action", "SET_TARGET_STATE",
		"caller", caller,
		"newState", newState,
		"previousState", previousState,
		"serviceId", h.cfg.ServiceId,
		"targetState", target,
		"timestampUs", snx_lib_utils_time.Now().UnixMicro(),
	)

	return envelope, statusCode, result.err
}

func (h *Handler) transitionFromHealthyLocked(
	ctx context.Context,
	target TargetState,
	timeout time.Duration,
	skipPersistence bool,
) transitionResult {
	switch target {
	case TargetState_Idle, TargetState_Stopped:
		if err := h.commitTargetStateLocked(ctx, target, skipPersistence); err != nil {
			return transitionResult{err: err}
		}
		h.startDrainLocked(target, timeout)
		return transitionResult{}
	case TargetState_Running:
		return transitionResult{err: h.commitTargetStateLocked(ctx, target, skipPersistence)}
	default:
		return transitionResult{statusCode: http.StatusConflict, err: errStateChangeUnsupported}
	}
}

func (h *Handler) transitionFromDrainingLocked(
	ctx context.Context,
	target TargetState,
	timeout time.Duration,
	skipPersistence bool,
) transitionResult {
	switch target {
	case TargetState_Idle, TargetState_Stopped:
		// Terminal target is updated while drain continues.
		// If a previous drain failed and no drain is active, retry.
		if err := h.commitTargetStateLocked(ctx, target, skipPersistence); err != nil {
			return transitionResult{err: err}
		}
		if h.drainCancel == nil {
			h.startDrainLocked(target, timeout)
		}
		return transitionResult{}
	case TargetState_Running:
		if err := h.resumeFn(); err != nil {
			return transitionResult{
				statusCode: http.StatusInternalServerError,
				err:        fmt.Errorf("resuming service from draining: %w", err),
			}
		}
		h.cancelDrainLocked()
		h.stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)
		h.drainErr.Store("")
		h.drainStartUs.Store(0)
		// Always update in-memory target to match the actual operational state.
		// Persistence may fail (e.g., Redis unavailable for clearing the stale
		// halt key), but the service IS running — report it honestly.
		h.targetState.Store(target)
		if !skipPersistence {
			if err := h.persistTargetState(ctx, target); err != nil {
				return transitionResult{err: err}
			}
		}
		return transitionResult{}
	default:
		return transitionResult{statusCode: http.StatusConflict, err: errStateChangeUnsupported}
	}
}

func (h *Handler) transitionFromIdleLocked(
	ctx context.Context,
	target TargetState,
	skipPersistence bool,
) transitionResult {
	switch target {
	case TargetState_Idle:
		return transitionResult{}
	case TargetState_Running:
		if err := h.resumeFn(); err != nil {
			return transitionResult{
				statusCode: http.StatusInternalServerError,
				err:        fmt.Errorf("resuming service from idle: %w", err),
			}
		}
		h.stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Healthy)
		h.targetState.Store(target)
		if !skipPersistence {
			if err := h.persistTargetState(ctx, target); err != nil {
				return transitionResult{err: err}
			}
		}
		return transitionResult{}
	case TargetState_Stopped:
		if err := h.commitTargetStateLocked(ctx, target, skipPersistence); err != nil {
			return transitionResult{err: err}
		}
		h.stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_ShuttingDown)
		return transitionResult{shouldShutdown: true}
	default:
		return transitionResult{statusCode: http.StatusConflict, err: errStateChangeUnsupported}
	}
}

func (h *Handler) commitTargetStateLocked(
	ctx context.Context,
	target TargetState,
	skipPersistence bool,
) error {
	if !skipPersistence {
		if err := h.persistTargetState(ctx, target); err != nil {
			return err
		}
	}

	h.targetState.Store(target)
	return nil
}

func (h *Handler) startDrainLocked(target TargetState, timeout time.Duration) {
	h.cancelDrainLocked()

	if timeout <= 0 {
		timeout = h.cfg.DrainTimeout
	}
	if timeout <= 0 {
		timeout = defaultDrainTimeout
	}

	now := snx_lib_utils_time.Now()
	h.drainErr.Store("")
	h.drainStartUs.Store(now.UnixMicro())
	h.stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Draining)

	drainCtx, drainCancel := context.WithTimeout(context.Background(), timeout)
	h.drainCancel = drainCancel
	h.drainDone = make(chan struct{})
	h.drainSeq++
	drainSeq := h.drainSeq

	go h.runDrain(drainCtx, drainSeq, target)
}

func (h *Handler) runDrain(ctx context.Context, drainSeq uint64, target TargetState) {
	drainErr := h.drainFn(ctx)

	h.mu.Lock()
	shouldShutdown := false
	cancelDrain := h.drainCancel
	doneCh := h.drainDone
	if drainSeq != h.drainSeq {
		h.mu.Unlock()
		return
	}

	h.drainCancel = nil
	if drainErr != nil {
		h.logger.Error("drain operation failed",
			"error", drainErr,
			"serviceId", h.cfg.ServiceId,
		)
		h.drainErr.Store(responseDrainOperationFailed)
		h.drainDone = nil
		h.mu.Unlock()
		if cancelDrain != nil {
			cancelDrain()
		}
		if doneCh != nil {
			close(doneCh)
		}
		return
	}
	h.drainErr.Store("")
	h.drainStartUs.Store(0)

	currentTarget := h.getTargetState()
	switch currentTarget {
	case TargetState_Running:
		h.drainDone = nil
		h.mu.Unlock()
		if cancelDrain != nil {
			cancelDrain()
		}
		if doneCh != nil {
			close(doneCh)
		}
		return
	case TargetState_Idle:
		h.stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)
	case TargetState_Stopped:
		h.stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_ShuttingDown)
		shouldShutdown = true
	default:
		// keep draining state unchanged for unknown target
	}

	h.drainDone = nil
	h.mu.Unlock()
	if cancelDrain != nil {
		cancelDrain()
	}
	if doneCh != nil {
		close(doneCh)
	}

	if shouldShutdown {
		h.shutdownFn()
	}
}

func (h *Handler) cancelDrainLocked() {
	h.drainSeq++
	if h.drainCancel != nil {
		h.drainCancel()
		h.drainCancel = nil
	}
	if h.drainDone != nil {
		close(h.drainDone)
		h.drainDone = nil
	}
}

func (h *Handler) persistTargetState(ctx context.Context, target TargetState) error {
	if h.persistence == nil {
		return nil
	}

	var err error
	if target == TargetState_Running {
		err = h.persistence.ClearTargetState(ctx)
	} else {
		err = h.persistence.SaveTargetState(ctx, target)
	}
	if err == nil {
		return nil
	}

	// Save (halt): Redis unavailable is tolerable — no stale key exists, so the
	// service restarts healthy. Any other save error is fatal.
	// Clear (resume): Redis unavailable means the stale halt key survives and
	// would auto-halt the service on next restart. Always propagate clear errors.
	if target != TargetState_Running && errors.Is(err, errStatePersistenceRedisUnavailable) {
		return nil
	}

	return fmt.Errorf("persisting target state: %w", err)
}

func (h *Handler) buildEnvelope(ctx context.Context) StateEnvelope {
	currentTime := snx_lib_utils_time.Now()
	status := h.currentAdminStatus()
	target := h.getTargetState()
	uptimeSeconds := int64(snx_lib_utils_time.Since(h.startTime) / time.Second)
	drainStartUs := h.drainStartUs.Load()

	drainDurationMs := int64(0)
	if drainStartUs > 0 {
		drainDurationMs = (currentTime.UnixMicro() - drainStartUs) / int64(time.Millisecond/time.Microsecond)
	}

	metrics := StateMetrics{
		DrainDurationMs: drainDurationMs,
		InFlightOps:     parseInFlightOps(h.stateManager.GetMetrics()),
	}

	if h.jetStreamQueues != nil {
		queues, err := h.jetStreamQueues.CollectJetStreamQueueDepths(ctx)
		if err != nil && len(queues) == 0 {
			h.logger.Error(errJetStreamQueuesCollect, "error", err)
		} else {
			if err != nil {
				h.logger.Error(errJetStreamQueuesCollect, "error", err)
			}
			metrics.JetStreamQueueDepths = &JetStreamQueueDepthsMetrics{
				CollectPartial: err != nil,
				Queues:         queues,
			}
		}
	}

	return StateEnvelope{
		Error: h.getDrainError(),
		Metadata: StateMetadata{
			ServiceId:     h.cfg.ServiceId,
			Status:        status,
			TargetState:   string(target),
			UptimeSeconds: uptimeSeconds,
			Version:       h.cfg.Version,
		},
		Metrics:         metrics,
		ServiceSpecific: h.serviceStateFn(),
	}
}

func (h *Handler) stateRouteHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.handleGetState(w, r)
		case http.MethodPut:
			h.handleSetState(w, r)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	})
}

func (h *Handler) currentAdminStatus() string {
	switch h.stateManager.GetServiceState() {
	case snx_lib_runtime_health_types.ServiceState_Draining:
		return adminStatusDraining
	case snx_lib_runtime_health_types.ServiceState_Healthy:
		return adminStatusRunning
	case snx_lib_runtime_health_types.ServiceState_Idle:
		return adminStatusIdle
	case snx_lib_runtime_health_types.ServiceState_ShuttingDown:
		return adminStatusStopped
	case snx_lib_runtime_health_types.ServiceState_Starting:
		return adminStatusStarting
	case snx_lib_runtime_health_types.ServiceState_Unhealthy:
		return adminStatusError
	default:
		return adminStatusUnknown
	}
}

func (h *Handler) getDrainError() string {
	value, ok := h.drainErr.Load().(string)
	if !ok {
		return ""
	}
	return value
}

func (h *Handler) getTargetState() TargetState {
	value, ok := h.targetState.Load().(TargetState)
	if !ok || !isValidTargetState(value) {
		return TargetState_Running
	}
	return value
}

func callerFromRequest(r *http.Request) string {
	caller := r.Header.Get("X-Admin-Caller")
	if caller == "" {
		return "unknown"
	}
	if len(caller) > maxCallerLength {
		return caller[:maxCallerLength]
	}
	return caller
}

// Returns a safe, non-leaking error message for HTTP responses.
// Known safe sentinel errors pass through; all others get a generic message.
func sanitizeErrorForResponse(err error) string {
	switch {
	case errors.Is(err, errStateChangeConflict):
		return errStateChangeConflict.Error()
	case errors.Is(err, errStateChangeUnsupported):
		return errStateChangeUnsupported.Error()
	default:
		return responseStateTransitionError
	}
}

func isValidTargetState(target TargetState) bool {
	switch target {
	case TargetState_Idle, TargetState_Running, TargetState_Stopped:
		return true
	default:
		return false
	}
}

func parseInFlightOps(metrics map[string]any) int64 {
	rawInFlightOps, exists := metrics["inFlightOps"]
	if !exists {
		return 0
	}

	switch value := rawInFlightOps.(type) {
	case int:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
