package config

import (
	"context"
	"testing"
)

func TestToContext(t *testing.T) {
	cfg := &Config{
		WorkingDirectory: "/test",
		Provider: Provider{
			Name: "test-provider",
		},
	}

	ctx := context.Background()
	newCtx := ToContext(ctx, cfg)

	if newCtx == ctx {
		t.Error("ToContext should return a new context")
	}
}

func TestFromContext(t *testing.T) {
	cfg := &Config{
		WorkingDirectory: "/test",
		Provider: Provider{
			Name: "test-provider",
		},
	}

	ctx := context.Background()
	ctx = ToContext(ctx, cfg)

	retrieved := FromContext(ctx)
	if retrieved == nil {
		t.Error("FromContext returned nil")
	}

	if retrieved.WorkingDirectory != cfg.WorkingDirectory {
		t.Errorf("Expected working directory %q, got %q", cfg.WorkingDirectory, retrieved.WorkingDirectory)
	}
}

func TestFromContextEmpty(t *testing.T) {
	ctx := context.Background()
	retrieved := FromContext(ctx)

	if retrieved != nil {
		t.Error("FromContext should return nil for context without config")
	}
}
