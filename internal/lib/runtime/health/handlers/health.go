package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
)

type HealthHandler struct {
	logger      snx_lib_logging.Logger
	stateReader snx_lib_runtime_health_types.IStateReader
}

func NewHealthHandler(
	logger snx_lib_logging.Logger,
	state_manager snx_lib_runtime_health_types.IStateReader,
) HealthHandler {
	return HealthHandler{
		logger:      logger,
		stateReader: state_manager,
	}
}

type HealthResponse struct {
	ServiceState snx_lib_runtime_health_types.ServiceState `json:"service_state"`
	Metrics      map[string]any                            `json:"metrics"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	state := h.stateReader.GetServiceState()
	metrics := h.stateReader.GetMetrics()

	rawData := HealthResponse{
		ServiceState: state,
		Metrics:      metrics,
	}

	parsedData, err := json.Marshal(rawData)
	if err != nil {
		h.logger.Warn("could not marshal state",
			"error", err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "something went wrong: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	_, err = w.Write(parsedData)
	if err != nil {
		h.logger.Error("something went wrong when sending the health response",
			"error", err,
		)
		return
	}
}
