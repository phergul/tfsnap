package snapshot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/util"
)

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
