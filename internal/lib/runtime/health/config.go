package health

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port     int    // config: "port"
	Endpoint string // config: "endpoint"
}

func LoadConfig(v *viper.Viper) (*Config, error) {
	prefix := "health"

	cfg := &Config{
		Port:     v.GetInt(prefix + ".port"),
		Endpoint: v.GetString(prefix + ".endpoint"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var errors []string

	/*
		TODO: Add Proper PORT validation
		"Valid port numbers range from 0 to 65535, but are divided into three ranges: Well-Known Ports (0-1023),
		Registered Ports (1024-49151), and Dynamic/Private Ports (49152-65535).
		While all numbers in this range are technically valid, well-known ports are typically reserved for system services,
		while the other ranges are used for applications and temporary connections"
	*/
	if c.Port < 1 {
		errors = append(errors, "invalid health port")
	}

	// TODO: Add Proper PORT validation
	if c.Endpoint == "" {
		errors = append(errors, "invalid health endpoint")
	}

	if len(errors) > 0 {
		return fmt.Errorf("health api config validation failed: %s", strings.Join(errors, ";\n"))
	}

	return nil
}
