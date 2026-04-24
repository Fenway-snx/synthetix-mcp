package admin

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port int `mapstructure:"port"`
}

func LoadConfig(v *viper.Viper) (*Config, error) {
	cfg := &Config{
		Port: v.GetInt("admin.port"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var validationErrors []string

	if c.Port < 1 {
		validationErrors = append(validationErrors, "invalid admin port")
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf(
			"admin config validation failed: %s",
			strings.Join(validationErrors, ";\n"),
		)
	}

	return nil
}
