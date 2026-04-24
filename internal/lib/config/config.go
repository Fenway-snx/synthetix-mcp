package config

import (
	"fmt"
	"os"
	"strings"

	viper_cast "github.com/spf13/cast"
	viper "github.com/spf13/viper"

	snx_lib_runtime_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/types"
)

var (
	deploymentModeEnvironmentFallback = []string{
		"deployment_mode",
		"environment",
	}

	errDeploymentModeRequired = fmt.Errorf(`deployment mode is required: set "%s" (or "%s")`,
		deploymentModeEnvironmentFallback[0],
		deploymentModeEnvironmentFallback[1],
	)
)

// T.B.C.
//
// Preconditions:
//   - `v != nil`
func GetInt64OrDefault(v *viper.Viper, key string, defaultValue int64) int64 {
	value := v.Get(key)

	if value == nil {
		return defaultValue
	} else {
		if i, err := viper_cast.ToInt64E(value); err != nil {
			return defaultValue
		} else {
			return i
		}
	}
}

// Load creates a new Viper instance configured with the given prefix. It
// sets up environment variable support and attempts to read from a config
// file. The caller is responsible for setting defaults before or after
// calling this function.
func Load(prefix string) (*viper.Viper, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetEnvPrefix(prefix)
	v.SetConfigName("config")
	v.SetConfigType("env")
	v.AddConfigPath("./config")

	// Try to read config file, but don't fail if it doesn't exist
	if err := v.ReadInConfig(); err != nil {
		// Only return error if it's not a "file not found" error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// File not found is OK - we'll use defaults and env vars
	}

	return v, nil
}

// Reads the deployment mode from the Viper instance, checking supported
// strings in priority order. The returned warning is non-empty when a
// deprecated fallback key resolved the mode; callers should log it once a
// logger is available. Returns an error when no recognised value is found.
func LoadDeploymentMode(v *viper.Viper) (
	r snx_lib_runtime_types.DeploymentMode,
	warning string,
	err error,
) {
	values := make([]string, len(deploymentModeEnvironmentFallback))
	for i, key := range deploymentModeEnvironmentFallback {
		values[i] = v.GetString(key)
	}

	var index int
	r, index = snx_lib_runtime_types.ParseDeploymentMode(values)
	if r == snx_lib_runtime_types.DeploymentMode_Unknown {
		err = errDeploymentModeRequired

		return
	}

	if index > 0 {
		warning = fmt.Sprintf(
			"deprecated config key %q used for deployment mode, but %q preferred",
			deploymentModeEnvironmentFallback[index],
			deploymentModeEnvironmentFallback[0],
		)

		// TODO: this should be logged properly, but rn we have complex/messy
		// code paths. For now, we will just write to stderr
		fmt.Fprintf(os.Stderr, "[WARN] %s\n", warning)
	}

	return
}
