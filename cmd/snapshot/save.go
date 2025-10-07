package snapshot

import (
	"fmt"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/phergul/TerraSnap/internal/snapshot"
	"github.com/spf13/cobra"
)

var SaveCmd = &cobra.Command{
	Use:                   "save [snapshot-name]",
	Short:                 "Save a new snapshot of your terraform configuration and binary",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		err := snapshot.BuildSnapshotMetadata(cfg, args[0])
		if err != nil {
			fmt.Printf("Failed to build snapshot: %v\n", err)
			return
		}
		fmt.Printf("Built snapshot successfully")
	},
}
