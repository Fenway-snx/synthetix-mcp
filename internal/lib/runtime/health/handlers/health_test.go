package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_runtime_health_state_manager "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/state_manager"
	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
)

func Test_HealthHandler_RETURNS_200_FOR_DRAINING(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Draining)
	stateManager.SetMetric("inFlightOps", 3)

	handler := NewHealthHandler(snx_lib_logging_doubles.NewStubLogger(), stateManager)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	require.Equal(t, http.StatusOK, responseRecorder.Code)

	var response HealthResponse
	require.NoError(t, json.Unmarshal(responseRecorder.Body.Bytes(), &response))
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Draining, response.ServiceState)
	assert.EqualValues(t, 3, response.Metrics["inFlightOps"])
}

func Test_HealthHandler_RETURNS_200_FOR_IDLE(t *testing.T) {
	stateManager := snx_lib_runtime_health_state_manager.NewStateManager()
	stateManager.SetServiceState(snx_lib_runtime_health_types.ServiceState_Idle)
	stateManager.SetMetric("inFlightOps", 0)

	handler := NewHealthHandler(snx_lib_logging_doubles.NewStubLogger(), stateManager)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	require.Equal(t, http.StatusOK, responseRecorder.Code)

	var response HealthResponse
	require.NoError(t, json.Unmarshal(responseRecorder.Body.Bytes(), &response))
	assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Idle, response.ServiceState)
	assert.EqualValues(t, 0, response.Metrics["inFlightOps"])
}
