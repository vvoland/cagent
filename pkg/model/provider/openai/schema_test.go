package openai

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3/shared"
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

func TestRemoveFormatFields(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"format":      "uri",
				"description": "The URL",
			},
			"email": map[string]any{
				"type":   "string",
				"format": "email",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "The name",
			},
		},
	}

	updated := removeFormatFields(schema)

	url := updated["properties"].(map[string]any)["url"].(map[string]any)
	assert.Equal(t, "string", url["type"])
	assert.Equal(t, "The URL", url["description"])
	assert.NotContains(t, url, "format")

	email := updated["properties"].(map[string]any)["email"].(map[string]any)
	assert.Equal(t, "string", email["type"])
	assert.NotContains(t, email, "format")

	name := updated["properties"].(map[string]any)["name"].(map[string]any)
	assert.Equal(t, "string", name["type"])
	assert.Equal(t, "The name", name["description"])
}

func TestRemoveFormatFields_NestedObjects(t *testing.T) {
	schema := shared.FunctionParameters{
		"type": "object",
		"properties": map[string]any{
			"user": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"email": map[string]any{
						"type":   "string",
						"format": "email",
					},
					"website": map[string]any{
						"type":   "string",
						"format": "uri",
					},
				},
			},
			"name": map[string]any{
				"type":   "string",
				"format": "hostname",
			},
		},
	}

	updated := removeFormatFields(schema)

	user := updated["properties"].(map[string]any)["user"].(map[string]any)
	email := user["properties"].(map[string]any)["email"].(map[string]any)
	assert.NotContains(t, email, "format")
	assert.Equal(t, "string", email["type"])

	website := user["properties"].(map[string]any)["website"].(map[string]any)
	assert.NotContains(t, website, "format")

	name := updated["properties"].(map[string]any)["name"].(map[string]any)
	assert.NotContains(t, name, "format")
}

func TestRemoveFormatFields_ArrayItems(t *testing.T) {
	schema := shared.FunctionParameters{
		"type": "object",
		"properties": map[string]any{
			"urls": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":   "string",
					"format": "uri",
				},
			},
			"contacts": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"email": map[string]any{
							"type":   "string",
							"format": "email",
						},
					},
				},
			},
		},
	}

	updated := removeFormatFields(schema)

	urls := updated["properties"].(map[string]any)["urls"].(map[string]any)
	urlItems := urls["items"].(map[string]any)
	assert.NotContains(t, urlItems, "format")
	assert.Equal(t, "string", urlItems["type"])

	contacts := updated["properties"].(map[string]any)["contacts"].(map[string]any)
	contactItems := contacts["items"].(map[string]any)
	email := contactItems["properties"].(map[string]any)["email"].(map[string]any)
	assert.NotContains(t, email, "format")
	assert.Equal(t, "string", email["type"])
}

func TestRemoveFormatFields_NilSchema(t *testing.T) {
	assert.Nil(t, removeFormatFields(nil))
}

func TestRemoveFormatFields_NoProperties(t *testing.T) {
	schema := shared.FunctionParameters{"type": "object"}
	updated := removeFormatFields(schema)
	assert.Equal(t, schema, updated)
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
