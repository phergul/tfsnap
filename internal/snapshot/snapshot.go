package snapshot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/phergul/TerraSnap/internal/config"
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
	return copyTFFiles(cfg.WorkingDirectory, filepath.Join(cfg.SnapshotDirectory, metadata.Id, "terraform"))
}

func copyTFFiles(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if strings.Contains(path, ".tfsnap") {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !(strings.HasSuffix(path, ".tf") || strings.HasSuffix(path, ".tfvars") || strings.HasSuffix(path, "terraform.lock.hcl")) {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
