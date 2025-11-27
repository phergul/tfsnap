package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

type SourceMapping struct {
	LocalSource    string `yaml:"local_source"`
	RegistrySource string `yaml:"registry_source"`
}

type Provider struct {
	Name              string        `yaml:"name"`
	LocalBuildCommand string        `yaml:"local_build_command,omitempty"`
	ProviderDirectory string        `yaml:"provider_directory,omitempty"`
	SourceMapping     SourceMapping `yaml:"source_mappings"`
}

type Config struct {
	WorkingDirectory  string   `yaml:"working_directory"`
	Provider          Provider `yaml:"provider"`
	SnapshotDirectory string   `yaml:"snapshot_directory"`
}

func (c *Config) WriteConfig(filePath string) error {
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

	cfgFile, err := buildConfigPath()
	if err != nil {
		return cfg, fmt.Errorf("failed to locate config file: %w", err)
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return cfg, fmt.Errorf("error reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("error parsing config yaml: %w", err)
	}

	// log.Printf("Loaded config from %s: %+v", cfgFile, cfg)
	return cfg, nil
}

func buildConfigPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		cfgPath := filepath.Join(dir, ".tfsnap", "config.yaml")
		if _, err := os.Stat(cfgPath); err == nil {
			return cfgPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("config file not found")
}
