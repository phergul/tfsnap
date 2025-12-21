package autosave

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/tfsnap/internal/config"
	"github.com/spf13/cobra"
)

func TestRestoreSnapshot(t *testing.T) {
	tmpDir := t.TempDir()

	snapDir := filepath.Join(tmpDir, AutosaveSnapshotName)
	tfConfigDir := filepath.Join(snapDir, "tfconfig")
	if err := os.MkdirAll(tfConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create snapshot dir: %v", err)
	}

	testTfFile := filepath.Join(tfConfigDir, "main.tf")
	tfContent := `resource "aws_instance" "test" { }`
	if err := os.WriteFile(testTfFile, []byte(tfContent), 0644); err != nil {
		t.Fatalf("Failed to create test tf file: %v", err)
	}

	workDir := filepath.Join(tmpDir, "work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	cfg := &config.Config{
		SnapshotDirectory: tmpDir,
		WorkingDirectory:  workDir,
	}

	err := RestoreSnapshot(cfg)
	if err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}

	// Validate terraform files were restored
	restoredTfFile := filepath.Join(workDir, "main.tf")
	if _, err := os.Stat(restoredTfFile); os.IsNotExist(err) {
		t.Error("Terraform file was not restored to working directory")
	}

	// Validate content was restored correctly
	restored, err := os.ReadFile(restoredTfFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restored) != tfContent {
		t.Errorf("Restored content mismatch: expected %q, got %q", tfContent, string(restored))
	}
}

func TestRestoreSnapshotNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		SnapshotDirectory: tmpDir,
		WorkingDirectory:  tmpDir,
	}

	err := RestoreSnapshot(cfg)
	if err == nil {
		t.Error("RestoreSnapshot should return error for nonexistent snapshot")
	}
}

func TestPreRunInitCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use: "init",
	}

	ctx := context.Background()
	cmd.SetContext(ctx)

	PreRun(cmd, []string{})
}

func TestPreRunRestoreCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use: "restore",
	}

	ctx := context.Background()
	cmd.SetContext(ctx)

	PreRun(cmd, []string{})
}

func TestAutosaveSnapshotNameConstant(t *testing.T) {
	if AutosaveSnapshotName != "autosave" {
		t.Errorf("Expected AutosaveSnapshotName to be 'autosave', got %q", AutosaveSnapshotName)
	}
}
