package snapshot

import (
	"fmt"
	"log"
	"strings"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/snapshot"
	"github.com/spf13/cobra"
)

var DeleteCmd = &cobra.Command{
	Use:   "delete <snapshot-name>",
	Short: "Delete a saved snapshot",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		if err := snapshot.DeleteSnapshot(cfg, args[0]); err != nil {
			fmt.Println("Error deleting snapshot:", err)
			return
		}
		log.Println("Snapshot deleted successfully:", args[0])
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			return nil, cobra.ShellCompDirectiveError
		}

		snapshotIds := snapshot.ListSnapshotNames(cfg)
		if len(snapshotIds) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		suggestions := []string{}
		for _, id := range snapshotIds {
			if strings.HasPrefix(id, toComplete) {
				suggestions = append(suggestions, id)
			}
		}
		return suggestions, cobra.ShellCompDirectiveDefault
	},
}
