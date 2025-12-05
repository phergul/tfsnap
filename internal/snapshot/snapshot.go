package snapshot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/util"
)

const (
	snapshotConfigFile      = "metadata.json"
	snapshotTFConfigFileDir = "tfconfig"
)

func BuildSnapshot(cfg *config.Config, name, description string, includeBinary, includeGit bool) (*Metadata, error) {
	provider, err := detectProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to detect provider: %w", err)
	}
	log.Printf("Including binary: %v, Including git: %v\n", includeBinary, includeGit)

	configAnalysis, err := AnalyseTFConfig(cfg.WorkingDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze Terraform config: %w", err)
	}

	if includeBinary && provider.IsLocalBuild {
		if binaryPath, err := findProviderBinary(cfg); err != nil {
			return nil, fmt.Errorf("failed to find provider binary: %w", err)
		} else {
			if err := captureProviderBinary(cfg, binaryPath, name, provider); err != nil {
				return nil, fmt.Errorf("failed to capture provider binary: %w", err)
			}
		}
	} else if includeBinary && !provider.IsLocalBuild {
		fmt.Println("Warning: Provider is not a local build; binary will not be included")
	}

	if includeGit {
		log.Printf("Getting git info from provider dir: %s\n", cfg.Provider.ProviderDirectory)
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
		Id:             name,
		CreatedAt:      time.Now(),
		ModifiedAt:     time.Now(),
		Provider:       provider,
		Description:    description,
		ConfigAnalysis: configAnalysis,
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

func UpdateSnapshot(cfg *config.Config, name string) (*Metadata, error) {
	log.Println("Updating metedata for snapshot:", name)
	metadata, err := readMetadata(filepath.Join(cfg.SnapshotDirectory, name, snapshotConfigFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	provider, err := detectProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to detect provider: %w", err)
	}

	configAnalysis, err := AnalyseTFConfig(cfg.WorkingDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze Terraform config: %w", err)
	}

	binaryIncluded := false
	if _, err := os.Stat(filepath.Join(cfg.SnapshotDirectory, name, "binary/")); err == nil && !os.IsNotExist(err) {
		binaryIncluded = true
	}

	if binaryIncluded && provider.IsLocalBuild {
		binaryPath, err := findProviderBinary(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to find provider binary: %w", err)
		} else {
			if err := captureProviderBinary(cfg, binaryPath, name, provider); err != nil {
				return nil, fmt.Errorf("failed to capture provider binary: %w", err)
			}
		}
	}

	if metadata.Provider.GitInfo != nil {
		gitInfo := getGitInfo(cfg.Provider.ProviderDirectory)
		provider.GitInfo = gitInfo

		if gitInfo != nil && gitInfo.Commit != "" {
			log.Printf("Git info: %s (%s)\n", gitInfo.Commit[:7], gitInfo.Branch)
			if gitInfo.IsDirty {
				log.Printf("Warning: Uncommitted changes detected\n")
			}
		}
	}

	metadata.ConfigAnalysis = configAnalysis
	metadata.ModifiedAt = time.Now()
	log.Println(time.Now())
	log.Printf("updating metadata ModifiedAt --> %s\n", metadata.ModifiedAt.String())

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

func ListSnapshots(cfg *config.Config) ([]*Metadata, error) {
	var snapshots []*Metadata

	err := filepath.Walk(cfg.SnapshotDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == snapshotConfigFile {
			metadata, err := readMetadata(path)
			if err != nil {
				return fmt.Errorf("failed to read metadata from %s: %w", path, err)
			}
			snapshots = append(snapshots, metadata)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	return snapshots, nil
}

func ListSnapshotNames(cfg *config.Config) []string {
	var names []string
	snapshots, err := ListSnapshots(cfg)
	if err != nil {
		return names
	}
	for _, metadata := range snapshots {
		names = append(names, metadata.Id)
	}
	return names
}

func DeleteSnapshot(cfg *config.Config, name string) error {
	snapshotDir := filepath.Join(cfg.SnapshotDirectory, name)

	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot not found: %s", name)
	}

	if err := os.RemoveAll(snapshotDir); err != nil {
		return fmt.Errorf("failed to delete snapshot directory: %w", err)
	}
	log.Println("Snapshot deleted successfully:", name)
	return nil
}

func LoadSnapshot(cfg *config.Config, name string) error {
	snapshotDir := filepath.Join(cfg.SnapshotDirectory, name)

	err := loadTFFiles(filepath.Join(snapshotDir, snapshotTFConfigFileDir), cfg.WorkingDirectory)
	if err != nil {
		return fmt.Errorf("failed to load terraform files: %w", err)
	}

	return nil
}

func ReplaceWithEmptyConfig(cfg *config.Config) error {
	err := os.Remove(".terraform.lock.hcl")
	if err != nil {
		log.Println("failed to remove .terraform.lock.hcl:", err)
	}

	emptyConfig := fmt.Sprintf(`terraform {
  required_providers {
    %s = {
      source = "%s"
    }
  }
}

`, cfg.Provider.Name, cfg.Provider.SourceMapping.RegistrySource)

	err = os.WriteFile("main.tf", []byte(emptyConfig), 0644)
	if err != nil {
		log.Println("failed to write main.tf:", err)
		fmt.Println("failed to empty main.tf")
	}
	return nil
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

func CopyTerraformFiles(cfg *config.Config, metadata *Metadata) error {
	return util.CopyTFFiles(cfg.WorkingDirectory, filepath.Join(cfg.SnapshotDirectory, metadata.Id, snapshotTFConfigFileDir), false)
}

func loadTFFiles(snapshotTerraformDir, destDir string) error {
	return util.CopyTFFiles(snapshotTerraformDir, destDir, true)
}
