package autosave

import (
	"fmt"
	"log"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/phergul/tfsnap/internal/snapshot"
	"github.com/spf13/cobra"
)

const AutosaveSnapshotName = "autosave"

func PreRun(cmd *cobra.Command, args []string) {
	if cmd.Name() == "init" || cmd.Name() == "restore" || cmd.Name() == "completion" {
		return
	}
	log.Println("AUTOSAVING SNAPSHOT...")

	cfg := config.FromContext(cmd.Context())
	if cfg == nil {
		log.Println("Config not found in context, skipping autosave")
		return
	}

	autosaveSnapshot(cfg)
	log.Println("AUTOSAVE COMPLETE")
}

func autosaveSnapshot(cfg *config.Config) {
	meta, err := snapshot.BuildSnapshot(cfg, AutosaveSnapshotName, "Autosave snapshot", false, false)
	if err != nil {
		log.Printf("Autosave failed: %v", err)
	}

	err = snapshot.CopyTerraformFiles(cfg, meta)
	if err != nil {
		log.Printf("Failed to copy Terraform files: %v", err)
	}
}

func RestoreSnapshot(cfg *config.Config) error {
	err := snapshot.LoadSnapshot(cfg, AutosaveSnapshotName)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	return nil
}
