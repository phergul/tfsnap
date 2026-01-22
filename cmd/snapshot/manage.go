package snapshot

import (
	"fmt"
	"strings"

	"github.com/phergul/tfsnap/internal/autosave"
	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/snapshot"
	"github.com/phergul/tfsnap/internal/tui"
	"github.com/phergul/tfsnap/internal/util"
)

func Run(cfg *config.Config) error {
	metadataSlice, err := snapshot.ListSnapshots(cfg)
	if err != nil {
		return fmt.Errorf("failed to load snapshots: %w", err)
	}

	if len(metadataSlice) == 0 {
		fmt.Println("No snapshots found.")
		return nil
	}

	items := make([]tui.Item, len(metadataSlice))
	for i, metadata := range metadataSlice {
		items[i] = tui.Item{
			Label:   metadata.Id,
			Content: formatSnapshotDetails(*metadata),
			Meta:    metadata,
		}
	}

	actions := []tui.Action{
		{Key: "enter", Label: "load", Description: "Load this snapshot"},
		{Key: "d", Label: "delete", Description: "Delete this snapshot"},
	}

	result, err := tui.RunActionSelector("Snapshots", items, actions)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if result == nil {
		return nil
	}

	snapshotMeta, ok := result.Item.Meta.(*snapshot.Metadata)
	if !ok {
		return fmt.Errorf("invalid snapshot selected")
	}

	switch result.Action {
	case "enter":
		fmt.Println("Creating autosave...")
		if _, err := snapshot.BuildSnapshot(cfg, autosave.AutosaveSnapshotName, "Autosave snapshot", false, false); err != nil {
			fmt.Printf("Warning: Autosave failed: %v\n", err)
		}

		fmt.Println("Loading snapshot:", snapshotMeta.Id)
		if err := snapshot.LoadSnapshot(cfg, snapshotMeta.Id); err != nil {
			return fmt.Errorf("failed to load snapshot: %w", err)
		}
		fmt.Printf("✔ Snapshot '%s' loaded successfully!\n", snapshotMeta.Id)

	case "d":
		if err := snapshot.DeleteSnapshot(cfg, snapshotMeta.Id); err != nil {
			return fmt.Errorf("failed to delete snapshot: %w", err)
		}
		fmt.Printf("✔ Snapshot '%s' deleted successfully!\n", snapshotMeta.Id)
	}

	return nil
}

func formatSnapshotDetails(snapshotMeta snapshot.Metadata) string {
	var details strings.Builder
	fmt.Fprintf(&details, "Created: %s\n", snapshotMeta.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&details, "Modified: %s\n", snapshotMeta.ModifiedAt.Format("2006-01-02 15:04:05"))

	if snapshotMeta.Description != "" {
		fmt.Fprintf(&details, "\nDescription: %s\n", snapshotMeta.Description)
	}

	if snapshotMeta.Provider != nil {
		fmt.Fprintf(&details, "\nProvider: %s@", snapshotMeta.Provider.Name)
		version := "latest"
		if snapshotMeta.Provider.DetectedVersion != "" {
			version = snapshotMeta.Provider.DetectedVersion
		}
		if snapshotMeta.Provider.IsLocalBuild {
			version = "local"
		}
		fmt.Fprintf(&details, "%s\n", version)

		if snapshotMeta.Provider.GitInfo != nil {
			gitInfo := snapshotMeta.Provider.GitInfo
			if gitInfo.Commit != "" {
				fmt.Fprintf(&details, "Commit: %s\n", gitInfo.Commit[:min(7, len(gitInfo.Commit))])
			}
			if gitInfo.Branch != "" {
				fmt.Fprintf(&details, "Branch: %s\n", gitInfo.Branch)
			}
			if gitInfo.IsDirty {
				fmt.Fprintf(&details, "Status: Uncommitted changes\n")
			}
		}
	}

	if snapshotMeta.ConfigAnalysis != nil {
		fmt.Fprintf(&details, "\nResources: %d total\n", snapshotMeta.ConfigAnalysis.TotalCount)
		if len(snapshotMeta.ConfigAnalysis.Resources) > 0 {
			fmt.Fprintf(&details, "\nResources by type:\n")
			resourceKeys := util.SortedKeys(snapshotMeta.ConfigAnalysis.Resources)
			for _, resourceName := range resourceKeys {
				resource := snapshotMeta.ConfigAnalysis.Resources[resourceName]
				fmt.Fprintf(&details, "  %s: %d\n", resourceName, resource.Count)
			}
		}
	}

	if snapshotMeta.Provider != nil && snapshotMeta.Provider.Binary != nil {
		binary := snapshotMeta.Provider.Binary
		fmt.Fprintf(&details, "\nBinary included: Yes (%.1f MB)\n", float64(binary.Size)/(1024*1024))
	}

	return details.String()
}
