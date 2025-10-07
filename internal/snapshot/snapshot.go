package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/phergul/TerraSnap/internal/config"
)

type ProviderInfo struct {
	Name             string `json:"name"`
	Source           string `json:"source"`
	NormalisedSource string `json:"normalised_source"`
	Version          string `json:"version"`
	SchemaFile       string `json:"schema_file"`
	Binary           string `json:"binary"`
}

type Metadata struct {
	Id        string       `json:"id"`
	CreatedAt string       `json:"created_at"`
	Provider  ProviderInfo `json:"provider"`
}

func BuildSnapshotMetadata(cfg *config.Config, name string) error {
	provider, err := detectProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to detect provider: %w", err)
	}

	snapshotTime := getCurrentTime()

	metadata := &Metadata{
		Id:        name,
		CreatedAt: snapshotTime,
		Provider:  *provider,
	}

	metadataFile := fmt.Sprintf("%s.json", metadata.Id)
	metadataFilepath := filepath.Join(cfg.SnapshotDirectory, metadataFile)
	file, err := os.Create(metadataFilepath)
	if err != nil {
		return fmt.Errorf("failed to create %s file: %w", metadataFile, err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(metadata); err != nil {
		return fmt.Errorf("failed to write metadata to file: %w", err)
	}

	fmt.Printf("Snapshot metadata saved to %s", metadataFile)
	return nil
}

func detectProvider(cfg *config.Config) (*ProviderInfo, error) {
	module, diag := tfconfig.LoadModule(cfg.WorkingDirectory)
	if diag != nil && diag.Err() != nil {
		return nil, fmt.Errorf("failed to load terraform module: %w", diag.Err())
	}

	if len(module.RequiredProviders) == 0 {
		return nil, fmt.Errorf("no providers found in %s", module.Path)
	} else if len(module.RequiredProviders) > 1 {
		return nil, fmt.Errorf("multiple providers detected, only one is supported")
	}

	var provider ProviderInfo
	for name, req := range module.RequiredProviders {
		normalised := normaliseProviderSource(req.Source, cfg)
		provider = ProviderInfo{
			Name:             name,
			Version:          req.VersionConstraints[0],
			Source:           req.Source,
			NormalisedSource: normalised,
		}
	}

	return &provider, nil
}

func normaliseProviderSource(source string, cfg *config.Config) string {
	if cfg == nil || cfg.Provider.LocalAlias == "" {
		return source
	}
	return cfg.Provider.LocalAlias
}

func getCurrentTime() string {
	now := time.Now()

	return now.Format("2006-01-02_15-04-05")
}
