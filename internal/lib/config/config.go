package config

import (
	"fmt"
	"strings"

	viper_cast "github.com/spf13/cast"
	viper "github.com/spf13/viper"
)

// Reads an int64 config value or returns the fallback.
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

// Creates a configured Viper instance and attempts to read a config file.
func Load(prefix string) (*viper.Viper, error) {
	v := viper.New()

	normalizedPrefix := strings.ToLower(prefix) + "_"
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetEnvPrefix(prefix)
	v.SetConfigName("config")
	v.SetConfigType("env")
	v.AddConfigPath(".")

	// Try to read config file, but don't fail if it doesn't exist
	if err := v.ReadInConfig(); err != nil {
		// Only return error if it's not a "file not found" error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// File not found is OK - we'll use defaults and env vars
	} else {
		prefixedValues := map[string]any{}
		for key, value := range v.AllSettings() {
			normalizedKey := strings.ToLower(key)
			if trimmed, ok := strings.CutPrefix(normalizedKey, normalizedPrefix); ok {
				prefixedValues[trimmed] = value
			}
		}
		if len(prefixedValues) > 0 {
			if err := v.MergeConfigMap(prefixedValues); err != nil {
				return nil, fmt.Errorf("failed to normalize config prefix: %w", err)
			}
		}
	}

	return v, nil
}
