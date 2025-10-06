package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Provider struct {
		Name              string `yaml:"name"`
		LocalBuildCommand string `yaml:"local_build_command"`
	} `yaml:"provider"`
	SnapshotDirectory string `yaml:"snapshot_directory"`
}

func InitConfig() (Config, error) {
	var cfg Config

	workingDir, err := os.Getwd()
	if err != nil {
		return cfg, fmt.Errorf("failed to get working directory: %w", err)
	}

	configDir := filepath.Join(workingDir, ".tfsnap")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	viper.SetDefault("snapshot_directory", filepath.Join(configDir, "snapshots"))

	if err := viper.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("error reading config file: %w", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("error parsing config: %w", err)
	}

	if cfg.Provider.Name == "" {
		return cfg, fmt.Errorf("provider name is not set in config")
	}

	return cfg, nil
}
