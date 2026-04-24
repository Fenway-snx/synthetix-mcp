package concepts

import (
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

func Test_PrimaryTradeDirectionDisplayString(t *testing.T) {
	tests := []struct {
		dir  snx_lib_core.Direction
		want string
	}{
		{snx_lib_core.Direction_Long, "Open Long"},
		{snx_lib_core.Direction_Short, "Open Short"},
		{snx_lib_core.Direction_CloseLong, "Close Long"},
		{snx_lib_core.Direction_CloseShort, "Close Short"},
		{snx_lib_core.Direction(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, PrimaryTradeDirectionDisplayString(tt.dir))
		})
	}
}
