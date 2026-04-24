package state_manager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_runtime_health_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/health/types"
)

func Test_ServiceStates(t *testing.T) {

	manager := NewStateManager()
	t.Run("Zero value should be unknown", func(t *testing.T) {
		assert.Equal(t, snx_lib_runtime_health_types.ServiceState_Unknown, manager.GetServiceState(), "Invalid zero service state value.")
	})
}
