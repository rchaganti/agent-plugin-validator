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

	// 1. Initial resolve for manifest and mcp should return embedded defaults
	infoManifest, err := mgr.Resolve(SchemaTypeManifest, "")
	if err != nil {
		t.Fatalf("manifest resolve failed: %v", err)
	}
	if infoManifest.Source != "embedded default" {
		t.Errorf("expected source 'embedded default', got %q", infoManifest.Source)
	}
	if infoManifest.ID != CanonicalPluginSchemaURL {
		t.Errorf("unexpected manifest schema ID: %s", infoManifest.ID)
	}

	infoMCP, err := mgr.Resolve(SchemaTypeMCP, "")
	if err != nil {
		t.Fatalf("mcp resolve failed: %v", err)
	}
	if infoMCP.Source != "embedded default" {
		t.Errorf("expected source 'embedded default', got %q", infoMCP.Source)
	}
	if infoMCP.ID != CanonicalMCPSchemaURL {
		t.Errorf("unexpected MCP schema ID: %s", infoMCP.ID)
	}

	// 2. Write a mock cached schema for manifest
	mockCachedSchema := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/cached-schema.json",
		"title": "Cached Schema",
		"type": "object"
	}`)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	cachedPath := filepath.Join(tempDir, CachePluginFileName)
	if err := os.WriteFile(cachedPath, mockCachedSchema, 0644); err != nil {
		t.Fatalf("failed to write mock cache: %v", err)
	}

	// 3. Resolve manifest should now return cached schema
	infoManifest, err = mgr.Resolve(SchemaTypeManifest, "")
	if err != nil {
		t.Fatalf("cached resolve failed: %v", err)
	}
	if infoManifest.Source != "cached" {
		t.Errorf("expected source 'cached', got %q", infoManifest.Source)
	}
	if infoManifest.ID != "https://example.com/cached-schema.json" {
		t.Errorf("expected cached schema ID, got %q", infoManifest.ID)
	}

	// 4. Reset should remove cached schema and revert to embedded
	if err := mgr.Reset(SchemaTypeManifest); err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	infoManifest, err = mgr.Resolve(SchemaTypeManifest, "")
	if err != nil {
		t.Fatalf("post-reset resolve failed: %v", err)
	}
	if infoManifest.Source != "embedded default" {
		t.Errorf("expected source 'embedded default' after reset, got %q", infoManifest.Source)
	}
}

func TestDetectType(t *testing.T) {
	manifestJSON := []byte(`{"$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json", "name": "test"}`)
	mcpJSON := []byte(`{"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json", "mcpServers": {}}`)
	plainJSON := []byte(`{"mcpServers": {}}`)

	if got := DetectType(manifestJSON, "random.json"); got != SchemaTypeManifest {
		t.Errorf("expected manifest, got %s", got)
	}
	if got := DetectType(mcpJSON, "random.json"); got != SchemaTypeMCP {
		t.Errorf("expected mcp, got %s", got)
	}
	if got := DetectType(plainJSON, "mcp.json"); got != SchemaTypeMCP {
		t.Errorf("expected mcp from filename, got %s", got)
	}
	if got := DetectType(plainJSON, "plugin.json"); got != SchemaTypeManifest {
		t.Errorf("expected manifest from filename, got %s", got)
	}
}
