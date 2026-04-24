package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_runtime_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/types"
)

func newViperWith(kvs map[string]string) *viper.Viper {
	v := viper.New()
	for k, val := range kvs {
		v.Set(k, val)
	}
	return v
}

func Test_LoadDeploymentMode(t *testing.T) {
	tests := []struct {
		name        string
		keys        map[string]string
		wantMode    snx_lib_runtime_types.DeploymentMode
		wantWarning bool
		wantErr     bool
	}{
		// Primary key: "deployment_mode"
		{
			name:     "deployment_mode set to local",
			keys:     map[string]string{"deployment_mode": "local"},
			wantMode: snx_lib_runtime_types.DeploymentMode_Local,
		},
		{
			name:     "deployment_mode set to development",
			keys:     map[string]string{"deployment_mode": "development"},
			wantMode: snx_lib_runtime_types.DeploymentMode_Development,
		},
		{
			name:     "deployment_mode set to production",
			keys:     map[string]string{"deployment_mode": "production"},
			wantMode: snx_lib_runtime_types.DeploymentMode_Production,
		},
		{
			name:     "deployment_mode set to staging",
			keys:     map[string]string{"deployment_mode": "staging"},
			wantMode: snx_lib_runtime_types.DeploymentMode_Staging,
		},
		{
			name:     "deployment_mode accepts contraction dev",
			keys:     map[string]string{"deployment_mode": "dev"},
			wantMode: snx_lib_runtime_types.DeploymentMode_Development,
		},
		{
			name:     "deployment_mode accepts contraction prod",
			keys:     map[string]string{"deployment_mode": "prod"},
			wantMode: snx_lib_runtime_types.DeploymentMode_Production,
		},

		// Priority: deployment_mode wins over deprecated keys
		{
			name:     "deployment_mode wins over environment",
			keys:     map[string]string{"deployment_mode": "local", "environment": "production"},
			wantMode: snx_lib_runtime_types.DeploymentMode_Local,
		},

		// Deprecated fallback: "environment"
		{
			name:        "deprecated environment key",
			keys:        map[string]string{"environment": "dev"},
			wantMode:    snx_lib_runtime_types.DeploymentMode_Development,
			wantWarning: true,
		},
		{
			name:        "deprecated environment key with empty deployment_mode",
			keys:        map[string]string{"deployment_mode": "", "environment": "staging"},
			wantMode:    snx_lib_runtime_types.DeploymentMode_Staging,
			wantWarning: true,
		},

		// Error cases
		{
			name:    "no keys set",
			keys:    nil,
			wantErr: true,
		},
		{
			name:    "all keys empty",
			keys:    map[string]string{"deployment_mode": "", "environment": ""},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := newViperWith(tc.keys)

			dm, warning, err := LoadDeploymentMode(v)

			if tc.wantErr {
				require.ErrorIs(t, err, errDeploymentModeRequired)
				return
			}

			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tc.wantMode, dm, "deployment mode")

			if tc.wantWarning {
				assert.Contains(t, warning, "environment",
					"warning should name the deprecated key",
				)
				assert.Contains(t, warning, "deployment_mode",
					"warning should name the preferred key",
				)
			} else {
				assert.Empty(t, warning, "expected no warning for the primary key")
			}
		})
	}
}
