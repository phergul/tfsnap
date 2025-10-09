package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func CopyTFFiles(src, dst string, load bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !load && strings.Contains(path, ".tfsnap") {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !(strings.HasSuffix(path, ".tf") || strings.HasSuffix(path, ".tfvars") || strings.HasSuffix(path, "terraform.lock.hcl")) {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		return copyFile(path, targetPath)
	})
}
