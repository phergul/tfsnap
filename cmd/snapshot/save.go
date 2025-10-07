package snapshot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/phergul/TerraSnap/internal/config"
	"github.com/phergul/TerraSnap/internal/snapshot"
	"github.com/spf13/cobra"
)

var name string

var SaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save a new snapshot of your terraform configuration and binary",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		if name == "" {
			name = fmt.Sprintf("dev-%d", time.Now().Unix())
		}

		snapshotPath := filepath.Join(cfg.SnapshotDirectory)
		if err := os.MkdirAll(snapshotPath, 0755); err != nil {
			fmt.Printf("Failed to create snapshot directory: %v\n", err)
			return
		}

		log.Println("here", cfg)
		provider, err := detectProvider(cfg.WorkingDirectory, cfg.Provider.Name)
		if err != nil {
			fmt.Printf("Failed to detect provider: %v\n", err)
			return
		}
		return
		provider = getProviderFromConfiguration(cfg)
		if provider == nil {
			fmt.Println("No provider found to save in snapshot")
			return
		}

		meta := snapshot.Metadata{
			Id:        name,
			CreatedAt: time.Now(),
			Provider: snapshot.Provider{
				Name:       cfg.Provider.Name,
				Version:    "",
				SchemaFile: "",
				// Binary:     providerRelPath,
			},
		}

		// write metadata.json
		metaFile := filepath.Join(snapshotPath, "metadata.json")
		f, err := os.Create(metaFile)
		if err != nil {
			fmt.Printf("failed to create metadata file: %v\n", err)
			return
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&meta); err != nil {
			fmt.Printf("failed to write metadata: %v\n", err)
			f.Close()
			return
		}
		f.Close()

		fmt.Printf("Saved snapshot '%s' -> %s\n", name, snapshotPath)
	},
}

func getProviderFromConfiguration(cfg *config.Config) *snapshot.Provider {
	if cfg.Provider.ProviderDirectory == "" {
		return nil
	}

	// fileBytes, err := os.ReadFile(filepath.Join(cfg.WorkingDirectory, "main.tf"))
	// if err != nil {
	// 	fmt.Printf("failed to read main.tf: %v\n", err)
	// 	return nil
	// }
return nil
}

func detectProvider(dir string, wantName string) (*snapshot.Provider, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("terraform directory not found: %s", dir)
	}

	module, diag := tfconfig.LoadModule(dir)
	if diag != nil && diag.Err() != nil {
		return nil, fmt.Errorf("failed to load terraform module: %w", diag.Err())
	}

	fmt.Println("module: ", module)
	return nil, nil
	providers := map[string]string{}
	for name, req := range module.RequiredProviders {
		providers[name] = req.VersionConstraints[0]
	}

	// If user supplied a provider name, prefer it
	if wantName != "" {
		if v, ok := providers[wantName]; ok {
			return &snapshot.Provider{
				Name:    wantName,
				Version: v,
			}, nil
		}
		return nil, fmt.Errorf("configured provider %q not found in Terraform files", wantName)
	}

	// No explicit provider requested: pick if exactly one
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers detected in %s", dir)
	}
	if len(providers) == 1 {
		for n, v := range providers {
			return &snapshot.Provider{
				Name:    n,
				Version: v,
			}, nil
		}
	}

	// Multiple providers found; caller should disambiguate
	keys := make([]string, 0, len(providers))
	for k := range providers {
		keys = append(keys, k)
	}
	return nil, fmt.Errorf("multiple providers detected (%v); please specify one with --provider", keys)
}
