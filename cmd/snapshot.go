package cmd

import (
	"fmt"

	"github.com/phergul/tfsnap/cmd/snapshot"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage terraform snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return nil
		}

		return snapshot.Run(cfg)
	},
}

func init() {
	snapshotCmd.AddCommand(snapshot.SaveCmd)
}
