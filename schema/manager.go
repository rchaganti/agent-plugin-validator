package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	SchemaTypeManifest = "manifest"
	SchemaTypeMCP      = "mcp"

	CanonicalPluginSchemaURL = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"
	CanonicalMCPSchemaURL    = "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json"

	CachePluginFileName = "plugin.schema.json"
	CacheMCPFileName    = "mcp.schema.json"
)

// SchemaInfo holds information about a resolved or loaded schema.
type SchemaInfo struct {
	Type      string `json:"type"`   // "manifest" | "mcp"
	Source    string `json:"source"` // "embedded default" | "cached" | "override"
	Path      string `json:"path"`   // filesystem path, URL, or "(embedded)"
	ID        string `json:"id"`     // $id value from the schema
	Title     string `json:"title"`  // title from the schema
	RawSchema []byte `json:"-"`      // Raw schema byte content
}

// Manager manages local schema caching, resolution, and updating.
type Manager struct {
	CacheDir string
}

// NewManager creates a Manager with the default cache directory (~/.apv/schemas).
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	cacheDir := filepath.Join(home, ".apv", "schemas")
	return &Manager{CacheDir: cacheDir}, nil
}

// CachedPath returns the absolute path to the cached schema file for a schema type.
func (m *Manager) CachedPath(schemaType string) string {
	fileName := CachePluginFileName
	if schemaType == SchemaTypeMCP {
		fileName = CacheMCPFileName
	}
	return filepath.Join(m.CacheDir, fileName)
}

// CanonicalURL returns the default canonical schema URL for a schema type.
func CanonicalURL(schemaType string) string {
	if schemaType == SchemaTypeMCP {
		return CanonicalMCPSchemaURL
	}
	return CanonicalPluginSchemaURL
}

// DetectType inspects input document content and filename to determine if it is "mcp" or "manifest".
func DetectType(inputData []byte, filename string) string {
	var doc struct {
		Schema string `json:"$schema"`
	}
	if err := json.Unmarshal(inputData, &doc); err == nil && doc.Schema != "" {
		if strings.Contains(doc.Schema, "mcp.schema.json") {
			return SchemaTypeMCP
		}
		if strings.Contains(doc.Schema, "plugin.schema.json") {
			return SchemaTypeManifest
		}
	}

	base := strings.ToLower(filepath.Base(filename))
	if base == "mcp.json" {
		return SchemaTypeMCP
	}
	return SchemaTypeManifest
}

// Resolve resolves the active schema based on precedence:
// 1. Explicit override (file path or URL)
// 2. Cached schema in ~/.apv/schemas/
// 3. Embedded default schema
func (m *Manager) Resolve(schemaType string, override string) (*SchemaInfo, error) {
	if schemaType != SchemaTypeMCP {
		schemaType = SchemaTypeManifest
	}

	if override != "" {
		info, err := m.loadFromOverride(override)
		if err != nil {
			return nil, err
		}
		info.Type = schemaType
		return info, nil
	}

	cachedPath := m.CachedPath(schemaType)
	if _, err := os.Stat(cachedPath); err == nil {
		data, err := os.ReadFile(cachedPath)
		if err == nil {
			info, parseErr := parseSchemaMeta(data)
			if parseErr == nil {
				info.Type = schemaType
				info.Source = "cached"
				info.Path = cachedPath
				info.RawSchema = data
				return info, nil
			}
		}
	}

	// Fallback to embedded default
	data := EmbeddedSchema(schemaType)
	info, err := parseSchemaMeta(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded schema: %w", err)
	}
	info.Type = schemaType
	info.Source = "embedded default"
	info.Path = "(embedded)"
	info.RawSchema = data
	return info, nil
}

// Update fetches a schema from targetURL (or default canonical URL) and caches it locally.
func (m *Manager) Update(schemaType string, targetURL string) (*SchemaInfo, error) {
	if schemaType != SchemaTypeMCP {
		schemaType = SchemaTypeManifest
	}

	if targetURL == "" {
		targetURL = CanonicalURL(schemaType)
	}

	resp, err := http.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d fetching schema from %s", resp.StatusCode, targetURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	info, err := parseSchemaMeta(body)
	if err != nil {
		return nil, fmt.Errorf("invalid schema received from %s: %w", targetURL, err)
	}

	if err := os.MkdirAll(m.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", m.CacheDir, err)
	}

	cachedPath := m.CachedPath(schemaType)
	if err := os.WriteFile(cachedPath, body, 0644); err != nil {
		return nil, fmt.Errorf("failed to write cached schema: %w", err)
	}

	info.Type = schemaType
	info.Source = "cached"
	info.Path = cachedPath
	info.RawSchema = body
	return info, nil
}

// Reset removes the cached schema for a schema type (or all if schemaType is empty).
func (m *Manager) Reset(schemaType string) error {
	if schemaType == "" {
		_ = os.Remove(m.CachedPath(SchemaTypeManifest))
		_ = os.Remove(m.CachedPath(SchemaTypeMCP))
		return nil
	}

	cachedPath := m.CachedPath(schemaType)
	if err := os.Remove(cachedPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cached schema: %w", err)
	}
	return nil
}

// Show returns information about the active schema for a schema type.
func (m *Manager) Show(schemaType string) (*SchemaInfo, error) {
	return m.Resolve(schemaType, "")
}

func (m *Manager) loadFromOverride(target string) (*SchemaInfo, error) {
	var data []byte
	var err error

	if isURL(target) {
		resp, err := http.Get(target)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schema URL %s: %w", target, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d fetching schema from %s", resp.StatusCode, target)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema body from %s: %w", target, err)
		}
	} else {
		data, err = os.ReadFile(target)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema file %s: %w", target, err)
		}
	}

	info, err := parseSchemaMeta(data)
	if err != nil {
		return nil, fmt.Errorf("invalid schema at %s: %w", target, err)
	}

	info.Source = "override"
	info.Path = target
	info.RawSchema = data
	return info, nil
}

func parseSchemaMeta(data []byte) (*SchemaInfo, error) {
	var meta struct {
		ID    string `json:"$id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &SchemaInfo{
		ID:    meta.ID,
		Title: meta.Title,
	}, nil
}

func isURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || s[:8] == "https://")
}
