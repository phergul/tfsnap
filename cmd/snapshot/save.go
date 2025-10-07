package snapshot

import (
	"fmt"
	"log"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/phergul/TerraSnap/internal/snapshot"
	"github.com/spf13/cobra"
)

var SaveCmd = &cobra.Command{
	Use:                   "save [snapshot-name]",
	Short:                 "Save a new snapshot of your terraform configuration and binary",
	Args:                  cobra.MaximumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		var snapshotName string
		if len(args) > 0 {
			snapshotName = args[0]
		}

		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		snapshot, err := snapshot.BuildSnapshot(cfg.WorkingDirectory, snapshotName)
		if err != nil {
			fmt.Printf("Failed to build snapshot: %v\n", err)
			return
		}
		log.Printf("Built snapshot: %+v\n", snapshot)

	},
}
