package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/tfsnap/internal/config"
)

func TestAnalyseTFConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tfContent := `
resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
}

resource "aws_instance" "another" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.small"
}

resource "aws_s3_bucket" "storage" {
  bucket = "my-bucket"
}
`

	mainTfPath := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(mainTfPath, []byte(tfContent), 0644); err != nil {
		t.Fatalf("Failed to create test terraform file: %v", err)
	}

	analysis, err := AnalyseTFConfig(tmpDir)
	if err != nil {
		t.Fatalf("AnalyseTFConfig failed: %v", err)
	}

	if analysis == nil {
		t.Error("AnalyseTFConfig returned nil")
	}

	if analysis.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", analysis.TotalCount)
	}

	// Validate resource counts are tracked
	if count, ok := analysis.Resources["aws_instance"]; !ok || count.Count != 2 {
		t.Errorf("Expected 2 aws_instance resources, got %v", count)
	}

	if count, ok := analysis.Resources["aws_s3_bucket"]; !ok || count.Count != 1 {
		t.Errorf("Expected 1 aws_s3_bucket resource, got %v", count)
	}
}

func TestAnalyseTFConfigEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	mainTfPath := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(mainTfPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test terraform file: %v", err)
	}

	analysis, err := AnalyseTFConfig(tmpDir)
	if err != nil {
		t.Fatalf("AnalyseTFConfig failed: %v", err)
	}

	if analysis.TotalCount != 0 {
		t.Errorf("Expected total count 0, got %d", analysis.TotalCount)
	}
}

func TestListSnapshotNames(t *testing.T) {
	tmpDir := t.TempDir()

	expectedNames := []string{"snap1", "snap2"}

	for _, name := range expectedNames {
		snapDir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(snapDir, 0755); err != nil {
			t.Fatalf("Failed to create snapshot dir: %v", err)
		}

		metadataPath := filepath.Join(snapDir, "metadata.json")
		metadata := `{"id":"` + name + `","created_at":"2024-01-01T00:00:00Z","modified_at":"2024-01-01T00:00:00Z"}`
		if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil {
			t.Fatalf("Failed to write metadata: %v", err)
		}
	}

	cfg := &config.Config{
		SnapshotDirectory: tmpDir,
	}

	names := ListSnapshotNames(cfg)

	if len(names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(names))
	}
}

func TestDeleteSnapshot(t *testing.T) {
	tmpDir := t.TempDir()

	snapDir := filepath.Join(tmpDir, "test-snap")
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		t.Fatalf("Failed to create snapshot dir: %v", err)
	}

	cfg := &config.Config{
		SnapshotDirectory: tmpDir,
	}

	err := DeleteSnapshot(cfg, "test-snap")
	if err != nil {
		t.Fatalf("DeleteSnapshot failed: %v", err)
	}

	if _, err := os.Stat(snapDir); !os.IsNotExist(err) {
		t.Error("Snapshot directory was not deleted")
	}
}

func TestDeleteSnapshotNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		SnapshotDirectory: tmpDir,
	}

	err := DeleteSnapshot(cfg, "nonexistent")
	if err == nil {
		t.Error("DeleteSnapshot should return error for nonexistent snapshot")
	}
}

func TestReplaceWithEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	cfg := &config.Config{
		Provider: config.Provider{
			Name: "aws",
			SourceMapping: config.SourceMapping{
				RegistrySource: "hashicorp/aws",
			},
		},
	}

	err = ReplaceWithEmptyConfig(cfg)
	if err != nil {
		t.Fatalf("ReplaceWithEmptyConfig failed: %v", err)
	}

	mainTf := filepath.Join(tmpDir, "main.tf")
	if _, err := os.Stat(mainTf); os.IsNotExist(err) {
		t.Error("main.tf was not created")
	}

	content, err := os.ReadFile(mainTf)
	if err != nil {
		t.Fatalf("Failed to read main.tf: %v", err)
	}

	if len(content) == 0 {
		t.Error("main.tf is empty")
	}

	// Validate config contains required terraform block
	contentStr := string(content)
	if !contains(contentStr, "terraform") {
		t.Error("main.tf should contain terraform block")
	}

	if !contains(contentStr, "hashicorp/aws") {
		t.Error("main.tf should contain provider source")
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
