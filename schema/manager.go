package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	CanonicalSchemaURL = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"
	CacheFileName      = "plugin.schema.json"
)

// SchemaInfo holds information about a resolved or loaded schema.
type SchemaInfo struct {
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

// CachedPath returns the absolute path to the cached schema file.
func (m *Manager) CachedPath() string {
	return filepath.Join(m.CacheDir, CacheFileName)
}

// Resolve resolves the active schema based on precedence:
// 1. Explicit override (file path or URL)
// 2. Cached schema in ~/.apv/schemas/plugin.schema.json
// 3. Embedded default schema
func (m *Manager) Resolve(override string) (*SchemaInfo, error) {
	if override != "" {
		return m.loadFromOverride(override)
	}

	cachedPath := m.CachedPath()
	if _, err := os.Stat(cachedPath); err == nil {
		data, err := os.ReadFile(cachedPath)
		if err == nil {
			info, parseErr := parseSchemaMeta(data)
			if parseErr == nil {
				info.Source = "cached"
				info.Path = cachedPath
				info.RawSchema = data
				return info, nil
			}
		}
	}

	// Fallback to embedded default
	data := EmbeddedSchema()
	info, err := parseSchemaMeta(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded schema: %w", err)
	}
	info.Source = "embedded default"
	info.Path = "(embedded)"
	info.RawSchema = data
	return info, nil
}

// Update fetches a schema from the given URL (or CanonicalSchemaURL if empty)
// and caches it locally.
func (m *Manager) Update(targetURL string) (*SchemaInfo, error) {
	if targetURL == "" {
		targetURL = CanonicalSchemaURL
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

	cachedPath := m.CachedPath()
	if err := os.WriteFile(cachedPath, body, 0644); err != nil {
		return nil, fmt.Errorf("failed to write cached schema: %w", err)
	}

	info.Source = "cached"
	info.Path = cachedPath
	info.RawSchema = body
	return info, nil
}

// Reset removes any cached schema, reverting to the embedded default.
func (m *Manager) Reset() error {
	cachedPath := m.CachedPath()
	if err := os.Remove(cachedPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cached schema: %w", err)
	}
	return nil
}

// Show returns information about the currently active schema (resolving without override).
func (m *Manager) Show() (*SchemaInfo, error) {
	return m.Resolve("")
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
