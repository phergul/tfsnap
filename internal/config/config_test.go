package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".tfsnap")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `
config_path: /tmp/.tfsnap/config.yaml
working_directory: /tmp/work
provider:
  name: test
  local_build_command: ""
  provider_directory: ""
  source_mappings:
    local_source: local/test
    registry_source: registry/test
snapshot_directory: /tmp/snapshots
working_strategy: ""
example_client_type: registry
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Provider.Name != "test" {
		t.Errorf("Expected provider name 'test', got %q", cfg.Provider.Name)
	}
}

func TestLoadConfigInvalidPath(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadConfig should return error for nonexistent file")
	}
}

func TestWriteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := Config{
		ConfigPath:       configPath,
		WorkingDirectory: "/tmp/work",
		Provider: Provider{
			Name: "test",
			SourceMapping: SourceMapping{
				LocalSource:    "local/test",
				RegistrySource: "registry/test",
			},
		},
		SnapshotDirectory: "/tmp/snapshots",
	}

	if err := cfg.WriteConfig(); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}
