package openai

import (
	"maps"
	"slices"

	"github.com/openai/openai-go/v3/shared"

	"github.com/docker/cagent/pkg/tools"
)

// ConvertParametersToSchema converts parameters to OpenAI Schema format
func ConvertParametersToSchema(params any) (shared.FunctionParameters, error) {
	p, err := tools.SchemaToMap(params)
	if err != nil {
		return nil, err
	}

	return fixSchemaArrayItems(makeAllRequired(p)), nil
}

// makeAllRequired make all the parameters "required"
// because that's what the Response API wants, now.
func makeAllRequired(schema shared.FunctionParameters) shared.FunctionParameters {
	if schema == nil {
		return makeAllRequired(map[string]any{"type": "object", "properties": map[string]any{}})
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return schema
	}

	reallyRequired := map[string]bool{}
	if required, ok := schema["required"].([]any); ok {
		for _, name := range required {
			reallyRequired[name.(string)] = true
		}
	}

	// We can't use a nil 'required' attribute
	newRequired := []any{}

	// Sort property names for deterministic output
	propNames := slices.Sorted(maps.Keys(properties))

	for _, propName := range propNames {
		newRequired = append(newRequired, propName)
		if reallyRequired[propName] {
			continue
		}

		// Make its type nullable
		if propMap, ok := properties[propName].(map[string]any); ok {
			if typeValue, ok := propMap["type"].(string); ok {
				propMap["type"] = []string{typeValue, "null"}
			}
		}
	}

	schema["required"] = newRequired
	schema["additionalProperties"] = false
	return schema
}

// In Docker Desktop 4.52, the MCP Gateway produces an invalid tools shema for `mcp-config-set`.
func fixSchemaArrayItems(schema shared.FunctionParameters) shared.FunctionParameters {
	propertiesValue, ok := schema["properties"]
	if !ok {
		return schema
	}

	properties, ok := propertiesValue.(map[string]any)
	if !ok {
		return schema
	}

	for _, propValue := range properties {
		prop, ok := propValue.(map[string]any)
		if !ok {
			continue
		}

		checkForMissingItems := false
		switch t := prop["type"].(type) {
		case string:
			checkForMissingItems = t == "array"
		case []string:
			checkForMissingItems = slices.Contains(t, "array")
		}
		if !checkForMissingItems {
			continue
		}

		if _, ok := prop["items"]; !ok {
			prop["items"] = map[string]any{"type": "object"}
		}
	}

	return schema
}
