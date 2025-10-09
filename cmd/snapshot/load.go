package snapshot

import (
	"fmt"

	"github.com/phergul/TerraSnap/internal/config"
	"github.com/phergul/TerraSnap/internal/snapshot"
	"github.com/spf13/cobra"
)

var LoadCmd = &cobra.Command{
	Use:   "load [snapshot-name]",
	Short: "Load a previously saved snapshot",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		fmt.Println("Loading snapshot:", args[0])
		metadata, err := snapshot.LoadSnapshot(cfg, args[0])
		if err != nil {
			fmt.Println("Error loading snapshot:", err)
			return
		}
		fmt.Println("Snapshot loaded successfully:", args[0])
		fmt.Printf("Snapshot metadata: %+v\n", metadata)
	},
}
