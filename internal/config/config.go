package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type Provider struct {
	Name              string `yaml:"name"`
	LocalBuildCommand string `yaml:"local_build_command,omitempty"`
	ProviderDirectory string `yaml:"provider_directory,omitempty"`
}

type Config struct {
	Provider Provider `yaml:"provider"`
}

var configDir string

func (c *Config) WriteConfig(filePath string) error {
	configDir = filePath

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func LoadConfig() (Config, error) {
	var cfg Config

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return cfg, fmt.Errorf("config directory does not exist: %s", configDir)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.SetDefault("snapshot_directory", filepath.Join(configDir, "snapshots"))

	if err := v.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("error reading config file: %w", err)
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("error parsing config: %w", err)
	}

	return cfg, nil
}
