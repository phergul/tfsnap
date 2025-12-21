package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	srcContent := "test content"
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	dstPath := filepath.Join(tmpDir, "dest.txt")
	err := CopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != srcContent {
		t.Errorf("Content mismatch: expected %q, got %q", srcContent, string(dstContent))
	}
}

func TestCopyFileNonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	err := CopyFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("CopyFile should validate source exists and return error for nonexistent source")
	}

	// Verify error is non-empty
	if err != nil && err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := HashFile(filePath)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash is empty")
	}

	// Validate hash is consistent
	hash2, err := HashFile(filePath)
	if err != nil {
		t.Fatalf("Second HashFile call failed: %v", err)
	}

	if hash != hash2 {
		t.Errorf("Hashes should be consistent: %q vs %q", hash, hash2)
	}

	// Validate hash has expected length (SHA256 hex is 64 chars)
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	if !DirExists(tmpDir) {
		t.Error("DirExists should return true for existing directory")
	}

	if DirExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("DirExists should return false for nonexistent directory")
	}

	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if DirExists(filePath) {
		t.Error("DirExists should return false for files")
	}
}

func TestCopyTFFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	tfFile := filepath.Join(srcDir, "main.tf")
	tfVarsFile := filepath.Join(srcDir, "terraform.tfvars")
	ignoredFile := filepath.Join(srcDir, "ignored.txt")

	files := map[string]string{
		tfFile:      "resource \"aws_instance\" \"example\" {}",
		tfVarsFile:  "instance_count = 5",
		ignoredFile: "should be ignored",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	err := CopyTFFiles(srcDir, dstDir, false)
	if err != nil {
		t.Fatalf("CopyTFFiles failed: %v", err)
	}

	// Validate terraform files were copied
	if _, err := os.Stat(filepath.Join(dstDir, "main.tf")); os.IsNotExist(err) {
		t.Error("main.tf was not copied")
	}

	if _, err := os.Stat(filepath.Join(dstDir, "terraform.tfvars")); os.IsNotExist(err) {
		t.Error("terraform.tfvars was not copied")
	}

	// Validate non-terraform files were not copied
	if _, err := os.Stat(filepath.Join(dstDir, "ignored.txt")); !os.IsNotExist(err) {
		t.Error("ignored.txt should not have been copied")
	}
}

func TestSortedKeys(t *testing.T) {
	testMap := map[string]int{
		"zebra":  1,
		"apple":  2,
		"mango":  3,
		"banana": 4,
	}

	keys := SortedKeys(testMap)

	if len(keys) != 4 {
		t.Errorf("Expected 4 keys, got %d", len(keys))
	}

	expected := []string{"apple", "banana", "mango", "zebra"}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Expected key %q at position %d, got %q", expected[i], i, key)
		}
	}
}

func TestSortedKeysEmpty(t *testing.T) {
	emptyMap := make(map[string]int)
	keys := SortedKeys(emptyMap)

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for empty map, got %d", len(keys))
	}
}

func TestGetCache(t *testing.T) {
	tmpDir := t.TempDir()

	cache := GetCache[string](tmpDir, "test-cache")
	if cache == nil {
		t.Error("GetCache returned nil")
	}
}

func TestFileCacheSetGet(t *testing.T) {
	tmpDir := t.TempDir()

	cache := GetCache[string](tmpDir, "test")
	if cache == nil {
		t.Fatal("GetCache returned nil")
	}

	testKey := "test-key"
	testValue := "test-value"

	err := cache.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("Cache.Set failed: %v", err)
	}

	retrieved, err := cache.Get(testKey)
	if err != nil {
		t.Fatalf("Cache.Get failed: %v", err)
	}

	if retrieved == nil || *retrieved != testValue {
		t.Errorf("Expected %q, got %v", testValue, retrieved)
	}
}

func TestFileCacheGetMissing(t *testing.T) {
	tmpDir := t.TempDir()

	cache := GetCache[string](tmpDir, "test")
	if cache == nil {
		t.Fatal("GetCache returned nil")
	}

	_, err := cache.Get("nonexistent-key")
	if err == nil {
		t.Error("Cache.Get should return error for missing key")
	}
}
