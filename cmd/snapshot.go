package cmd

import (
	"github.com/spf13/cobra"
	"github.com/phergul/TerraSnap/cmd/snapshot"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage terraform snapshots",
}

func init() {
	snapshotCmd.AddCommand(snapshot.SaveCmd)
}
