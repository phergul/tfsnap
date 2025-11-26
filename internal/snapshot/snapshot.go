package snapshot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/phergul/TerraSnap/internal/config"
	"github.com/phergul/TerraSnap/internal/util"
)

const (
	snapshotConfigFile      = "metadata.json"
	snapshotTFConfigFileDir = "tfconfig"
)

func BuildSnapshotMetadata(cfg *config.Config, name, description string, includeBinary, includeGit bool) (*Metadata, error) {
	provider, err := detectProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to detect provider: %w", err)
	}

	if includeBinary && provider.IsLocalBuild {
		binaryPath, err := findProviderBinary(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to find provider binary: %w", err)
		} else {
			if err := captureProviderBinary(cfg, binaryPath, name, provider); err != nil {
				return nil, fmt.Errorf("failed to capture provider binary: %w", err)
			}
		}
	}

	if includeGit {
		gitInfo := getGitInfo(cfg.Provider.ProviderDirectory)
		provider.GitInfo = gitInfo

		if gitInfo != nil && gitInfo.Commit != "" {
			log.Printf("Git info: %s (%s)\n", gitInfo.Commit[:7], gitInfo.Branch)
			if gitInfo.IsDirty {
				log.Printf("Warning: Uncommitted changes detected\n")
			}
		}
	}

	metadata := &Metadata{
		Id:          name,
		CreatedAt:   time.Now(),
		Provider:    provider,
		Description: description,
	}

	metadataFilepath := filepath.Join(cfg.SnapshotDirectory, name, snapshotConfigFile)
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

	log.Printf("Snapshot metadata saved to %s", metadataFilepath)
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
	return util.CopyTFFiles(cfg.WorkingDirectory, filepath.Join(cfg.SnapshotDirectory, metadata.Id, snapshotTFConfigFileDir), false)
}

func LoadSnapshot(cfg *config.Config, name string) (*Metadata, error) {
	snapshotDir := filepath.Join(cfg.SnapshotDirectory, name)
	metadataFile := filepath.Join(snapshotDir, snapshotConfigFile)

	metadata, err := readMetadata(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot metadata: %w", err)
	}

	err = loadTFFiles(filepath.Join(snapshotDir, snapshotTFConfigFileDir), cfg.WorkingDirectory)
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

func findProviderBinary(cfg *config.Config) (string, error) {
	if cfg.Provider.ProviderDirectory == "" {
		return "", fmt.Errorf("provider directory not configured")
	}

	possiblePaths := []string{
		filepath.Join(cfg.Provider.ProviderDirectory, "terraform-provider-"+cfg.Provider.Name),
		filepath.Join(cfg.Provider.ProviderDirectory, "bin", "terraform-provider-"+cfg.Provider.Name),
		filepath.Join(cfg.Provider.ProviderDirectory, "dist", "terraform-provider-"+cfg.Provider.Name),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("provider binary not found in %s", cfg.Provider.ProviderDirectory)
}

func captureProviderBinary(cfg *config.Config, binaryPath, snapshotName string, provider *ProviderInfo) error {
	providerDir := filepath.Join(cfg.SnapshotDirectory, snapshotName, "provider")
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		return fmt.Errorf("failed to create provider directory: %w", err)
	}

	hash, err := util.HashFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to hash binary: %w", err)
	}

	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}

	binaryName := filepath.Base(binaryPath)
	destPath := filepath.Join(providerDir, binaryName)

	if err := util.CopyFile(binaryPath, destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	provider.Binary = &Binary{
		OriginalPath:       binaryPath,
		SnapshotBinaryPath: filepath.Join("provider", binaryName),
		Hash:               hash,
		Size:               info.Size(),
	}

	log.Printf("Provider binary captured (hash: %s, size: %.2f MB)\n",
		hash[:8], float64(info.Size())/1024/1024)

	return nil
}
