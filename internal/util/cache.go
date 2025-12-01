package util

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	CacheMiss = "cache miss"
	CacheHit  = "cache hit"
)

type FileCache[T any] struct {
	dir string
}

func GetCache[T any](dir, cacheType string) *FileCache[T] {
	cacheDir := filepath.Join(filepath.Join(dir, ".tfsnap"), "cache", cacheType)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("failed to create cache directory: %v", err)
		return nil
	}

	return &FileCache[T]{dir: cacheDir}
}

func (c *FileCache[T]) Get(key string) (*T, error) {
	data, err := os.ReadFile(filepath.Join(c.dir, key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s: %w", CacheMiss, err)
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return &value, nil
}

func (c *FileCache[T]) Set(key string, value T) error {
	jsonData, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	if err := os.WriteFile(filepath.Join(c.dir, key), jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}
