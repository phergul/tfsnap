package snapshot

import (
	"fmt"
	"log"
	"strings"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/snapshot"
	"github.com/phergul/tfsnap/internal/util"
	"github.com/spf13/cobra"
)

var detailed bool

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved snapshots",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg == nil {
			fmt.Println("configuration not found in context; run `tfsnap init` first")
			return
		}

		metadataSlice, err := snapshot.ListSnapshots(cfg)
		if err != nil {
			fmt.Println("Error loading snapshots:", err)
			return
		}
		if len(metadataSlice) == 0 {
			fmt.Println("No snapshots found.")
			return
		}

		log.Println("Snapshots loaded successfully:")
		for _, metadata := range metadataSlice {
			fmt.Println(printSnapshotDetails(*metadata))
		}
	},
}

func printSnapshotDetails(snapshot snapshot.Metadata) string {
	var details strings.Builder
	details.WriteString(fmt.Sprintf("\nSnapshot Id: %s\n", snapshot.Id))
	details.WriteString(fmt.Sprintf("Created At: %s\n", snapshot.CreatedAt))
	details.WriteString(fmt.Sprintf("Modified At: %s\n", snapshot.ModifiedAt))
	if snapshot.Description != "" {
		details.WriteString(fmt.Sprintf("Description: %s\n", snapshot.Description))
	}
	if snapshot.Provider != nil {
		details.WriteString(fmt.Sprintf("Provider: %s@", snapshot.Provider.Name))
		version := "latest"
		if snapshot.Provider.DetectedVersion != "" {
			version = snapshot.Provider.DetectedVersion
		}
		if snapshot.Provider.IsLocalBuild {
			version = "local"
		}
		details.WriteString(fmt.Sprintf("%s\n", version))
	}

	if detailed {
		details.WriteString(fmt.Sprintf("\nTotal number of resources: %d\n", snapshot.ConfigAnalysis.TotalCount))
		details.WriteString("Resources by type:\n")
		resourceKeys := util.SortedKeys(snapshot.ConfigAnalysis.Resources)
		for _, resourceName := range resourceKeys {
			resource := snapshot.ConfigAnalysis.Resources[resourceName]
			details.WriteString(fmt.Sprintf("  [%s]: %d\n", resourceName, resource.Count))
		}
	}
	fmt.Println("------------------------------------")
	return details.String()
}

func init() {
	ListCmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "Whether to print the details for the snapshots in the list")
}
