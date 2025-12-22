package cmd

import (
	"fmt"

	"github.com/phergul/tfsnap/internal/autosave"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore autosaved snapshot",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		fmt.Println("Restoring autosave...")
		err := autosave.RestoreSnapshot(cfg)
		if err != nil {
			fmt.Println("Error restoring snapshot:", err)
			return
		}
		fmt.Println("Restored")
	},
}
