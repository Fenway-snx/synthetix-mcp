package state_manager

import (
	"maps"
	"sync"

	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
)

type StateManager struct {
	serviceState snx_lib_runtime_health_types.ServiceState
	metrics      map[string]any
	mu           sync.RWMutex
}

func NewStateManager() *StateManager {
	return &StateManager{
		metrics: make(map[string]any),
	}
}

func (s *StateManager) SetServiceState(newServiceState snx_lib_runtime_health_types.ServiceState) {
	s.mu.Lock()
	s.serviceState = newServiceState
	s.mu.Unlock()
}

func (s *StateManager) GetServiceState() snx_lib_runtime_health_types.ServiceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serviceState
}

func (s *StateManager) SetMetric(key string, value any) (exists bool, previousValue any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previousValue, exists = s.metrics[key]

	s.metrics[key] = value

	return exists, previousValue
}

func (s *StateManager) GetMetrics() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return maps.Clone(s.metrics)
}
