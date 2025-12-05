package snapshot

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/snapshot"
	"github.com/phergul/tfsnap/internal/util"
	"github.com/spf13/cobra"
)

var (
	description   string
	includeBinary bool
	includeGit    bool
	persist       bool
)

var SaveCmd = &cobra.Command{
	Use:   "save <snapshot-name>",
	Short: "Save a new snapshot of your terraform configuration and binary",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		var metadata *snapshot.Metadata
		var err error
		if !util.DirExists(filepath.Join(cfg.SnapshotDirectory, args[0])) {
			fmt.Println("Saving snapshot:", args[0])
			metadata, err = snapshot.BuildSnapshot(cfg, args[0], description, includeBinary, includeGit)
			if err != nil {
				fmt.Printf("Failed to build snapshot: %v\n", err)
				return
			}
		} else {
			fmt.Println("Updating existing snapshot:", args[0])
			metadata, err = snapshot.UpdateSnapshot(cfg, args[0])
			if err != nil {
				fmt.Printf("Failed to update snapshot: %v\n", err)
				return
			}
		}

		log.Println("Copying terraform files...")
		err = snapshot.CopyTerraformFiles(cfg, metadata)
		if err != nil {
			fmt.Printf("Failed to copy terraform files: %v\n", err)
			return
		}

		fmt.Printf("\nSuccessfully saved snapshot: %s\n", args[0])
		versionInfo := "latest"
		if metadata.Provider.DetectedVersion != "" {
			versionInfo = metadata.Provider.DetectedVersion
		}
		if metadata.Provider.IsLocalBuild {
			versionInfo = "local"
		}
		fmt.Printf("Summary: %s@%s\n", metadata.Provider.Name, versionInfo)
		if metadata.Provider.Binary != nil {
			fmt.Printf("Provider binary: %.2f MB (hash: %s)\n",
				float64(metadata.Provider.Binary.Size)/1024/1024,
				metadata.Provider.Binary.Hash[:8])
		}
		if metadata.Provider.GitInfo != nil && metadata.Provider.GitInfo.Commit != "" {
			fmt.Printf("Git: %s (%s)\n", metadata.Provider.GitInfo.Commit[:7], metadata.Provider.GitInfo.Branch)
			if metadata.Provider.GitInfo.IsDirty {
				fmt.Printf("Warning: Uncommitted changes detected\n")
			}
		}

		if persist {
			return
		}
		if err := snapshot.ReplaceWithEmptyConfig(cfg); err != nil {
			fmt.Printf("Failed to replace terraform files with empty config: %v\n", err)
		}
	},
}

func init() {
	SaveCmd.Flags().StringVarP(&description, "description", "d", "", "Description of this snapshot")
	SaveCmd.Flags().BoolVarP(&includeBinary, "include-binary", "b", false, "Whether to include the binary of the provider")
	SaveCmd.Flags().BoolVarP(&includeGit, "include-git", "g", false, "Whether to include provider repo git info")
	SaveCmd.Flags().BoolVarP(&persist, "persist", "p", false, "Whether to persist the saved config")
}
