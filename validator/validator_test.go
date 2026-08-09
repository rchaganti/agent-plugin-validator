package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rchaganti/agent-plugin-validator/schema"
)

func TestValidateManifests(t *testing.T) {
	mgr, err := schema.NewManager()
	if err != nil {
		t.Fatalf("failed to create schema manager: %v", err)
	}

	manifestSchema, err := mgr.Resolve(schema.SchemaTypeManifest, "")
	if err != nil {
		t.Fatalf("failed to resolve embedded manifest schema: %v", err)
	}

	mcpSchema, err := mgr.Resolve(schema.SchemaTypeMCP, "")
	if err != nil {
		t.Fatalf("failed to resolve embedded mcp schema: %v", err)
	}

	tests := []struct {
		name            string
		fixtureFile     string
		schemaInfo      *schema.SchemaInfo
		expectValid     bool
		expectedErrAt   string
		expectWarnCount int
	}{
		{
			name:        "Valid Minimal Manifest",
			fixtureFile: "valid_minimal.json",
			schemaInfo:  manifestSchema,
			expectValid: true,
		},
		{
			name:        "Valid Full Manifest",
			fixtureFile: "valid_full.json",
			schemaInfo:  manifestSchema,
			expectValid: true,
		},
		{
			name:          "Missing Name Field",
			fixtureFile:   "missing_name.json",
			schemaInfo:    manifestSchema,
			expectValid:   false,
			expectedErrAt: "/",
		},
		{
			name:          "Uppercase Name Violation",
			fixtureFile:   "invalid_name_uppercase.json",
			schemaInfo:    manifestSchema,
			expectValid:   false,
			expectedErrAt: "/name",
		},
		{
			name:            "Unknown Top Level Field (Warning per Spec §5.2)",
			fixtureFile:     "unknown_fields.json",
			schemaInfo:      manifestSchema,
			expectValid:     true,
			expectWarnCount: 1,
		},
		{
			name:        "Valid MCP Configuration",
			fixtureFile: "valid_mcp.json",
			schemaInfo:  mcpSchema,
			expectValid: true,
		},
		{
			name:          "Invalid MCP Configuration (stdio missing command)",
			fixtureFile:   "invalid_mcp.json",
			schemaInfo:    mcpSchema,
			expectValid:   false,
			expectedErrAt: "/mcpServers/bad-tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "testdata", tt.fixtureFile)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read fixture %s: %v", path, err)
			}

			result, err := Validate(data, tt.schemaInfo)
			if err != nil {
				t.Fatalf("validation error: %v", err)
			}

			if result.Valid != tt.expectValid {
				t.Errorf("expected valid=%v, got valid=%v. Errors: %+v, Warnings: %+v",
					tt.expectValid, result.Valid, result.Errors, result.Warnings)
			}

			if !tt.expectValid && tt.expectedErrAt != "" {
				found := false
				for _, issue := range result.Errors {
					if issue.Path == tt.expectedErrAt {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error at path %q, got errors: %+v", tt.expectedErrAt, result.Errors)
				}
			}

			if tt.expectWarnCount > 0 && len(result.Warnings) != tt.expectWarnCount {
				t.Errorf("expected %d warnings, got %d: %+v", tt.expectWarnCount, len(result.Warnings), result.Warnings)
			}
		})
	}
}
