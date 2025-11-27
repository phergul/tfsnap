package cmd

import (
	"github.com/phergul/TerraSnap/cmd/snapshot"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage terraform snapshots",
}

func init() {
	snapshotCmd.AddCommand(snapshot.SaveCmd)
	snapshotCmd.AddCommand(snapshot.LoadCmd)
	snapshotCmd.AddCommand(snapshot.ListCmd)
	snapshotCmd.AddCommand(snapshot.DeleteCmd)
}
