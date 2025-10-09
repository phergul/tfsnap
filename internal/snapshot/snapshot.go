package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/phergul/TerraSnap/internal/config"
	"github.com/phergul/TerraSnap/internal/util"
)

type ProviderInfo struct {
	Name             string `json:"name"`
	DetectedSource   string `json:"detected_source"`
	DetectedVersion  string `json:"detected_version"`
	NormalizedSource string `json:"normalized_source,omitempty"`
	IsLocalBuild     bool   `json:"is_local_build"`
	SchemaFile       string `json:"schema_file,omitempty"`
	Binary           string `json:"binary,omitempty"`
}

type Metadata struct {
	Id        string       `json:"id"`
	CreatedAt string       `json:"created_at"`
	Provider  ProviderInfo `json:"provider"`
}

func BuildSnapshotMetadata(cfg *config.Config, name string) (*Metadata, error) {
	provider, err := detectProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to detect provider: %w", err)
	}

	snapshotTime := getCurrentTime()

	metadata := &Metadata{
		Id:        name,
		CreatedAt: snapshotTime,
		Provider:  *provider,
	}

	metadataFilepath := filepath.Join(cfg.SnapshotDirectory, name, "metadata.json")
	if err := os.MkdirAll(filepath.Dir(metadataFilepath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	file, err := os.Create(metadataFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s file: %w", metadataFilepath, err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(metadata); err != nil {
		return nil, fmt.Errorf("failed to write metadata to file: %w", err)
	}

	fmt.Printf("Snapshot metadata saved to %s", metadataFilepath)
	return metadata, nil
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
		detectedSource := req.Source
		detectedVersion := ""
		if len(req.VersionConstraints) > 0 {
			detectedVersion = req.VersionConstraints[0]
		}
		normalizedSource, isLocal := normalizeProviderSource(detectedSource, cfg)

		provider = ProviderInfo{
			Name:             name,
			DetectedSource:   detectedSource,
			DetectedVersion:  detectedVersion,
			NormalizedSource: normalizedSource,
			IsLocalBuild:     isLocal,
		}
	}

	return &provider, nil
}

func normalizeProviderSource(detectedSource string, cfg *config.Config) (string, bool) {
	mapping := cfg.Provider.SourceMapping
	if detectedSource == mapping.LocalSource {
		return mapping.RegistrySource, true
	}
	if detectedSource == mapping.RegistrySource {
		return mapping.RegistrySource, false
	}

	// fallback to pattern-based detection
	if isLikelyLocalSource(detectedSource) {
		normalized := extractNormalizedSource(detectedSource)
		return normalized, true
	}

	return detectedSource, false
}

func isLikelyLocalSource(source string) bool {
	if strings.Contains(source, ".com/") ||
		strings.Contains(source, ".io/") ||
		strings.Contains(source, ".net/") ||
		strings.Contains(source, ".org/") {
		return true
	}
	return false
}

func extractNormalizedSource(source string) string {
	parts := strings.Split(source, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return source
}

func getCurrentTime() string {
	now := time.Now()

	return now.Format("2006-01-02_15-04-05")
}

func CopyTerraformFiles(cfg *config.Config, metadata *Metadata) error {
	return util.CopyTFFiles(cfg.WorkingDirectory, filepath.Join(cfg.SnapshotDirectory, metadata.Id, "terraform"), false)
}

func LoadSnapshot(cfg *config.Config, name string) (*Metadata, error) {
	snapshotDir := filepath.Join(cfg.SnapshotDirectory, name)
	metadataFile := filepath.Join(snapshotDir, "metadata.json")

	metadata, err := readMetadata(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot metadata: %w", err)
	}

	err = loadTFFiles(filepath.Join(snapshotDir, "terraform"), cfg.WorkingDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to load terraform files: %w", err)
	}

	return metadata, nil
}

func readMetadata(filePath string) (*Metadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer file.Close()

	var metadata Metadata
	if err := json.NewDecoder(file).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata JSON: %w", err)
	}
	return &metadata, nil
}

func loadTFFiles(snapshotTerraformDir, destDir string) error {
	return util.CopyTFFiles(snapshotTerraformDir, destDir, true)
}
