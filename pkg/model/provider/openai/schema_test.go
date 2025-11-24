package openai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/tools"
)

func TestMakeAllRequired(t *testing.T) {
	type DirectoryTreeArgs struct {
		Path     string `json:"path" jsonschema:"The directory path to traverse"`
		MaxDepth int    `json:"max_depth,omitempty" jsonschema:"Maximum depth to traverse (optional)"`
	}
	schema := tools.MustSchemaFor[DirectoryTreeArgs]()

	schemaMap, err := tools.SchemaToMap(schema)
	require.NoError(t, err)
	required := schemaMap["required"].([]any)
	assert.Len(t, required, 1)
	assert.Contains(t, required, "path")

	updatedSchema := makeAllRequired(schemaMap)
	required = updatedSchema["required"].([]any)
	assert.Len(t, required, 2)
	assert.Contains(t, required, "max_depth")
	assert.Contains(t, required, "path")
}

func TestMakeAllRequired_NoParameter(t *testing.T) {
	type NoArgs struct{}
	schema := tools.MustSchemaFor[NoArgs]()

	schemaMap, err := tools.SchemaToMap(schema)
	require.NoError(t, err)

	buf, err := json.Marshal(schemaMap)
	require.NoError(t, err)
	assert.JSONEq(t, `{"additionalProperties":false,"properties":{},"type":"object"}`, string(buf))

	updatedSchema := makeAllRequired(schemaMap)
	buf, err = json.Marshal(updatedSchema)
	require.NoError(t, err)
	assert.JSONEq(t, `{"additionalProperties":false,"properties":{},"type":"object","required":[]}`, string(buf))
}

func TestMakeAllRequired_NilSchema(t *testing.T) {
	updatedSchema := makeAllRequired(nil)
	buf, err := json.Marshal(updatedSchema)
	require.NoError(t, err)
	assert.JSONEq(t, `{"additionalProperties":false,"properties":{},"type":"object","required":[]}`, string(buf))
}

func TestFixSchemaArrayItems(t *testing.T) {
	schema := `{
  "properties": {
    "arguments": {
      "description": "Arguments to pass to the tool (can be any valid JSON value)",
      "type": [
        "string",
        "number",
        "boolean",
        "object",
        "array",
        "null"
      ]
    },
    "name": {
      "description": "Name of the tool to execute",
      "type": "string"
    }
  },
  "required": [
    "name"
  ],
  "type": "object"
}`

	schemaMap := map[string]any{}
	err := json.Unmarshal([]byte(schema), &schemaMap)
	require.NoError(t, err)

	updatedSchema := fixSchemaArrayItems(schemaMap)
	buf, err := json.Marshal(updatedSchema)
	require.NoError(t, err)

	assert.JSONEq(t, `{
  "properties": {
    "arguments": {
      "description": "Arguments to pass to the tool (can be any valid JSON value)",
      "type": [
        "string",
        "number",
        "boolean",
        "object",
        "array",
        "null"
      ]
    },
    "name": {
      "description": "Name of the tool to execute",
      "type": "string"
    }
  },
  "required": [
    "name"
  ],
  "type": "object"
}`, string(buf))
}
