package schema

import (
	_ "embed"
	"encoding/json"
)

//go:embed plugin.schema.json
var embeddedSchema []byte

// EmbeddedSchema returns the raw byte content of the embedded default schema.
func EmbeddedSchema() []byte {
	return embeddedSchema
}

// EmbeddedSchemaInfo parses basic metadata ($id, title) from the embedded schema.
func EmbeddedSchemaInfo() (id string, title string) {
	var meta struct {
		ID    string `json:"$id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(embeddedSchema, &meta); err == nil {
		return meta.ID, meta.Title
	}
	return "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json", "Agent Plugins Manifest"
}
