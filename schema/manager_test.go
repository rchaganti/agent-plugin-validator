package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "apv-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := &Manager{CacheDir: tempDir}

	// 1. Initial resolve should return embedded default
	info, err := mgr.Resolve("")
	if err != nil {
		t.Fatalf("initial resolve failed: %v", err)
	}
	if info.Source != "embedded default" {
		t.Errorf("expected source 'embedded default', got %q", info.Source)
	}
	if info.ID == "" {
		t.Error("expected non-empty schema ID")
	}

	// 2. Write a mock cached schema
	mockCachedSchema := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/cached-schema.json",
		"title": "Cached Schema",
		"type": "object"
	}`)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	cachedPath := filepath.Join(tempDir, CacheFileName)
	if err := os.WriteFile(cachedPath, mockCachedSchema, 0644); err != nil {
		t.Fatalf("failed to write mock cache: %v", err)
	}

	// 3. Resolve should now return cached schema
	info, err = mgr.Resolve("")
	if err != nil {
		t.Fatalf("cached resolve failed: %v", err)
	}
	if info.Source != "cached" {
		t.Errorf("expected source 'cached', got %q", info.Source)
	}
	if info.ID != "https://example.com/cached-schema.json" {
		t.Errorf("expected cached schema ID, got %q", info.ID)
	}

	// 4. Override should take precedence over cached
	mockOverrideFile := filepath.Join(tempDir, "override.json")
	mockOverrideSchema := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/override-schema.json",
		"title": "Override Schema",
		"type": "object"
	}`)
	if err := os.WriteFile(mockOverrideFile, mockOverrideSchema, 0644); err != nil {
		t.Fatalf("failed to write mock override: %v", err)
	}

	info, err = mgr.Resolve(mockOverrideFile)
	if err != nil {
		t.Fatalf("override resolve failed: %v", err)
	}
	if info.Source != "override" {
		t.Errorf("expected source 'override', got %q", info.Source)
	}
	if info.ID != "https://example.com/override-schema.json" {
		t.Errorf("expected override schema ID, got %q", info.ID)
	}

	// 5. Reset should remove cached schema and revert to embedded
	if err := mgr.Reset(); err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	info, err = mgr.Resolve("")
	if err != nil {
		t.Fatalf("post-reset resolve failed: %v", err)
	}
	if info.Source != "embedded default" {
		t.Errorf("expected source 'embedded default' after reset, got %q", info.Source)
	}
}
