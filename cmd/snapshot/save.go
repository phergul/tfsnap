package snapshot

import (
	"fmt"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/phergul/TerraSnap/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	description   string
	includeBinary bool
	includeGit    bool
)

var SaveCmd = &cobra.Command{
	Use:   "save [snapshot-name]",
	Short: "Save a new snapshot of your terraform configuration and binary",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		fmt.Println("Saving snapshot:", args[0])
		metadata, err := snapshot.BuildSnapshotMetadata(cfg, args[0], description, includeBinary, includeGit)
		if err != nil {
			fmt.Printf("Failed to build snapshot: %v\n", err)
			return
		}

		fmt.Println("Copying terraform files...")
		err = snapshot.CopyTerraformFiles(cfg, metadata)
		if err != nil {
			fmt.Printf("Failed to copy terraform files: %v\n", err)
			return
		}

		fmt.Printf("\nSuccessfully saved snapshot: %s\n", args[0])
		fmt.Printf("Summary: %s@%s\n", metadata.Provider.Name, metadata.Provider.DetectedVersion)
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
	},
}

func init() {
	SaveCmd.Flags().StringVarP(&description, "description", "d", "", "Description of this snapshot")
	SaveCmd.Flags().BoolVarP(&includeBinary, "include-binary", "b", false, "Whether to include the binary of the provider (local)")
	SaveCmd.Flags().BoolVarP(&includeGit, "include-git", "g", false, "Whether to include info on the providers git branch")
}

