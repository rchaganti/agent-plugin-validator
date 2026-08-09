package schema

import (
	_ "embed"
	"encoding/json"
)

//go:embed plugin.schema.json
var embeddedPluginSchema []byte

//go:embed mcp.schema.json
var embeddedMCPSchema []byte

// EmbeddedSchema returns the raw byte content for the given schema type ("manifest" or "mcp").
func EmbeddedSchema(schemaType string) []byte {
	if schemaType == SchemaTypeMCP {
		return embeddedMCPSchema
	}
	return embeddedPluginSchema
}

// EmbeddedSchemaInfo parses basic metadata ($id, title) from the embedded schema.
func EmbeddedSchemaInfo(schemaType string) (id string, title string) {
	data := EmbeddedSchema(schemaType)
	var meta struct {
		ID    string `json:"$id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(data, &meta); err == nil {
		return meta.ID, meta.Title
	}
	if schemaType == SchemaTypeMCP {
		return CanonicalMCPSchemaURL, "Agent Plugins MCP Configuration"
	}
	return CanonicalPluginSchemaURL, "Agent Plugins Manifest"
}
