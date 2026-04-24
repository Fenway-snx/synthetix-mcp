package metrics

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Default configuration values
const (
	DefaultMetricsPort          = 9090
	DefaultBlockProfileRate     = 1 // Sample all blocking events when pprof is enabled
	DefaultMutexProfileFraction = 1 // Sample all mutex events when pprof is enabled
)

// Holds metrics and pprof configuration.
type Config struct {
	// Port is the HTTP port for the metrics server
	Port int // config: "port"

	// PprofEnabled enables pprof debug endpoints on the metrics server
	PprofEnabled bool // config: "pprof_enabled"

	// BlockProfileRate is the block profile rate (0 = disabled, 1 = all events)
	// Only used when PprofEnabled is true
	BlockProfileRate int // config: "block_profile_rate"

	// MutexProfileFraction is the mutex profile fraction (0 = disabled, 1 = all events)
	// Only used when PprofEnabled is true
	MutexProfileFraction int // config: "mutex_profile_fraction"
}

// LoadConfig loads metrics configuration from viper
func LoadConfig(v *viper.Viper) (*Config, error) {
	keyPrefix := "metrics"

	cfg := &Config{
		BlockProfileRate:     v.GetInt(keyPrefix + ".block_profile_rate"),
		MutexProfileFraction: v.GetInt(keyPrefix + ".mutex_profile_fraction"),
		Port:                 v.GetInt(keyPrefix + ".port"),
		PprofEnabled:         v.GetBool(keyPrefix + ".pprof_enabled"),
	}

	// Apply defaults
	if cfg.Port == 0 {
		cfg.Port = DefaultMetricsPort
	}

	// Apply pprof defaults only when pprof is enabled
	if cfg.PprofEnabled {
		if cfg.BlockProfileRate == 0 {
			cfg.BlockProfileRate = DefaultBlockProfileRate
		}
		if cfg.MutexProfileFraction == 0 {
			cfg.MutexProfileFraction = DefaultMutexProfileFraction
		}
	}

	// Validate the configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid metrics config: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var errors []string

	if c.Port < 1 || c.Port > 65535 {
		errors = append(errors, "metrics port must be between 1 and 65535")
	}

	if len(errors) > 0 {
		return fmt.Errorf("metrics config validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}
